package regctl

import (
	"context"

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"

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
	label["apps"] = regv1.K8sPrefix + reg.Name

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

func PodName(c client.Client, reg *regv1.Registry) (string, error) {
	pod, err := getPod(c, reg)
	if err != nil {
		log.Error(err, "Pod error")
		return "", err
	}

	return pod.Name, nil
}

func DeletePod(c client.Client, reg *regv1.Registry) error {
	pod, err := getPod(c, reg)
	if err != nil {
		log.Error(err, "Pod error")
		return err
	}

	if err := c.Delete(context.TODO(), pod); err != nil {
		log.Error(err, "Unknown error delete pod")
		return err
	}

	return nil
}
