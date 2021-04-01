package controllers

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/orphaner/kidle/pkg/api/v1beta1"
	"github.com/orphaner/kidle/pkg/utils/pointer"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
)

type Idler struct {
	client.Client
	Log logr.Logger
	record.EventRecorder
}

const MetadataPreviousReplicas = "kidle.beroot.org/previous-replicas"

func (i *Idler) Reconcile(ctx context.Context, instance v1beta1.IdlingResource, deploy *v1.Deployment) (ctrl.Result, error) {

	// Idle Deployment
	if instance.Spec.Idle && *deploy.Spec.Replicas > 0 {
		if err := i.Idle(ctx, deploy); err != nil {
			i.Event(&instance,
				corev1.EventTypeWarning,
				"ScalingDeployment",
				fmt.Sprintf("Failed to idle deployment %s: %s", instance.Spec.IdlingResourceRef.Name, err))
			return ctrl.Result{}, fmt.Errorf("error during idling: %v", err)
		}
		i.Event(&instance,
			corev1.EventTypeNormal,
			"ScalingDeployment",
			fmt.Sprintf("Scaled to 0"))
		return ctrl.Result{}, nil
	}

	// Wakeup deployment
	if !instance.Spec.Idle && *deploy.Spec.Replicas == 0 {
		if replicas, err := i.Wakeup(ctx, deploy); err != nil {
			i.Event(&instance,
				corev1.EventTypeWarning,
				"ScalingDeployment",
				fmt.Sprintf("Failed to wake up deployment %s: %s", instance.Spec.IdlingResourceRef.Name, err))
			return ctrl.Result{}, fmt.Errorf("error during waking up: %v", err)
		} else {
			i.Event(&instance,
				corev1.EventTypeNormal,
				"ScalingDeployment",
				fmt.Sprintf("Scaled to %d", replicas))
			return ctrl.Result{}, nil
		}
	}
	return ctrl.Result{}, nil
}

func (i *Idler) Idle(ctx context.Context, d *v1.Deployment) error {
	if d.ObjectMeta.Annotations == nil {
		d.ObjectMeta.Annotations = make(map[string]string)
	}
	d.ObjectMeta.Annotations[MetadataPreviousReplicas] = strconv.Itoa(int(*d.Spec.Replicas))
	d.Spec.Replicas = pointer.Int32(0)
	if err := i.Update(ctx, d); err != nil {
		i.Log.Error(err, "unable to downscale deployment")
		return err
	}
	i.Log.V(1).Info("deployment idled", "name", d.Name)
	return nil
}

func (i *Idler) Wakeup(ctx context.Context, d *v1.Deployment) (*int32, error) {
	var previousReplicas *int32
	if v, err := strconv.Atoi(d.ObjectMeta.Annotations[MetadataPreviousReplicas]); err != nil {
		return nil, err
	} else {
		previousReplicas = pointer.Int32(int32(v))
	}
	d.Spec.Replicas = previousReplicas
	if err := i.Update(ctx, d); err != nil {
		i.Log.Error(err, "unable to wakeup deployment")
		return nil, err
	}
	i.Log.V(1).Info("deployment waked up", "name", d.Name)
	return previousReplicas, nil
}
