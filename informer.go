package kool

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

type Informer[T any] interface {
	Informer() cache.SharedIndexInformer
	Lister() Lister[T]
}

func New() {
	var scheme *runtime.Scheme
	scheme.KnownTypes()
}

func NewFilteredInformer[T any](client Client[T], ns string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	obj := mustBeRuntimeObject(new(T))
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				return client.List(context.TODO(), options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return client.Watch(context.TODO(), options)
			},
		},
		obj,
		resyncPeriod,
		cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
	)
}
