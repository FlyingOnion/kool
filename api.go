package kool

import (
	"github.com/mitchellh/mapstructure"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type DeepCopyGen[T any] interface {
	DeepCopyInto(*T)
}

func DeepCopy[T any](in, out *T) {
	var _in any = in
	if d, ok := _in.(DeepCopyGen[T]); ok {
		// use deepcopy-gen
		d.DeepCopyInto(out)
		return
	}
	mapstructure.Decode(in, out)
}

type List[T any] struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []T `json:"items"`
}

func (listA *List[T]) DeepCopyInto(listB *List[T]) {
	*listB = *listA
	listB.TypeMeta = listA.TypeMeta
	listA.ListMeta.DeepCopyInto(&listB.ListMeta)
	if listA.Items != nil {
		in, out := &listA.Items, &listB.Items

		*out = make([]T, len(*in))
		for i := range *in {
			DeepCopy[T](&(*in)[i], &(*out)[i])
		}
	}
	return
}

func (in *List[T]) DeepCopy() *List[T] {
	if in == nil {
		return nil
	}
	out := new(List[T])
	in.DeepCopyInto(out)
	return out
}

func (in *List[T]) DeepCopyObject() runtime.Object {
	if copy := in.DeepCopy(); copy != nil {
		return copy
	}
	return nil
}
