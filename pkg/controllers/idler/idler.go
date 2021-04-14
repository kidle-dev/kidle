package idler

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	kidlev1beta1 "github.com/orphaner/kidle/pkg/api/v1beta1"
	"github.com/orphaner/kidle/pkg/utils/k8s"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Idler interface {
	SetReference(ctx context.Context, instanceName string) error
	RemoveAnnotations(ctx context.Context) error

	NeedIdle(instance *kidlev1beta1.IdlingResource) bool
	NeedWakeup(instance *kidlev1beta1.IdlingResource) bool

	Idle(ctx context.Context) error
	Wakeup(ctx context.Context) (*int32, error)
}

type ObjectIdler struct {
	client.Client
	Log           logr.Logger
	Object        metav1.Object
	RuntimeObject runtime.Object
}

func NewObjectIdler(client client.Client, log logr.Logger, o interface{}) ObjectIdler {
	return ObjectIdler{
		Client:        client,
		Log:           log,
		Object:        o.(metav1.Object),
		RuntimeObject: o.(runtime.Object),
	}
}

func (o *ObjectIdler) SetReference(ctx context.Context, instanceName string) error {
	if !k8s.HasAnnotation(o.Object, kidlev1beta1.MetadataIdlingResourceReference) {
		o.Log.Info(fmt.Sprintf("Set reference for object %v", o.Object.GetName()))

		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			o.Get(ctx, types.NamespacedName{Namespace: o.Object.GetNamespace(), Name: o.Object.GetName()}, o.RuntimeObject)
			k8s.AddAnnotation(o.Object, kidlev1beta1.MetadataIdlingResourceReference, instanceName)
			return o.Update(ctx, o.RuntimeObject)
		})
		if err != nil {
			o.Log.Error(err, "unable to add reference in annotations")
			return err
		}
	}
	return nil
}

func (o *ObjectIdler) RemoveAnnotations(ctx context.Context) error {
	if k8s.HasAnnotation(o.Object, kidlev1beta1.MetadataIdlingResourceReference) ||
		k8s.HasAnnotation(o.Object, kidlev1beta1.MetadataPreviousReplicas) {
		o.Log.Info(fmt.Sprintf("Remove annotations for object %v", o.Object.GetName()))

		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			o.Get(ctx, types.NamespacedName{Namespace: o.Object.GetNamespace(), Name: o.Object.GetName()}, o.RuntimeObject)
			k8s.RemoveAnnotation(o.Object, kidlev1beta1.MetadataIdlingResourceReference)
			k8s.RemoveAnnotation(o.Object, kidlev1beta1.MetadataPreviousReplicas)
			return o.Update(ctx, o.RuntimeObject)
		})
		if err != nil {
			o.Log.Error(err, "unable to remove kidle annotations")
			return err
		}
	}
	return nil
}
