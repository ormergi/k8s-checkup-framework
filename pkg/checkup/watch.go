package checkup

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8swatch "k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
)

const JobNameLabel = "job-name"

func WaitForJobToComplete(client *kubernetes.Clientset, jobName, jobNamespace string, timeout time.Duration) (*batchv1.Job, error) {
	jobLabel := fmt.Sprintf("%s=%s", JobNameLabel, jobName)
	jobWatcher, err := client.BatchV1().Jobs(jobNamespace).Watch(context.Background(), metav1.ListOptions{LabelSelector: jobLabel})
	if err != nil {
		return nil, err
	}

	completedJob, err := waitForJobCompletionEvent(timeout, jobWatcher)
	if err != nil {
		return nil, err
	}

	return completedJob, nil
}

func waitForJobCompletionEvent(timeout time.Duration, watcher k8swatch.Interface) (*batchv1.Job, error) {
	const timeoutErrorMsg = "timeout reached"
	timeoutTimer := time.NewTimer(timeout)
	eventsCh := watcher.ResultChan()
	defer watcher.Stop()
	for {
		select {
		case <-timeoutTimer.C:
			return nil, fmt.Errorf("%s", timeoutErrorMsg)
		case event := <-eventsCh:
			job, ok := event.Object.(*batchv1.Job)
			if !ok {
				continue
			}

			log.Printf("got job event:\n")
			raw, _ := json.MarshalIndent(job.Status, "", " ")
			log.Println(string(raw))

			if isJobFailedOrCompleted(job) {
				return job, nil
			}
		}
	}
}

func isJobFailedOrCompleted(job *batchv1.Job) bool {
	for _, condition := range job.Status.Conditions {
		if condition.Type == batchv1.JobComplete || condition.Type == batchv1.JobFailed {
			return true
		}
	}
	return false
}
