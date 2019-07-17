package kubecontroller

import (
	"fmt"
	"time"

	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/fields"
	rt "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
)

type CustomController interface {
	AddFunc(obj interface{})
	UpdateFunc(oldObj, newObj interface{})
	DeleteFunc(obj interface{})
}

// CustomResource is for creating a Kubernetes TPR/CRD
type CustomResource struct {
	// Name of the custom resource
	Name string

	// Plural of the custom resource in plural
	Plural string

	// Group the custom resource belongs to
	Group string

	// Version which should be defined in a const above
	Version string

	// Scope of the CRD. Namespaced or cluster
	Scope apiextensionsv1beta1.ResourceScope

	// Kind is the serialized interface of the resource.
	Kind string

	// ShortNames is the shortened version of the resource
	ShortNames []string
}

type Watcher struct {
	controller CustomController
	namespace  string
	client     rest.Interface
	resource   CustomResource

	indexer  cache.Indexer
	queue    workqueue.RateLimitingInterface
	informer cache.Controller
}

type message struct {
	operator string
	oldObj   interface{}
	obj      interface{}
}

func NewWatcher(controller CustomController, resource CustomResource, namespace string, client rest.Interface) *Watcher {
	return &Watcher{
		controller: controller,
		resource:   resource,
		namespace:  namespace,
		client:     client,
	}
}

func (w *Watcher) Watch(objType rt.Object, threadNum int, done <-chan struct{}) {

	listWatch := cache.NewListWatchFromClient(
		w.client,
		w.resource.Plural,
		w.namespace,
		fields.Everything(),
	)

	rateLimitQueue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	indexer, informer := cache.NewIndexerInformer(listWatch, objType, 0, cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			rateLimitQueue.Add(message{
				operator: "add",
				obj:      obj,
			})
		},
		UpdateFunc: func(obj interface{}, newObj interface{}) {
			rateLimitQueue.Add(message{
				operator: "update",
				obj:      newObj,
				oldObj:   obj,
			})
		},
		DeleteFunc: func(obj interface{}) {
			rateLimitQueue.Add(message{
				operator: "delete",
				obj:      obj,
			})
		},
	}, cache.Indexers{})

	w.queue = rateLimitQueue
	w.informer = informer
	w.indexer = indexer

	go w.run(threadNum, done)
}

func (w *Watcher) run(threadniess int, stopchan <-chan struct{}) {
	defer runtime.HandleCrash()

	defer w.queue.ShutDown()

	go w.informer.Run(stopchan)

	if !cache.WaitForCacheSync(stopchan, w.informer.HasSynced) {
		runtime.HandleError(fmt.Errorf("Timeout waiting for caches to sync"))
		return
	}

	for i := 0; i < threadniess; i++ {
		go wait.Until(w.runworker, time.Second, stopchan)
	}

	<-stopchan
	klog.Infof("Stopping %s controller", w.resource.Plural)
}

func (w *Watcher) runworker() {
	for w.processNextItem() {
	}
}

func (w *Watcher) processNextItem() bool {

	key, quit := w.queue.Get()
	if quit {
		return false
	}

	defer w.queue.Done(key)

	err := w.syncToHandle(key.(message))
	if err != nil {
		klog.Errorf("Fetching object with key %s from store failed with %v", key, err)
	}

	w.errorHandler(key.(message), err)
	return true
}

func (w *Watcher) errorHandler(msg message, err error) {

	key := msg
	if err == nil {
		w.queue.Forget(key)
		return
	}

	//retry 5 times
	if w.queue.NumRequeues(key) < 5 {
		w.queue.AddRateLimited(key)
		return
	}

	w.queue.Forget(key)
	runtime.HandleError(err)
}

func (w *Watcher) syncToHandle(msg message) error {
	obj := msg.obj

	if obj == nil {
		klog.Warningf("Get nil obj from queue")
		return nil
	}

	switch msg.operator {
	case "add":
		w.controller.AddFunc(obj)
	case "update":
		w.controller.UpdateFunc(msg.oldObj, obj)
	case "delete":
		w.controller.DeleteFunc(obj)
	default:
		klog.Errorf("%s's operator %s unknown\n", w.resource.Plural, msg.operator)
	}
	return nil
}
