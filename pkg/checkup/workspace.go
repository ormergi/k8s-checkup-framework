package checkup

import (
	"context"
	"log"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	namespacePrefix    = "checkup-"
	configMapPrefix    = "checkup-results-"
	serviceAccountName = "checkup-sa"
)

type Workspace struct {
	clientset            *kubernetes.Clientset
	spec                 *Spec
	namespace            string
	serviceAccountName   string
	resultsConfigMapName string
}

func NewWorkspace(clientset *kubernetes.Clientset, spec *Spec) *Workspace {
	return &Workspace{
		clientset: clientset,
		spec:      spec,
	}
}

func (w *Workspace) Setup() error {
	if err := w.createNamespace(); err != nil {
		return err
	}

	if err := w.createServiceAccount(); err != nil {
		return err
	}

	if err := w.createResultsConfigMap(); err != nil {
		return err
	}

	return nil
}

func (w *Workspace) Teardown() error {
	if err := w.deleteNamespace(); err != nil {
		return err
	}

	return nil
}

func (w *Workspace) createNamespace() error {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: namespacePrefix,
		},
	}

	createdNamespace, err := w.clientset.CoreV1().Namespaces().Create(context.Background(), namespace, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	w.namespace = createdNamespace.Name
	log.Printf("Successfuly created namesapce: %q\n", w.namespace)

	return nil
}

func (w *Workspace) deleteNamespace() error {
	if err := w.clientset.CoreV1().Namespaces().Delete(context.Background(), w.namespace, metav1.DeleteOptions{}); err != nil {
		return err
	}

	log.Printf("Successfuly deleted namesapce: %q\n", w.namespace)
	w.namespace = ""

	return nil
}

func (w *Workspace) createServiceAccount() error {
	namespace := w.namespace
	name := serviceAccountName

	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	if _, err := w.clientset.CoreV1().ServiceAccounts(namespace).Create(context.Background(), serviceAccount, metav1.CreateOptions{}); err != nil {
		return err
	}

	log.Printf("Successfuly created ServiceAccount: %s/%s\n", namespace, name)
	w.serviceAccountName = name

	return nil
}

func (w *Workspace) createResultsConfigMap() error {
	namespace := w.namespace

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: configMapPrefix,
			Namespace:    namespace,
		},
	}

	createdConfigMap, err := w.clientset.CoreV1().ConfigMaps(namespace).Create(context.Background(), configMap, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	name := createdConfigMap.Name

	log.Printf("Successfuly created ConfigMap: %s/%s\n", namespace, name)
	w.resultsConfigMapName = name

	return nil
}
