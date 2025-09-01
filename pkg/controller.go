package pkg

import (
	"context"
	"reflect"
	"time"

	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	informercorev1 "k8s.io/client-go/informers/core/v1"
	informernetv1 "k8s.io/client-go/informers/networking/v1"
	"k8s.io/client-go/kubernetes"
	listercorev1 "k8s.io/client-go/listers/core/v1"
	listernetv1 "k8s.io/client-go/listers/networking/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

const (
	workNum  = 5
	annoKey  = "ingress/http"
	maxRetry = 10
)

type QueueKey string

type controller struct {
	client        kubernetes.Interface
	serviceLister listercorev1.ServiceLister
	ingressLister listernetv1.IngressLister
	queue         workqueue.TypedRateLimitingInterface[QueueKey]
}

func NewController(clientset *kubernetes.Clientset,
	serviceInformer informercorev1.ServiceInformer,
	ingressInformer informernetv1.IngressInformer,
) *controller {
	c := controller{
		client:        clientset,
		serviceLister: serviceInformer.Lister(),
		ingressLister: ingressInformer.Lister(),
		queue:         workqueue.NewTypedRateLimitingQueue(workqueue.TypedRateLimiter[QueueKey](workqueue.TypedWithMaxWaitRateLimiter[QueueKey]{})),
	}
	//add serviceInformer to ResourceEventHandler
	serviceInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addService,
		UpdateFunc: c.updatService,
	})
	ingressInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		DeleteFunc: c.deleteIngress,
	})
	return &c
}

func (c *controller) addService(obj interface{}) {
	c.enqueue(obj)
}

func (c *controller) updatService(oldobj interface{}, newobj interface{}) {
	if reflect.DeepEqual(oldobj, newobj) {
		return
	}
	c.enqueue(newobj)
}

func (c *controller) deleteIngress(obj interface{}) {
	ingress := obj.(*netv1.Ingress)
	ownerference := metav1.GetControllerOf(ingress)
	//if ownerference represent, delete the ownerference
	if ownerference != nil && ownerference.Kind != "Service" {
		return
	}
	c.enqueue(obj)
}

func (c *controller) enqueue(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		runtime.HandleError(err)
		return
	}
	key1 := QueueKey(key)
	c.queue.Add(key1)
}

func (c *controller) dequeue(obj interface{}) {

}

func (c *controller) Run(stopCh chan struct{}) {
	for i := 0; i < workNum; i++ {
		go wait.Until(c.worker, time.Minute, stopCh)
	}
	<-stopCh
}

func (c *controller) worker() {
	for c.processNextItem() {
	}
}

func (c *controller) processNextItem() bool {
	item, shutdown := c.queue.Get()
	if shutdown {
		return false
	}
	//remove item from queue after done
	defer c.dequeue(item)
	//deal service logic
	err := c.syncService(item)
	if err != nil {
		c.handleError(item, err)
	}
	return true
}

func (c *controller) handleError(key QueueKey, err error) {
	//if number of retry lower than maxtry , retry again
	if c.queue.NumRequeues(key) < maxRetry {
		c.queue.AddRateLimited(key)
		return
	}
	runtime.HandleError(err)
	c.queue.Forget(key)
}

func (c *controller) syncService(key QueueKey) error {
	keyname := string(key)
	//split keyname to namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(keyname)
	if err != nil {
		return err
	}
	//get service from indexer
	service, err := c.serviceLister.Services(namespace).Get(name)
	//canot found service in indexer , do nothing
	if errors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	_, ok := service.Annotations[annoKey]
	//convention over configuration
	ingress, err := c.ingressLister.Ingresses(namespace).Get(name)
	if ok && errors.IsNotFound(err) {
		ig := c.createIngress(service)
		_, err = c.client.NetworkingV1().Ingresses(namespace).Create(context.TODO(), ig, metav1.CreateOptions{})
		if err != nil {
			return err
		}
	} else if !ok && ingress != nil {
		c.client.NetworkingV1().Ingresses(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *controller) createIngress(service *corev1.Service) *netv1.Ingress {
	icn := "ingress"
	PathType := netv1.PathTypePrefix
	controllerRef := metav1.GetControllerOf(service)
	var ownerRefs []metav1.OwnerReference
	if controllerRef != nil {
		ownerRefs = []metav1.OwnerReference{*controllerRef}
	}
	return &netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:            service.Name,
			Namespace:       service.Namespace,
			OwnerReferences: ownerRefs,
		},
		Spec: netv1.IngressSpec{
			IngressClassName: &icn,
			Rules: []netv1.IngressRule{
				{
					Host: "example.com",
					IngressRuleValue: netv1.IngressRuleValue{
						HTTP: &netv1.HTTPIngressRuleValue{
							Paths: []netv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &PathType,
									Backend: netv1.IngressBackend{
										Service: &netv1.IngressServiceBackend{
											Name: service.Name,
											Port: netv1.ServiceBackendPort{
												Number: 80,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}
