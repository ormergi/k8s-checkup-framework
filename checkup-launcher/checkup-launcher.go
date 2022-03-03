package main

import (
	"context"
	"fmt"
	"log"
	"os"

	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/orelmisan/k8s-checkup-framework/checkup-launcher/checkup"
)

func main() {
	cmNamespace, cmName, err := getConfigMapFullNameFromEnv()
	if err != nil {
		log.Fatalf("Failed to get configMap full name: %v\n", err.Error())
	}

	clientset, err := createK8sClientSet()
	if err != nil {
		log.Fatalf("Failed to create K8s clientset: %v\n", err.Error())
	}

	configMap, err := getConfigMap(clientset, cmNamespace, cmName)
	if err != nil {
		log.Fatalf("Failed to get ConfigMap: %v\n", err.Error())
	}

	checkupSpec, err := checkup.NewSpecFromConfigMap(configMap)
	if err != nil {
		log.Fatalf("Failed to create checkup spec: %v\n", err.Error())
	}

	logCheckupSpec(checkupSpec)
}

func createK8sClientSet() (*kubernetes.Clientset, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return clientset, nil
}

func getConfigMapFullNameFromEnv() (namespace, name string, err error) {
	const (
		ConfigMapNamespaceEnvVarName = "CONFIGMAP_NAMESPACE"
		ConfigMapNameEnvVarName      = "CONFIGMAP_NAME"
	)

	var exists bool

	namespace, exists = os.LookupEnv(ConfigMapNamespaceEnvVarName)
	if !exists {
		return "", "", fmt.Errorf("failed to read %s", ConfigMapNamespaceEnvVarName)
	}

	name, exists = os.LookupEnv(ConfigMapNameEnvVarName)
	if !exists {
		return "", "", fmt.Errorf("failed to read %s", ConfigMapNameEnvVarName)
	}

	return namespace, name, nil
}

func getConfigMap(clientset *kubernetes.Clientset, namespace, name string) (*corev1.ConfigMap, error) {
	configMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return configMap, nil
}

func logCheckupSpec(spec *checkup.Spec) {
	log.Println("Checkup spec:")
	log.Printf("\tImage: %q\n", spec.Image())
	log.Printf("\tTimeout: %q\n", spec.Timeout())

	log.Printf("\tParams:\n")
	for k, v := range spec.Params() {
		log.Printf("\t\t%q: %q\n", k, v)
	}

	log.Printf("\tClusterRoles:\n")
	for _, clusterRole := range spec.ClusterRoles() {
		log.Printf("\t\t%q\n", clusterRole)
	}

	log.Printf("\tRoles:\n")
	for _, role := range spec.Roles() {
		log.Printf("\t\t%q\n", role)
	}
}
