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
	informer cache.SharedIndexInformer
}

func (m *informer[T]) Informer() cache.SharedIndexInformer {
	return m.informer
}

func (m *informer[T]) Lister() Lister[T] {
	return NewLister[T](m.Informer().GetIndexer())
}

func NewInformer[T any](client *rest.RESTClient, resyncPeriod time.Duration) Informer[T] {
	return &informer[T]{newSharedIndexInformer[T](client, "", resyncPeriod)}
}

type NamespacedInformer[T any] interface {
	Informer() cache.SharedIndexInformer
	Lister() NamespacedLister[T]
}

type namespacedInformer[T any] struct {
	ns       string
	informer cache.SharedIndexInformer
}

func (m *namespacedInformer[T]) Informer() cache.SharedIndexInformer {
	return m.informer
}

func (m *namespacedInformer[T]) Lister() NamespacedLister[T] {
	return &namespacedLister[T]{
		ns:      m.ns,
		indexer: m.Informer().GetIndexer(),
	}
}

func NewNamespacedInformer[T any](client *rest.RESTClient, ns string, resyncPeriod time.Duration) NamespacedInformer[T] {
	return &namespacedInformer[T]{
		ns:       ns,
		informer: newSharedIndexInformer[T](client, ns, resyncPeriod),
	}
}

func newSharedIndexInformer[T any](client *rest.RESTClient, ns string, resyncPeriod time.Duration) cache.SharedIndexInformer {
	obj := mustBeRuntimeObject(new(T))
	kind := reflect.TypeOf(obj).Elem().Name()
	resource := plural.Plural(strings.ToLower(kind))
	c := NewTypedClient[T](client, ns, resource)
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				return c.List(context.TODO(), options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return c.Watch(context.TODO(), options)
			},
		},
		obj,
		resyncPeriod,
		cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
	)
}
