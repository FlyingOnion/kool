package kool

import (
	"context"
	"reflect"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

type Informer[T any] interface {
	Informer() cache.SharedIndexInformer
	Lister() Lister[T]
}

type informer[T any] struct {
	client       Client[T]
	resyncPeriod time.Duration
	// obj is used to initialize SharedIndexInformer
	obj runtime.Object
}

func (m *informer[T]) Informer() cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				return m.client.List(context.TODO(), options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return m.client.Watch(context.TODO(), options)
			},
		},
		m.obj,
		m.resyncPeriod,
		cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
	)
}

func (m *informer[T]) Lister() Lister[T] {
	return NewLister[T](m.Informer().GetIndexer())
}

func NewInformer[T any](client *rest.RESTClient, resyncPeriod time.Duration) Informer[T] {
	obj := mustBeRuntimeObject(new(T))
	kind := reflect.TypeOf(obj).Elem().Name()
	resource := plural.Plural(strings.ToLower(kind))
	return &informer[T]{
		client:       NewTypedClient[T](client, metav1.NamespaceAll, resource),
		resyncPeriod: resyncPeriod,
		obj:          obj,
	}
}

type NamespacedInformer[T any] interface {
	Informer() cache.SharedIndexInformer
	Lister() NamespacedLister[T]
}

type namespacedInformer[T any] struct {
	ns           string
	client       Client[T]
	resyncPeriod time.Duration
	// obj is used to initialize SharedIndexInformer
	obj runtime.Object
}

func (m *namespacedInformer[T]) Informer() cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				return m.client.List(context.TODO(), options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return m.client.Watch(context.TODO(), options)
			},
		},
		m.obj,
		m.resyncPeriod,
		cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
	)
}

func (m *namespacedInformer[T]) Lister() NamespacedLister[T] {
	return &namespacedLister[T]{
		ns:      m.ns,
		indexer: m.Informer().GetIndexer(),
	}
}

func NewNamespacedInformer[T any](client *rest.RESTClient, ns string, resyncPeriod time.Duration) NamespacedInformer[T] {
	obj := mustBeRuntimeObject(new(T))
	kind := reflect.TypeOf(obj).Elem().Name()
	resource := plural.Plural(strings.ToLower(kind))
	return &namespacedInformer[T]{
		ns:           ns,
		client:       NewTypedClient[T](client, ns, resource),
		resyncPeriod: resyncPeriod,
		obj:          obj,
	}
}
