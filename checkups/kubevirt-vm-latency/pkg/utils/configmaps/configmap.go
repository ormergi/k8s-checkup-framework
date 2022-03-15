package configmaps

import (
	"context"
	"log"

	k8scorev1 "k8s.io/api/core/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kubevirt.io/client-go/kubecli"
)

func Get(client kubecli.KubevirtClient, namespace, name string) (*k8scorev1.ConfigMap, error) {
	return client.CoreV1().ConfigMaps(namespace).Get(context.Background(), name, k8smetav1.GetOptions{})
}

func AppendData(virtClient kubecli.KubevirtClient, namespace, name string, data map[string]string) error {
	configMap, err := Get(virtClient, namespace, name)
	if err != nil {
		return err
	}

	if configMap.Data == nil {
		configMap.Data = map[string]string{}
	}
	for k, v := range data {
		configMap.Data[k] = v
	}

	log.Printf("updating ConfigMap %s/%s..", configMap.Namespace, configMap.Name)
	if _, err := update(virtClient, configMap); err != nil {
		return err
	}

	return nil
}

func update(client kubecli.KubevirtClient, cm *k8scorev1.ConfigMap) (*k8scorev1.ConfigMap, error) {
	return client.CoreV1().ConfigMaps(cm.Namespace).
		Update(context.Background(), cm, k8smetav1.UpdateOptions{})
}
