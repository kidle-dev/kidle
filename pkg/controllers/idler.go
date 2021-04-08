package controllers

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	kidlev1beta1 "github.com/orphaner/kidle/pkg/api/v1beta1"
	"github.com/orphaner/kidle/pkg/utils/k8s"
	"github.com/orphaner/kidle/pkg/utils/pointer"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
)

type Idler interface {
	SetReference(ctx context.Context, instanceName string) error
	RemoveAnnotations(ctx context.Context) error

	NeedIdle(instance *kidlev1beta1.IdlingResource) bool
	NeedWakeup(instance *kidlev1beta1.IdlingResource) bool

	Idle(ctx context.Context) error
	Wakeup(ctx context.Context) (*int32, error)
}

type DeploymentIdler struct {
	client.Client
	Log        logr.Logger
	Deployment *v1.Deployment
}

func (r *IdlingResourceReconciler) ReconcileWithIdler(ctx context.Context, instance *kidlev1beta1.IdlingResource, idler Idler) (ctrl.Result, error) {


	// Add a reference on the deployment
	err := idler.SetReference(ctx, instance.Name)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error during adding annotation: %v", err)
	}

	// Deal with the idling resource deletion
	if instance.IsBeingDeleted() {

		// Wakeup deployment
		replicas, err := idler.Wakeup(ctx)
		if err != nil {
			r.Event(instance,
				corev1.EventTypeWarning,
				"RestoringDeployment",
				fmt.Sprintf("Failed to restore deployment %s: %s", instance.Spec.IdlingResourceRef.Name, err))
			return ctrl.Result{}, fmt.Errorf("error during restoring: %v", err)
		}
		r.Event(instance,
			corev1.EventTypeNormal,
			"RestoringDeployment",
			fmt.Sprintf("Restored to %d", *replicas))

		// Remove deployment annotations
		if err := idler.RemoveAnnotations(ctx); err != nil {
			r.Event(instance,
				corev1.EventTypeWarning,
				"removing annotations",
				fmt.Sprintf("Failed to remove annotations: %s", err))
			return ctrl.Result{}, fmt.Errorf("error when removing annotations: %v", err)
		}
		r.Event(instance,
			corev1.EventTypeNormal,
			"Deleted",
			"Kidle annotations on deployment are deleted")

		// All is OK, remove the finalizer
		if err := r.removeFinalizer(ctx, instance); err != nil {
			r.Event(instance,
				corev1.EventTypeWarning,
				"deleting finalizer",
				fmt.Sprintf("Failed to delete finalizer: %s", err))
			return ctrl.Result{}, fmt.Errorf("error when deleting finalizer: %v", err)
		}
		r.Event(instance,
			corev1.EventTypeNormal,
			"Deleted",
			"Object finalizer is deleted")
		return ctrl.Result{}, nil
	}

	// Wakeup deployment
	if idler.NeedWakeup(instance) {
		replicas, err := idler.Wakeup(ctx)
		if err != nil {
			r.Event(instance,
				corev1.EventTypeWarning,
				"ScalingDeployment",
				fmt.Sprintf("Failed to wake up deployment %s: %s", instance.Spec.IdlingResourceRef.Name, err))
			return ctrl.Result{}, fmt.Errorf("error during waking up: %v", err)
		}
		r.Event(instance,
			corev1.EventTypeNormal,
			"ScalingDeployment",
			fmt.Sprintf("Scaled to %d", *replicas))
		return ctrl.Result{}, nil
	}

	// Idle Deployment
	if idler.NeedIdle(instance) {
		if err := idler.Idle(ctx); err != nil {
			r.Event(instance,
				corev1.EventTypeWarning,
				"ScalingDeployment",
				fmt.Sprintf("Failed to idle deployment %s: %s", instance.Spec.IdlingResourceRef.Name, err))
			return ctrl.Result{}, fmt.Errorf("error during idling: %v", err)
		}
		r.Event(instance,
			corev1.EventTypeNormal,
			"ScalingDeployment",
			fmt.Sprintf("Scaled to 0"))
		return ctrl.Result{}, nil
	}
	return ctrl.Result{}, nil
}

func (i *DeploymentIdler) NeedIdle(instance *kidlev1beta1.IdlingResource) bool {
	return instance.Spec.Idle && *i.Deployment.Spec.Replicas > 0
}

func (i *DeploymentIdler) NeedWakeup(instance *kidlev1beta1.IdlingResource) bool {
	return !instance.Spec.Idle && *i.Deployment.Spec.Replicas == 0
}

func (i *DeploymentIdler) SetReference(ctx context.Context, instanceName string) error {
	if !k8s.HasAnnotation(&i.Deployment.ObjectMeta, kidlev1beta1.MetadataIdlingResourceReference) {
		i.Log.Info(fmt.Sprintf("Set reference for deployment %v", i.Deployment.Name))

		k8s.AddAnnotation(&i.Deployment.ObjectMeta, kidlev1beta1.MetadataIdlingResourceReference, instanceName)
		if err := i.Update(ctx, i.Deployment); err != nil {
			i.Log.Error(err, "unable to add reference in annotations")
			return err
		}
	}
	return nil
}

func (i *DeploymentIdler) RemoveAnnotations(ctx context.Context) error {
	if k8s.HasAnnotation(&i.Deployment.ObjectMeta, kidlev1beta1.MetadataIdlingResourceReference) ||
		k8s.HasAnnotation(&i.Deployment.ObjectMeta, kidlev1beta1.MetadataPreviousReplicas) {
		i.Log.Info(fmt.Sprintf("Remove annotations for deployment %v", i.Deployment.Name))

		k8s.RemoveAnnotation(&i.Deployment.ObjectMeta, kidlev1beta1.MetadataIdlingResourceReference)
		k8s.RemoveAnnotation(&i.Deployment.ObjectMeta, kidlev1beta1.MetadataPreviousReplicas)
		if err := i.Update(ctx, i.Deployment); err != nil {
			i.Log.Error(err, "unable to remove kidle annotations")
			return err
		}
	}
	return nil
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
