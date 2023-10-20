package main

import (
	"context"
	"errors"
	"time"

	"github.com/FlyingOnion/kool"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

var (
	ErrSyncTimeout = errors.New("Timed out waiting for caches to sync")
)

type Controller struct {
	podLister kool.Lister[corev1.Pod] // range

	podSynced cache.InformerSynced // range

	queue        workqueue.RateLimitingInterface
	retryOnError int
}

func NewController(
	podInformer kool.Informer[corev1.Pod], // range
	queue workqueue.RateLimitingInterface,
	retryOnError int,
) *Controller {
	c := &Controller{
		podLister:    podInformer.Lister(),             // range
		podSynced:    podInformer.Informer().HasSynced, // range
		queue:        queue,
		retryOnError: retryOnError, // TODO: replace this
	}
	podInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.AddPod,
		UpdateFunc: c.UpdatePod,
		DeleteFunc: c.DeletePod,
	})
	return c
}

func (c *Controller) Run(ctx context.Context, workers int) {
	defer runtime.HandleCrash()

	// Let the workers stop when we are done
	defer c.queue.ShutDown()
	logger := klog.FromContext(ctx)
	logger.Info("Starting pod controller")       // TODO: set placeholder
	defer logger.Info("Stopping pod controller") // TODO: set placeholder

	// go c.podInformer.Run(stopCh) // TODO: move to factory

	// Wait for all involved caches to be synced, before processing items from the queue is started
	if !cache.WaitForCacheSync(ctx.Done(), c.podSynced) {
		runtime.HandleError(ErrSyncTimeout)
		return
	}

	for i := 0; i < workers; i++ {
		go wait.UntilWithContext(ctx, c.runWorker, time.Second)
	}

	<-ctx.Done()
}

func (c *Controller) runWorker(ctx context.Context) {
	for c.processNextItem(ctx) {
	}
}

func (c *Controller) processNextItem(ctx context.Context) bool {
	// Wait until there is a new item in the working queue
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	// Tell the queue that we are done with processing this key. This unblocks the key for other workers
	// This allows safe parallel processing because two pods with the same key are never processed in
	// parallel.
	defer c.queue.Done(key)

	// Invoke the method containing the business logic
	err := c.sync(ctx, key.(string))
	// Handle the error if something went wrong during the execution of the business logic
	c.handleErr(ctx, err, key)

	return true
}

func (c *Controller) sync(ctx context.Context, key string) error {
	logger := klog.FromContext(ctx)
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		logger.Error(err, "Failed to split meta namespace cache key", "cacheKey", key)
		return err
	}
	return c.doSync(ctx, namespace, name)
}

func (c *Controller) handleErr(ctx context.Context, err error, key any) {
	// TODO: modify this function
	if err == nil {
		c.queue.Forget(key)
		return
	}

	logger := klog.FromContext(ctx)

	if c.queue.NumRequeues(key) < c.retryOnError {
		logger.Error(err, "Failed to sync object", "cacheKey", key)
		c.queue.AddRateLimited(key)
		return
	}

	c.queue.Forget(key)
	runtime.HandleError(err)
	logger.Info("Dropping object out of the queue", "cacheKey", key)
}
