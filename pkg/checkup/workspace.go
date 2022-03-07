package checkup

import (
	"context"
	"log"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type Workspace struct {
	clientset *kubernetes.Clientset
	spec      *Spec
	namespace string
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

	return nil
}

func (w *Workspace) Teardown() error {
	if err := w.deleteNamespace(); err != nil {
		return err
	}

	return nil
}

func (w *Workspace) createNamespace() error {
	const namespacePrefix = "checkup-"

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
