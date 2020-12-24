package ingress

import (
	"context"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	ingressControllerSVCName      = "ingress-nginx-shared-controller"
	ingressControllerSVCNamesapce = "ingress-nginx-shared"
)

var logger logr.Logger = logf.Log.WithName("ingress controller")

func GetIngressControllerSVC() (*corev1.Service, error) {
	c, err := client.New(config.GetConfigOrDie(), client.Options{})
	if err != nil {
		return nil, err
	}

	svc := &corev1.Service{}
	err = c.Get(context.TODO(), types.NamespacedName{Name: ingressControllerSVCName, Namespace: ingressControllerSVCNamesapce}, svc)
	if err != nil {
		return nil, err
	}

	return svc, nil
}

func GetIngressControllerIP() string {
	svc, err := GetIngressControllerSVC()
	if err != nil {
		logger.Error(err, "there is no ingress controller service", "service name", ingressControllerSVCName, "service namespace", ingressControllerSVCNamesapce)
		return ""
	}

	if len(svc.Status.LoadBalancer.Ingress) > 0 {
		return svc.Status.LoadBalancer.Ingress[0].IP
	}

	return ""
}
