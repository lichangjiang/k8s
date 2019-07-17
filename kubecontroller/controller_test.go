package kubecontroller

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/lichangjiang/k8s/kubeutil"
	corev1 "k8s.io/api/core/v1"
)

type ConfigMapController struct {
}

var right int32 = 0
var stopChan = make(chan struct{})
var errorChan = make(chan string)
var tp *testing.T

var configResource = CustomResource{
	Name:   "configmap",
	Plural: "configmaps",
}

func (c *ConfigMapController) AddFunc(obj interface{}) {
	cm, ok := obj.(*corev1.ConfigMap)

	if ok {
		tp.Logf("cm name %s", cm.GetName())
		atomic.AddInt32(&right, 1)
	} else {
		errorChan <- "configmap convert error"
	}
}

func (c *ConfigMapController) UpdateFunc(obj, newobj interface{}) {

}

func (c *ConfigMapController) DeleteFunc(obj interface{}) {

}

func TestFunction(t *testing.T) {

	tp = t
	clientset, err := kubeutil.CreateK8sClientset(kubeutil.DefaultConfigStr(), false)
	if err != nil {
		t.Fatal(err.Error())
	}

	controller := &ConfigMapController{}
	watcher := NewWatcher(controller, configResource, "default", clientset.CoreV1().RESTClient())

	watcher.Watch(&corev1.ConfigMap{}, 2, stopChan)
	t.Log("starting watching configmap")

	select {
	case err := <-errorChan:
		t.Error(err)
		close(stopChan)
	case <-time.After(3 * time.Second):
		close(stopChan)
		if right > 0 {
			t.Logf("add configmap num:%d", right)
		} else {
			t.Errorf("error add configmap")
		}
	}

}
