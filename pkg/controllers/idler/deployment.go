package idler

import (
	"context"
	"github.com/go-logr/logr"
	kidlev1beta1 "github.com/orphaner/kidle/pkg/api/v1beta1"
	"github.com/orphaner/kidle/pkg/utils/k8s"
	"github.com/orphaner/kidle/pkg/utils/pointer"
	v1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
)

type DeploymentIdler struct {
	client.Client
	Log        logr.Logger
	Deployment *v1.Deployment
	ObjectIdler
}

func NewDeploymentIdler(client client.Client, log logr.Logger, deployment *v1.Deployment) *DeploymentIdler {
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
	k8s.AddAnnotation(&i.Deployment.ObjectMeta, kidlev1beta1.MetadataPreviousReplicas, strconv.Itoa(int(*i.Deployment.Spec.Replicas)))
	if i.Deployment.Spec.Replicas != pointer.Int32(0) {
		i.Deployment.Spec.Replicas = pointer.Int32(0)
		if err := i.Update(ctx, i.Deployment); err != nil {
			i.Log.Error(err, "unable to downscale deployment")
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
		i.Deployment.Spec.Replicas = previousReplicas
		if err := i.Update(ctx, i.Deployment); err != nil {
			i.Log.Error(err, "unable to wakeup deployment")
			return nil, err
		}
		i.Log.V(1).Info("deployment waked up", "name", i.Deployment.Name)
	} else {
		i.Log.V(2).Info("deployment already waked up", "name", i.Deployment.Name)
	}
	return previousReplicas, nil
}

