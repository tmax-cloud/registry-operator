package regctl

import (
	"context"
	"github.com/go-logr/logr"
	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	"k8s.io/apimachinery/pkg/types"

	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/operator-framework/operator-lib/status"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// RegistryService things to handle service resource
type RegistryService struct {
	c            client.Client
	cond         status.ConditionType
	requirements []status.ConditionType
	manifest     func() (interface{}, error)
	logger       logr.Logger
}

// NewRegistryService creates new registry service
func NewRegistryService(client client.Client, manifest func() (interface{}, error), cond status.ConditionType, logger logr.Logger) *RegistryService {

	return &RegistryService{
		c:        client,
		manifest: manifest,
		cond:     cond,
		logger:   logger.WithName("Service"),
	}
}

// Ready checks that service is ready
func (r *RegistryService) ReconcileByConditionStatus(reg *regv1.Registry) (bool, error) {
	logger := r.logger.WithName("ReconcileByConditionStatus")
	var err error
	defer func() {
		if err != nil {
			reg.Status.Conditions.SetCondition(
				status.Condition{
					Type:    r.cond,
					Status:  corev1.ConditionFalse,
					Message: err.Error(),
				})
		}
	}()

	for _, dep := range r.requirements {
		if !reg.Status.Conditions.GetCondition(dep).IsTrue() {
			r.logger.Info(string(r.cond) + " needs " + string(dep))
			return true, nil
		}
	}

	ctx := context.TODO()
	m, err := r.manifest()
	if err != nil {
		return false, err
	}
	manifest := m.(*corev1.Service)
	svc := &corev1.Service{}
	if err = r.c.Get(ctx, types.NamespacedName{Name: manifest.Name, Namespace: manifest.Namespace}, svc); err != nil {
		if errors.IsNotFound(err) {
			r.logger.Info("not found. create new one.")
			if err = r.c.Create(ctx, manifest); err != nil {
				return true, err
			}
		}
		return true, err
	}

	switch svc.Spec.Type {
	case corev1.ServiceTypeLoadBalancer:
		loadBalancer := svc.Status.LoadBalancer
		lbIP := ""
		// [TODO] Specific Condition is needed
		if len(loadBalancer.Ingress) == 1 {
			if loadBalancer.Ingress[0].Hostname == "" {
				lbIP = loadBalancer.Ingress[0].IP
			} else {
				lbIP = loadBalancer.Ingress[0].Hostname
			}
		} else if len(loadBalancer.Ingress) == 0 {
			// Several Ingress
			err = regv1.MakeRegistryError("NotReady")
			return true, err
		}
		reg.Status.LoadBalancerIP = lbIP
		reg.Status.ServerURL = "https://" + lbIP
		logger.Info("LoadBalancer info", "LoadBalancer IP", lbIP)
	case corev1.ServiceTypeClusterIP:
		if svc.Spec.ClusterIP == "" {
			err = regv1.MakeRegistryError("NotReady")
			return true, err
		}
		logger.Info("Service Type is ClusterIP(Ingress)")
		// [TODO]
	}

	reg.Status.ClusterIP = svc.Spec.ClusterIP
	reg.Status.Conditions.SetCondition(
		status.Condition{
			Type:    r.cond,
			Status:  corev1.ConditionTrue,
			Message: "Success",
		})
	r.logger.Info("fine")
	return false, nil
}

func (r *RegistryService) Require(cond status.ConditionType) ResourceController {
	r.requirements = append(r.requirements, cond)
	return r
}
