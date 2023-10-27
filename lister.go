package kool

import (
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
)

type Lister[T any] interface {
	List(selector labels.Selector) ([]*T, error)
	Namespaced(ns string) NamespacedLister[T]
}

type NamespacedLister[T any] interface {
	List(selector labels.Selector) ([]*T, error)
	Get(name string) (*T, error)
}

// lister[T] implements generic interface Lister[T]
type lister[T any] struct {
	indexer cache.Indexer
}

func (l *lister[T]) List(selector labels.Selector) (ret []*T, err error) {
	err = cache.ListAll(l.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*T))
	})
	return ret, err
}

func (l *lister[T]) Namespaced(ns string) NamespacedLister[T] {
	nl := namespacedLister[T]{ns: ns, indexer: l.indexer}
	return &nl
}

func NewLister[T any](indexer cache.Indexer) Lister[T] {
	mustBeRuntimeObject(new(T))
	l := lister[T]{indexer: indexer}
	return &l
}

// namespacedLister[T] implements generic interface NamespacedLister[T]
type namespacedLister[T any] struct {
	ns      string
	indexer cache.Indexer
}

func (l *namespacedLister[T]) List(selector labels.Selector) (ret []*T, err error) {
	err = cache.ListAllByNamespace(l.indexer, l.ns, selector, func(m interface{}) {
		ret = append(ret, m.(*T))
	})
	return ret, err
}

func (l *namespacedLister[T]) Get(name string) (*T, error) {
	obj, exists, err := l.indexer.GetByKey(l.ns + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(schema.GroupResource{}, name) // TODO: change groupresource
	}
	return obj.(*T), nil
}
