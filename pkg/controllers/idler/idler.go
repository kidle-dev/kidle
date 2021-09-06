package idler

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	kidlev1beta1 "github.com/kidle-dev/kidle/pkg/api/v1beta1"
	"github.com/kidle-dev/kidle/pkg/utils/k8s"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	RuntimeObject client.Object
}

func NewObjectIdler(k8sClient client.Client, log logr.Logger, o interface{}) ObjectIdler {
	return ObjectIdler{
		Client:        k8sClient,
		Log:           log,
		Object:        o.(metav1.Object),
		RuntimeObject: o.(client.Object),
	}
}

func (o *ObjectIdler) SetReference(ctx context.Context, instanceName string) error {
	if !k8s.HasAnnotation(o.Object, kidlev1beta1.MetadataIdlingResourceReference) {
		o.Log.Info(fmt.Sprintf("Set reference for object %v", o.Object.GetName()))

		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			if err := o.Get(ctx, types.NamespacedName{Namespace: o.Object.GetNamespace(), Name: o.Object.GetName()}, o.RuntimeObject); err != nil {
				return err
			}
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
			if err := o.Get(ctx, types.NamespacedName{Namespace: o.Object.GetNamespace(), Name: o.Object.GetName()}, o.RuntimeObject); err != nil {
				return err
			}
			k8s.RemoveAnnotation(o.Object, kidlev1beta1.MetadataIdlingResourceReference)
			k8s.RemoveAnnotation(o.Object, kidlev1beta1.MetadataPreviousReplicas)
			k8s.RemoveAnnotation(o.Object, kidlev1beta1.MetadataExpectedState)
			return o.Update(ctx, o.RuntimeObject)
		})
		if err != nil {
			o.Log.Error(err, "unable to remove kidle annotations")
			return err
		}
	}
	return nil
}
