package main

import (
	"context"
	"fmt"
	"log"
	"os"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/orelmisan/k8s-checkup-framework/pkg/checkup"
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

	workspace := checkup.NewCheckupWorkspace(checkupSpec)
	if err := workspace.SetupCheckupWorkspace(clientset); err != nil {
		log.Fatalf("Failed to setup the checkup's environment: %v", err)
	}
	jobErr := workspace.StartAndWaitCheckupJob(clientset)

	checkupJob := workspace.Job()
	if err := logCheckupJobLogs(clientset, checkupJob.Namespace); err != nil {
		log.Printf("Failed to dump checkup job logs: %v", err)
	}

	if jobErr != nil {
		log.Fatalf("Error occured while running checkup job: %v", jobErr)
	}
	if isJobFailed(checkupJob) {
		log.Fatalf("Checkup job completed with failure")
	}
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

func logCheckupJobLogs(client *kubernetes.Clientset, jobNamespace string) error {
	podList, err := client.CoreV1().Pods(jobNamespace).List(context.Background(), metav1.ListOptions{LabelSelector: checkup.JobNameLabel})
	if err != nil {
		return err
	}
	if len(podList.Items) < 1 {
		return fmt.Errorf("no checkup job underlaying pods were found")
	}
	checkupJobPod := podList.Items[0]
	rawLogs, err := client.CoreV1().Pods(checkupJobPod.Namespace).GetLogs(checkupJobPod.Name, &corev1.PodLogOptions{}).
		DoRaw(context.Background())

	log.Printf("Checkup logs:\n%s", string(rawLogs))

	return nil
}

func isJobFailed(job *batchv1.Job) bool {
	for _, condition := range job.Status.Conditions {
		if condition.Type == batchv1.JobFailed {
			return true
		}
	}
	return false
}
