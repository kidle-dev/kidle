package idler

import (
	"context"
	"github.com/go-logr/logr"
	kidlev1beta1 "github.com/orphaner/kidle/pkg/api/v1beta1"
	"github.com/orphaner/kidle/pkg/utils/k8s"
	"github.com/orphaner/kidle/pkg/utils/pointer"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
)

type StatefulSetIdler struct {
	client.Client
	Log         logr.Logger
	StatefulSet *appsv1.StatefulSet
	ObjectIdler
}

func NewStatefulSetIdler(client client.Client, log logr.Logger, statefulSet *appsv1.StatefulSet) *StatefulSetIdler {
	return &StatefulSetIdler{
		Client:      client,
		Log:         log,
		StatefulSet: statefulSet,
		ObjectIdler: NewObjectIdler(client, log, statefulSet),
	}
}

func (i *StatefulSetIdler) NeedIdle(instance *kidlev1beta1.IdlingResource) bool {
	return instance.Spec.Idle && *i.StatefulSet.Spec.Replicas > 0
}

func (i *StatefulSetIdler) NeedWakeup(instance *kidlev1beta1.IdlingResource) bool {
	return !instance.Spec.Idle && *i.StatefulSet.Spec.Replicas == 0
}

func (i *StatefulSetIdler) Idle(ctx context.Context) error {
	if i.StatefulSet.Spec.Replicas != pointer.Int32(0) {
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			i.Get(ctx, types.NamespacedName{Namespace: i.StatefulSet.Namespace, Name: i.StatefulSet.Name}, i.StatefulSet)
			k8s.AddAnnotation(&i.StatefulSet.ObjectMeta, kidlev1beta1.MetadataPreviousReplicas, strconv.Itoa(int(*i.StatefulSet.Spec.Replicas)))
			k8s.AddAnnotation(&i.StatefulSet.ObjectMeta, kidlev1beta1.MetadataExpectedState, "0")
			i.StatefulSet.Spec.Replicas = pointer.Int32(0)
			return i.Update(ctx, i.StatefulSet)
		})
		if err != nil {
			i.Log.Error(err, "unable to downscale statefulset")
			return err
		}
		i.Log.V(1).Info("statefulset idled", "name", i.StatefulSet.Name)
	} else {
		i.Log.V(2).Info("statefulset already idled", "name", i.StatefulSet.Name)
	}
	return nil
}

func (i *StatefulSetIdler) Wakeup(ctx context.Context) (*int32, error) {
	previousReplicas := pointer.Int32(1)

	if metadataPreviousReplicas, found := k8s.GetAnnotation(&i.StatefulSet.ObjectMeta, kidlev1beta1.MetadataPreviousReplicas); found {
		if v, err := strconv.Atoi(metadataPreviousReplicas); err != nil {
			return nil, err
		} else {
			previousReplicas = pointer.Int32(int32(v))
		}
	}

	if i.StatefulSet.Spec.Replicas != previousReplicas {
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			i.Get(ctx, types.NamespacedName{Namespace: i.StatefulSet.Namespace, Name: i.StatefulSet.Name}, i.StatefulSet)
			k8s.AddAnnotation(&i.StatefulSet.ObjectMeta, kidlev1beta1.MetadataExpectedState, strconv.Itoa(int(*previousReplicas)))
			i.StatefulSet.Spec.Replicas = previousReplicas
			return i.Update(ctx, i.StatefulSet)
		})
		if err != nil {
			i.Log.Error(err, "unable to wakeup statefulset")
			return nil, err
		}
		i.Log.V(1).Info("statefulset waked up", "name", i.StatefulSet.Name)
	} else {
		i.Log.V(2).Info("statefulset already waked up", "name", i.StatefulSet.Name)
	}
	return previousReplicas, nil
}
