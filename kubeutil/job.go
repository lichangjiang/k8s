package kubeutil

import (
	"fmt"
	"time"

	batch "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
)

func RunReplaceableJob(k8sClient kubernetes.Interface,
	job *batch.Job) error {
	existingJob, err := k8sClient.BatchV1().Jobs(job.Namespace).Get(job.Name, metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		klog.Warningf("failed to detect job %s. %+v", job.Name, err)
	} else if err == nil {
		if existingJob.Status.Active > 0 {
			klog.Infof("Found previous job %s.Status=%+v", job.Name, existingJob.Status)
			return nil
		}

		klog.Infof("Removing previous job %s to start a new one", job.Name)
		err := DeleteBatchJob(k8sClient, job.Namespace, existingJob.Name, true)
		if err != nil {
			klog.Warningf("failed to remove job %s. %+v", job.Name, err)
		}
	}

	_, err = k8sClient.BatchV1().Jobs(job.Namespace).Create(job)
	return err
}

func DeleteBatchJob(k8sClient kubernetes.Interface, namespace, name string, wait bool) error {
	propagation := metav1.DeletePropagationForeground
	gracePeriod := int64(0)
	options := &metav1.DeleteOptions{GracePeriodSeconds: &gracePeriod, PropagationPolicy: &propagation}

	if err := k8sClient.BatchV1().Jobs(namespace).Delete(name, options); err != nil {
		return fmt.Errorf("failed to remove previous provisioning job for node %s. %+v", name, err)
	}

	if !wait {
		return nil
	}

	retries := 20
	sleepInterval := 2 * time.Second
	for i := 0; i < retries; i++ {
		_, err := k8sClient.BatchV1().Jobs(namespace).Get(name, metav1.GetOptions{})
		if err != nil && errors.IsNotFound(err) {
			klog.Infof("batch job %s deleted", name)
			return nil
		}

		klog.Infof("batch job %s still exists", name)
		time.Sleep(sleepInterval)
	}

	klog.Warningf("gave up waiting for batch job %s to be deleted", name)
	return nil
}

func WaitForJobCompletion(k8sClient kubernetes.Interface,
	job *batch.Job,
	timeout time.Duration) error {
	klog.Infof("waiting for job %s to complete...", job.Name)
	return wait.Poll(5*time.Second, timeout, func() (bool, error) {
		job, err := k8sClient.BatchV1().Jobs(job.Namespace).Get(job.Name, metav1.GetOptions{})
		if err != nil {
			return false, fmt.Errorf("failed to detect job %s. %+v", job.Name, err)
		}

		if job.Status.Active > 0 {
			return false, nil
		}
		if job.Status.Failed > 0 {
			return false, fmt.Errorf("job %s failed.", job.Name)
		}
		if job.Status.Succeeded > 0 {
			return true, nil
		}
		return false, nil
	})
}
