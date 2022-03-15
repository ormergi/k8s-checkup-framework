package checkup

import (
	"context"
	"fmt"
	"k8s.io/client-go/kubernetes"
	"log"
	"strings"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	checkupNamespaceName           = "checkup-workspace"
	resultsConfigMapName           = "checkup-results"
	checkupJobName                 = "checkup-job"
	serviceAccountName             = "checkup-sa"
	resultsConfigMapWriterRoleName = "results-configmap-writer"
)

const (
	resultsConfigMapNameEnvVarName      = "RESULT_CONFIGMAP_NAME"
	resultsConfigMapNameEnvVarNamespace = "RESULT_CONFIGMAP_NAMESPACE"
)

const (
	ClusterRoleKind = "ClusterRole"
	RoleKind        = "Role"
)

type workspace struct {
	namespace           *corev1.Namespace
	serviceAccount      *corev1.ServiceAccount
	resultConfigMap     *corev1.ConfigMap
	roles               map[string]rbacv1.Role
	roleBindings        map[string]rbacv1.RoleBinding
	clusterRoles        map[string]rbacv1.ClusterRole
	clusterRoleBindings map[string]rbacv1.ClusterRoleBinding
	envVars             []corev1.EnvVar
	resources           corev1.ResourceList
	checkupTimeout      time.Duration
	job                 *batchv1.Job
}

func (w *workspace) Job() *batchv1.Job {
	return w.job
}

func NewCheckupWorkspace(checkupSpec *Spec, clusterRoles []rbacv1.ClusterRole) *workspace {
	workspace := &workspace{}

	workspace.checkupTimeout = checkupSpec.timeout
	workspace.namespace = newNamespace(checkupNamespaceName)
	workspace.serviceAccount = newServiceAccount(serviceAccountName, checkupNamespaceName)
	workspace.resultConfigMap = newConfigMap(resultsConfigMapName, checkupNamespaceName)

	workspace.clusterRoles = map[string]rbacv1.ClusterRole{}
	for _, clusterRole := range clusterRoles {
		workspace.clusterRoles[clusterRole.Name] = clusterRole
	}

	subject := rbacv1.Subject{Kind: rbacv1.ServiceAccountKind, Name: serviceAccountName, Namespace: checkupNamespaceName}

	resultsWriterRole := newConfigMapWriterRole(
		resultsConfigMapWriterRoleName,
		checkupNamespaceName,
		workspace.resultConfigMap.Name,
	)
	resultsWriterRoleBinding := newRoleBindingForSubject(resultsWriterRole, subject)

	workspace.roles = map[string]rbacv1.Role{
		resultsWriterRole.Name: resultsWriterRole,
	}

	workspace.roleBindings = map[string]rbacv1.RoleBinding{
		resultsWriterRoleBinding.Name: resultsWriterRoleBinding,
	}

	workspace.clusterRoleBindings = make(map[string]rbacv1.ClusterRoleBinding, len(workspace.clusterRoles))
	for _, clusterRole := range clusterRoles {
		workspace.clusterRoleBindings[clusterRole.Name] = newClusterRoleBindingForSubject(clusterRole, subject)
	}

	checkupEnvVars := createEnvVarsFromSpec(checkupSpec.Params())
	checkupEnvVars = append(checkupEnvVars,
		corev1.EnvVar{Name: resultsConfigMapNameEnvVarName, Value: workspace.resultConfigMap.Name})
	checkupEnvVars = append(checkupEnvVars,
		corev1.EnvVar{Name: resultsConfigMapNameEnvVarNamespace, Value: workspace.resultConfigMap.Namespace})
	workspace.envVars = checkupEnvVars

	checkupContainerMemory := resource.NewScaledQuantity(100, resource.Mega)
	checkupContainerCpu := resource.NewQuantity(1, resource.DecimalSI)
	workspace.resources = corev1.ResourceList{
		corev1.ResourceMemory: *checkupContainerMemory,
		corev1.ResourceCPU:    *checkupContainerCpu,
	}

	workspace.job = newCheckupJob(checkupSpec.Image(),
		workspace.envVars,
		workspace.resources,
		checkupJobName,
		workspace.checkupTimeout)

	return workspace
}

func newNamespace(name string) *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}

func newServiceAccount(name, namespace string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

func newConfigMap(name, namespace string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

func newConfigMapWriterRole(name, namespace, configMapName string) rbacv1.Role {
	return rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Rules: []rbacv1.PolicyRule{{
			Verbs:         []string{"get", "update", "patch"},
			APIGroups:     []string{""},
			Resources:     []string{"configmaps"},
			ResourceNames: []string{configMapName},
		}},
	}
}

func newRoleBindingForSubject(role rbacv1.Role, subject rbacv1.Subject) rbacv1.RoleBinding {
	return rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s-to-sa", role.Name)},
		Subjects:   []rbacv1.Subject{subject},
		RoleRef:    rbacv1.RoleRef{APIGroup: rbacv1.GroupName, Kind: RoleKind, Name: role.Name},
	}
}

func newClusterRoleBindingForSubject(clusterRole rbacv1.ClusterRole, subject rbacv1.Subject) rbacv1.ClusterRoleBinding {
	return rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s-to-sa", clusterRole.Name)},
		Subjects:   []rbacv1.Subject{subject},
		RoleRef:    rbacv1.RoleRef{APIGroup: rbacv1.GroupName, Kind: ClusterRoleKind, Name: clusterRole.Name},
	}
}

func createEnvVarsFromSpec(parametersMap map[string]string) []corev1.EnvVar {
	var checkupEnvVars []corev1.EnvVar
	if len(parametersMap) > 0 {
		for k, v := range parametersMap {
			checkupEnvVars = append(checkupEnvVars, corev1.EnvVar{Name: strings.ToUpper(k), Value: v})
		}
	}
	return checkupEnvVars
}

func newCheckupJob(image string, envs []corev1.EnvVar, resources corev1.ResourceList, name string, timeout time.Duration) *batchv1.Job {
	checkupContainer := newCheckupJobContainer(image, envs, resources)
	terminationGracePeriodSeconds := int64(5)
	checkupPodSpec := newPodTemplateSpec(&terminationGracePeriodSeconds, []corev1.Container{checkupContainer})
	backoffLimit := int32(0)
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: batchv1.JobSpec{
			BackoffLimit: &backoffLimit,
			Template:     checkupPodSpec,
		},
	}
}

func newCheckupJobContainer(image string, envs []corev1.EnvVar, resources corev1.ResourceList) corev1.Container {
	return corev1.Container{
		Name:  "checkup",
		Image: image,
		Env:   envs,
		Resources: corev1.ResourceRequirements{
			Limits:   resources,
			Requests: resources,
		},
		ImagePullPolicy: corev1.PullAlways,
	}
}

func newPodTemplateSpec(terminationGracePeriodSeconds *int64, containers []corev1.Container) corev1.PodTemplateSpec {
	return corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{},
		Spec: corev1.PodSpec{
			ServiceAccountName:            serviceAccountName,
			RestartPolicy:                 corev1.RestartPolicyNever,
			TerminationGracePeriodSeconds: terminationGracePeriodSeconds,
			Containers:                    containers,
		},
	}
}

func (w *workspace) SetupCheckupWorkspace(client *kubernetes.Clientset) error {
	const errMsgPrefix = "failed to create checkup environment"

	if err := w.createNamespace(client); err != nil {
		return fmt.Errorf("%s: %v", errMsgPrefix, err)
	}

	if err := w.createServiceAccount(client); err != nil {
		return fmt.Errorf("%s: %v", errMsgPrefix, err)
	}

	if err := w.createResultsConfigMap(client); err != nil {
		return fmt.Errorf("%s: %v", errMsgPrefix, err)
	}

	if err := w.createResultsConfigMapWriterRole(client); err != nil {
		return fmt.Errorf("%s: %v", errMsgPrefix, err)
	}

	if err := w.createRoleBindings(client); err != nil {
		return fmt.Errorf("%s: %v", errMsgPrefix, err)
	}

	if err := w.createClusterRoleBindings(client); err != nil {
		return fmt.Errorf("%s: %v", errMsgPrefix, err)
	}

	return nil
}

func (w *workspace) createNamespace(client *kubernetes.Clientset) error {
	var err error

	w.namespace, err = client.CoreV1().Namespaces().Create(context.Background(), w.namespace, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	log.Printf("checkup namespace '%s' sucessfully created", w.namespace.Name)

	return nil
}

func (w *workspace) createServiceAccount(client *kubernetes.Clientset) error {
	var err error

	w.serviceAccount, err = client.CoreV1().ServiceAccounts(w.namespace.Name).
		Create(context.Background(), w.serviceAccount, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	log.Printf("checkup serviceAccount '%s/%s' sucessfully created",
		w.serviceAccount.Namespace, w.serviceAccount.Name)

	return nil
}

func (w *workspace) createResultsConfigMap(client *kubernetes.Clientset) error {
	var err error

	w.resultConfigMap, err = client.CoreV1().ConfigMaps(w.namespace.Name).
		Create(context.Background(), w.resultConfigMap, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	log.Printf("checkup results ConfigMap '%s/%s' sucessfully created",
		w.resultConfigMap.Namespace, w.resultConfigMap.Name)

	return nil
}

func (w *workspace) createResultsConfigMapWriterRole(client *kubernetes.Clientset) error {
	var err error

	resultsConfigMapWriterRole := w.roles[resultsConfigMapWriterRoleName]
	resultsConfigMapWriterRoleRef := &resultsConfigMapWriterRole
	resultsConfigMapWriterRoleRef, err = client.RbacV1().Roles(w.namespace.Name).
		Create(context.Background(), resultsConfigMapWriterRoleRef, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create results ConfigMap writer Role: %v", err)
	}
	log.Printf("checkup results ConfigMap writer Role '%s/%s' sucessfully created",
		resultsConfigMapWriterRoleRef.Namespace, resultsConfigMapWriterRoleRef.Name)

	w.roles[resultsConfigMapWriterRoleName] = *resultsConfigMapWriterRoleRef

	return nil
}

func (w *workspace) createClusterRoleBindings(client *kubernetes.Clientset) error {
	createdClusterRoleBindings := map[string]rbacv1.ClusterRoleBinding{}
	for k, v := range w.clusterRoleBindings {
		binding, err := client.RbacV1().ClusterRoleBindings().Create(context.Background(), &v, metav1.CreateOptions{})
		if err != nil {
			return err
		}
		log.Printf("ClusterRoleBinding for ClusterRole %s sucessfully created", k)
		createdClusterRoleBindings[k] = *binding
	}

	w.clusterRoleBindings = createdClusterRoleBindings

	return nil
}

func (w *workspace) createRoleBindings(client *kubernetes.Clientset) error {
	createdRoleBindings := map[string]rbacv1.RoleBinding{}
	for k, v := range w.roleBindings {
		binding, err := client.RbacV1().RoleBindings(w.namespace.Name).Create(context.Background(), &v, metav1.CreateOptions{})
		if err != nil {
			return err
		}
		log.Printf("RoleBinding for Role %s sucessfully created", k)
		createdRoleBindings[k] = *binding
	}

	w.roleBindings = createdRoleBindings

	return nil
}

func (w *workspace) StartAndWaitCheckupJob(client *kubernetes.Clientset) error {
	var err error
	createdJob, err := client.BatchV1().Jobs(w.namespace.Name).Create(context.Background(), w.job, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to start checkup job: %v", err)
	}
	w.job = createdJob
	log.Printf("checkup job %s sucessfully created", w.job.Name)

	completedJob, err := WaitForJobToComplete(client, w.job.Name, w.job.Namespace, w.checkupTimeout)
	if err != nil {
		return fmt.Errorf("failed to wait for checkup job to complete: %v", err)
	}
	w.job = completedJob
	log.Printf("checkup job %s completed", w.job.Name)

	return nil
}

func (w *workspace) Teardown(client *kubernetes.Clientset) error {
	var teardownErrors []error

	if err := w.deleteNamespace(client); err != nil {
		teardownErrors = append(teardownErrors, err)
	}

	if err := w.deleteClusterRoleBindings(client); err != nil {
		teardownErrors = append(teardownErrors, err)
	}

	if len(teardownErrors) > 0 {
		return concateErrors(teardownErrors)
	}

	return nil
}

func (w *workspace) deleteNamespace(client *kubernetes.Clientset) error {
	if err := client.CoreV1().Namespaces().Delete(context.Background(), w.namespace.Name, metav1.DeleteOptions{}); err != nil {
		return err
	}

	log.Printf("Successfuly deleted namesapce: %q\n", w.namespace.Name)
	w.namespace = nil

	return nil
}

func (w *workspace) deleteClusterRoleBindings(client *kubernetes.Clientset) error {
	clusterRoleBindingDeleteErrors := []error{}
	for _, clusterRoleBinding := range w.clusterRoleBindings {
		if err := deleteClusterRoleBinding(client, &clusterRoleBinding); err != nil {
			clusterRoleBindingDeleteErrors = append(clusterRoleBindingDeleteErrors, err)
		}
		log.Printf("Successfuly deleted ClusterRole: %s", clusterRoleBinding.Name)

	}

	if len(clusterRoleBindingDeleteErrors) > 0 {
		return concateErrors(clusterRoleBindingDeleteErrors)
	}

	return nil
}

func deleteClusterRoleBinding(client *kubernetes.Clientset, crd *rbacv1.ClusterRoleBinding) error {
	return client.RbacV1().ClusterRoleBindings().
		Delete(context.Background(), crd.Name, metav1.DeleteOptions{})
}

func concateErrors(errs []error) error {
	sb := strings.Builder{}

	for _, err := range errs {
		sb.WriteString(err.Error())
		sb.WriteString("\n")
	}

	return fmt.Errorf("%s", sb.String())
}

func (w *workspace) RetrieveCheckupStatus(client *kubernetes.Clientset) (*Status, error) {
	resultsConfigMap, err := w.getResultsConfigMap(client)
	if err != nil {
		return nil, err
	}

	status, err := newStatusFromConfigMap(resultsConfigMap)
	if err != nil {
		return nil, err
	}

	return status, nil
}

func (w *workspace) getResultsConfigMap(client *kubernetes.Clientset) (*corev1.ConfigMap, error) {
	configMap, err := client.CoreV1().ConfigMaps(w.namespace.Name).Get(context.Background(), w.resultConfigMap.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return configMap, nil
}
