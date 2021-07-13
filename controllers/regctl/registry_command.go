package regctl

import (
	"context"

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/internal/schemes"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("controller_registry")

func getPod(c client.Client, reg *regv1.Registry) (*corev1.Pod, error) {
	podList := &corev1.PodList{}
	label := map[string]string{}
	label["app"] = "registry"
	label["apps"] = schemes.SubresourceName(reg, schemes.SubTypeRegistryDeployment)

	labelSelector := labels.SelectorFromSet(labels.Set(label))
	listOps := &client.ListOptions{
		Namespace:     reg.Namespace,
		LabelSelector: labelSelector,
	}
	err := c.List(context.TODO(), podList, listOps)
	if err != nil {
		log.Error(err, "Failed to list pods.")
		return nil, err
	}

	if len(podList.Items) == 0 {
		return nil, regv1.MakeRegistryError(regv1.PodNotFound)
	}

	pod := &podList.Items[0]

	return pod, nil
}

// PodName returns registry pod name
func PodName(c client.Client, reg *regv1.Registry) (string, error) {
	pod, err := getPod(c, reg)
	if err != nil {
		log.Error(err, "Pod error")
		return "", err
	}

	return pod.Name, nil
}
