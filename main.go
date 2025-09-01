package main

import (
	"log"
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"main/pkg"
)

func main() {
	config, err := clientcmd.BuildConfigFromFlags("", "/root/.kube/config")
	if err != nil {
		log.Fatalln("cannot get cluster config ")
		return
	}
	//create informerfactory
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalln("cannot get kubrentes' clientSet")
		return
	}
	factory := informers.NewFilteredSharedInformerFactory(clientSet, 30*time.Second, "default", func(*v1.ListOptions) {})

	serviceinformer := factory.Core().V1().Services()
	ingressinformer := factory.Networking().V1().Ingresses()

	controller := pkg.NewController(clientSet, serviceinformer, ingressinformer)
	stopCh := make(chan struct{})
	factory.Start(stopCh)
	factory.WaitForCacheSync(stopCh)
	controller.Run(stopCh)
}
