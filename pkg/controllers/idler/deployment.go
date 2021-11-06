package idler

import (
	"context"
	"github.com/go-logr/logr"
	kidlev1beta1 "github.com/kidle-dev/kidle/pkg/api/v1beta1"
	"github.com/kidle-dev/kidle/pkg/utils/k8s"
	"github.com/kidle-dev/kidle/pkg/utils/pointer"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
)

type DeploymentIdler struct {
	client.Client
	Log        logr.Logger
	Deployment *appsv1.Deployment
	ObjectIdler
}

func NewDeploymentIdler(client client.Client, log logr.Logger, deployment *appsv1.Deployment) *DeploymentIdler {
	return &DeploymentIdler{
		Client:      client,
		Log:         log,
		Deployment:  deployment,
		ObjectIdler: NewObjectIdler(client, log, deployment),
	}
}

func (i *DeploymentIdler) NeedIdle(instance *kidlev1beta1.IdlingResource) bool {
	return instance.Spec.Idle && *i.Deployment.Spec.Replicas > 0
}

func (i *DeploymentIdler) NeedWakeup(instance *kidlev1beta1.IdlingResource) bool {
	return !instance.Spec.Idle && *i.Deployment.Spec.Replicas == 0
}

func (i *DeploymentIdler) Idle(ctx context.Context) error {
	if i.Deployment.Spec.Replicas != pointer.Int32(0) {
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			if err := i.Get(ctx, types.NamespacedName{Namespace: i.Deployment.Namespace, Name: i.Deployment.Name}, i.Deployment); err != nil {
				i.Log.Error(err, "unable to get deployment","name", i.Deployment.Name)
				return err
			}
			k8s.AddAnnotation(&i.Deployment.ObjectMeta, kidlev1beta1.MetadataPreviousReplicas, strconv.Itoa(int(*i.Deployment.Spec.Replicas)))
			k8s.AddAnnotation(&i.Deployment.ObjectMeta, kidlev1beta1.MetadataExpectedState, "0")
			i.Deployment.Spec.Replicas = pointer.Int32(0)
			return i.Update(ctx, i.Deployment)
		})
		if err != nil {
			i.Log.Error(err, "unable to downscale deployment", "name", i.Deployment.Name)
			return err
		}
		i.Log.V(1).Info("deployment idled", "name", i.Deployment.Name)
	} else {
		i.Log.V(2).Info("deployment already idled", "name", i.Deployment.Name)
	}
	return nil
}

func (i *DeploymentIdler) Wakeup(ctx context.Context) (*int32, error) {
	previousReplicas := pointer.Int32(1)

	if metadataPreviousReplicas, found := k8s.GetAnnotation(&i.Deployment.ObjectMeta, kidlev1beta1.MetadataPreviousReplicas); found {
		if v, err := strconv.Atoi(metadataPreviousReplicas); err != nil {
			return nil, err
		} else {
			previousReplicas = pointer.Int32(int32(v))
		}
	}

	if i.Deployment.Spec.Replicas != previousReplicas {
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			if err := i.Get(ctx, types.NamespacedName{Namespace: i.Deployment.Namespace, Name: i.Deployment.Name}, i.Deployment); err != nil {
				i.Log.Error(err, "unable to get deployment","name", i.Deployment.Name)
				return err
			}
			k8s.AddAnnotation(&i.Deployment.ObjectMeta, kidlev1beta1.MetadataExpectedState, strconv.Itoa(int(*previousReplicas)))
			i.Deployment.Spec.Replicas = previousReplicas
			return i.Update(ctx, i.Deployment)
		})
		if err != nil {
			i.Log.Error(err, "unable to wakeup deployment", "name", i.Deployment.Name)
			return nil, err
		}
		i.Log.V(1).Info("deployment woke up", "name", i.Deployment.Name)
	} else {
		i.Log.V(2).Info("deployment already woke up", "name", i.Deployment.Name)
	}
	return previousReplicas, nil
}
