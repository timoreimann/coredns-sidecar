package main

import (
	"flag"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
)

var (
	kubeconfig string
	masterURL  string
	hostsPath  string
)

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&hostsPath, "hostsfile", "", "Path to the hosts file. Log to standard out if missing.")
}

func main() {
	klog.InitFlags(nil)
	flag.Parse()

	stopCh := make(chan struct{})
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		close(stopCh)
	}()

	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		klog.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	informerFactory := informers.NewSharedInformerFactory(kubeClient, 10*time.Second)

	var hostsWriter io.Writer = os.Stdout
	if hostsPath != "" {
		klog.Infof("Using hosts file: %s", hostsPath)
		hostsWriter = atomicFileWriter{hostsPath}
	} else {
		klog.Info("Logging hosts updates to stdout")
	}

	controller := newController(kubeClient, informerFactory.Core().V1().Nodes(), hostsWriter)
	informerFactory.Start(stopCh)

	if err := controller.Run(stopCh, nil); err != nil {
		klog.Fatalf("Failed to run controller loop: %s", err)
	}
}
