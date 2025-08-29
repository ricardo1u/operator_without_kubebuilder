package pkg

import (
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
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

	return nil
}

func (c *controller) addService(obj interface{}) {

}

func (c *controller) updatService(oldobj interface{}, newobj interface{}) {

}

func (c *controller) deleteIngress(obj interface{}) {

}

func (c *controller) enqueue(obj interface{}) {

}

func (c *controller) dequeue(obj interface{}) {

}

func (c *controller) Run(stopCh chan struct{}) {

}

func (c *controller) worker() {

}

func (c *controller) processNextItem() bool {
	return true
}

func (c *controller) handleError(key string, err error) {

}

func (c *controller) syncService(key string) error {
	return nil
}

func (c *controller) createIngress(service *corev1.Service) *netv1.Ingress {
	return nil
}
