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
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
	"time"
)

type DeploymentIdler struct {
	client.Client
	Log        logr.Logger
	deployment *v1.Deployment
}

func (r *IdlingResourceReconciler) ReconcileDeployment(ctx context.Context, instance *kidlev1beta1.IdlingResource) (ctrl.Result, error) {

	var deploy v1.Deployment
	if err := r.Get(ctx, types.NamespacedName{Namespace: instance.Namespace, Name: instance.Spec.IdlingResourceRef.Name}, &deploy); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{RequeueAfter: 2 * time.Second}, nil
		}
		return ctrl.Result{}, fmt.Errorf("unable to read Deployment: %v", err)
	}

	idler := &DeploymentIdler{
		Client:     r.Client,
		Log:        r.Log,
		deployment: &deploy,
	}

	// Add a reference on the deployment
	err := idler.setReference(ctx, instance.Name)
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
		if err := idler.removeAnnotations(ctx); err != nil {
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
	if !instance.Spec.Idle && *deploy.Spec.Replicas == 0 {
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
	if instance.Spec.Idle && *deploy.Spec.Replicas > 0 {
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

func (i *DeploymentIdler) setReference(ctx context.Context, instanceName string) error {
	if !k8s.HasAnnotation(&i.deployment.ObjectMeta, kidlev1beta1.MetadataIdlingResourceReference) {
		i.Log.Info(fmt.Sprintf("Set reference for deployment %v", i.deployment.Name))

		k8s.AddAnnotation(&i.deployment.ObjectMeta, kidlev1beta1.MetadataIdlingResourceReference, instanceName)
		if err := i.Update(ctx, i.deployment); err != nil {
			i.Log.Error(err, "unable to add reference in annotations")
			return err
		}
	}
	return nil
}

func (i *DeploymentIdler) removeAnnotations(ctx context.Context) error {
	if k8s.HasAnnotation(&i.deployment.ObjectMeta, kidlev1beta1.MetadataIdlingResourceReference) ||
		k8s.HasAnnotation(&i.deployment.ObjectMeta, kidlev1beta1.MetadataPreviousReplicas) {
		i.Log.Info(fmt.Sprintf("Remove annotations for deployment %v", i.deployment.Name))

		k8s.RemoveAnnotation(&i.deployment.ObjectMeta, kidlev1beta1.MetadataIdlingResourceReference)
		k8s.RemoveAnnotation(&i.deployment.ObjectMeta, kidlev1beta1.MetadataPreviousReplicas)
		if err := i.Update(ctx, i.deployment); err != nil {
			i.Log.Error(err, "unable to remove kidle annotations")
			return err
		}
	}
	return nil
}

func (i *DeploymentIdler) Idle(ctx context.Context) error {
	k8s.AddAnnotation(&i.deployment.ObjectMeta, kidlev1beta1.MetadataPreviousReplicas, strconv.Itoa(int(*i.deployment.Spec.Replicas)))
	if i.deployment.Spec.Replicas != pointer.Int32(0) {
		i.deployment.Spec.Replicas = pointer.Int32(0)
		if err := i.Update(ctx, i.deployment); err != nil {
			i.Log.Error(err, "unable to downscale deployment")
			return err
		}
		i.Log.V(1).Info("deployment idled", "name", i.deployment.Name)
	} else {
		i.Log.V(2).Info("deployment already idled", "name", i.deployment.Name)
	}
	return nil
}

func (i *DeploymentIdler) Wakeup(ctx context.Context) (*int32, error) {
	previousReplicas := pointer.Int32(1)

	if metadataPreviousReplicas, found := k8s.GetAnnotation(&i.deployment.ObjectMeta, kidlev1beta1.MetadataPreviousReplicas); found {
		if v, err := strconv.Atoi(metadataPreviousReplicas); err != nil {
			return nil, err
		} else {
			previousReplicas = pointer.Int32(int32(v))
		}
	}
	if i.deployment.Spec.Replicas != previousReplicas {
		i.deployment.Spec.Replicas = previousReplicas
		if err := i.Update(ctx, i.deployment); err != nil {
			i.Log.Error(err, "unable to wakeup deployment")
			return nil, err
		}
		i.Log.V(1).Info("deployment waked up", "name", i.deployment.Name)
	} else {
		i.Log.V(2).Info("deployment already waked up", "name", i.deployment.Name)
	}
	return previousReplicas, nil
}
