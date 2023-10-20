package kool

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
	"k8s.io/kubectl/pkg/scheme"
)

type Client[T any] interface {
	Create(ctx context.Context, item *T, opts metav1.CreateOptions) (*T, error)
	Update(ctx context.Context, name string, item *T, opts metav1.UpdateOptions) (*T, error)
	UpdateStatus(ctx context.Context, name string, item *T, opts metav1.UpdateOptions) (*T, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*T, error)
	List(ctx context.Context, opts metav1.ListOptions) (*List[T], error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *T, err error)
}

type ClientWithApply[T, ApplyConfiguration any] interface {
	Client[T]
	Apply(ctx context.Context, ac *ApplyConfiguration, opts metav1.ApplyOptions) (result *T, err error)
	ApplyStatus(ctx context.Context, ac *ApplyConfiguration, opts metav1.ApplyOptions) (result *T, err error)
}

type restClient[T any] struct {
	client   *rest.RESTClient
	ns       string
	resource string
}

func NewTypedClient[T any](client *rest.RESTClient, ns, resource string) Client[T] {
	mustBeRuntimeObject(new(T))
	return &restClient[T]{
		client:   client,
		ns:       ns,
		resource: resource,
	}
}

func (c *restClient[T]) Create(ctx context.Context, item *T, opts metav1.CreateOptions) (result *T, err error) {
	result = new(T)
	err = c.client.Post().
		Namespace(c.ns).
		Resource(c.resource).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(item).
		Do(ctx).
		Into(mustBeRuntimeObject(result))
	return
}

func (c *restClient[T]) Update(ctx context.Context, name string, item *T, opts metav1.UpdateOptions) (result *T, err error) {
	result = new(T)
	err = c.client.Put().
		Namespace(c.ns).
		Resource(c.resource).
		Name(name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(item).
		Do(ctx).
		Into(mustBeRuntimeObject(result))
	return
}

func (c *restClient[T]) UpdateStatus(ctx context.Context, name string, item *T, opts metav1.UpdateOptions) (result *T, err error) {
	result = new(T)
	err = c.client.Put().
		Namespace(c.ns).
		Resource(c.resource).
		Name(name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(item).
		Do(ctx).
		Into(mustBeRuntimeObject(result))
	return
}

func (c *restClient[T]) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource(c.resource).
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

func (c *restClient[T]) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = MultiplyDuration(time.Second, *listOpts.TimeoutSeconds)
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource(c.resource).
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

func (c *restClient[T]) Get(ctx context.Context, name string, opts metav1.GetOptions) (result *T, err error) {
	result = new(T)
	err = c.client.Get().
		Namespace(c.ns).
		Resource(c.resource).
		Name(name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Do(ctx).
		Into(mustBeRuntimeObject(result))
	return
}

func (c *restClient[T]) List(ctx context.Context, opts metav1.ListOptions) (result *List[T], err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = MultiplyDuration(time.Second, *opts.TimeoutSeconds)
	}
	result = &List[T]{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource(c.resource).
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

func (c *restClient[T]) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = MultiplyDuration(time.Second, *opts.TimeoutSeconds)
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource(c.resource).
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

func (c *restClient[T]) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *T, err error) {
	result = new(T)
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource(c.resource).
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(mustBeRuntimeObject(result))
	return
}

var _ Client[corev1.Pod] = &restClient[corev1.Pod]{}
