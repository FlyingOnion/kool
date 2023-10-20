package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/FlyingOnion/kool"
	"github.com/spf13/pflag"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

func mustGetOrLogFatal[T any](v T, err error) T {
	if err != nil {
		klog.Fatal(err)
	}
	return v
}

func main() {
	var kubeconfig string
	var master string

	pflag.StringVar(&kubeconfig, "kubeconfig", "", "absolute path to the kubeconfig file")
	pflag.StringVar(&master, "master", "", "master url")
	pflag.Parse()

	config := mustGetOrLogFatal(clientcmd.BuildConfigFromFlags(master, kubeconfig))
	client := mustGetOrLogFatal(rest.RESTClientFor(config))

	podInformer := kool.NewInformer[corev1.Pod](client, 30*time.Second)

	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	controller := NewController(podInformer, queue, 3) // TODO: replace retryOnError

	sigC := make(chan os.Signal, 1)
	signal.Notify(sigC, syscall.SIGINT, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())

	go controller.Run(ctx, 1)

	select {
	case sig := <-sigC:
		klog.Infof("Received signal: %s", sig)
		signal.Stop(sigC)
		cancel()
	case <-ctx.Done():
	}

}
