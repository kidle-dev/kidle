/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	kidlev1beta1 "github.com/orphaner/kidle/pkg/api/v1beta1"
	"github.com/orphaner/kidle/pkg/utils/array"
	"github.com/orphaner/kidle/pkg/utils/pointer"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var (
	deployOwnerKey = ".metadata.controller"
	apiGVStr       = kidlev1beta1.GroupVersion.String()
)

// IdlingResourceReconciler reconciles a IdlingResource object
type IdlingResourceReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	record.EventRecorder
}

// +kubebuilder:rbac:groups=kidle.beroot.org,resources=idlingresources,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kidle.beroot.org,resources=idlingresources/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get

func (r *IdlingResourceReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("idlingresource", req.NamespacedName)

	log.V(1).Info("Starting reconcile loop")
	defer log.V(1).Info("Finish reconcile loop")

	var instance kidlev1beta1.IdlingResource
	err := r.Get(ctx, req.NamespacedName, &instance)

	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// Add finalizer for any kidleable kind
	if !instance.HasFinalizer(kidlev1beta1.IdlingResourceFinalizerName) {
		r.Log.Info(fmt.Sprintf("AddFinalizer for %v", req.NamespacedName))
		err := r.addFinalizer(ctx, &instance)
		if err != nil {
			r.Event(&instance, corev1.EventTypeWarning, "Adding finalizer", fmt.Sprintf("Failed to add finalizer: %s", err))
			return reconcile.Result{}, fmt.Errorf("error when adding finalizer: %v", err)
		}
		r.Event(&instance, corev1.EventTypeNormal, "Added", "Object finalizer is added")
	}

	// Reconcile
	ref := instance.Spec.IdlingResourceRef
	switch ref.Kind {
	case "Deployment":
		return r.ReconcileDeployment(ctx, &instance)

	case "StatefulSet":
		var st v1.StatefulSet
		nn := types.NamespacedName{
			Namespace: req.Namespace,
			Name:      ref.Name,
		}
		if err := r.Get(ctx, nn, &st); err != nil {
			log.Error(err, "unable to read StatefulSet")
			return ctrl.Result{}, err
		}
		if instance.Spec.Idle && *st.Spec.Replicas > 0 {
			st.Spec.Replicas = pointer.Int32(0)
			if err := r.Update(ctx, &st); err != nil {
				log.Error(err, "unable to downscale statefulset")
				return ctrl.Result{}, err
			}
			log.V(1).Info("statefulset idled", "name", ref.Name)
		}
	}

	return ctrl.Result{}, nil
}

func (r *IdlingResourceReconciler) addFinalizer(ctx context.Context, instance *kidlev1beta1.IdlingResource) error {
	controllerutil.AddFinalizer(instance, kidlev1beta1.IdlingResourceFinalizerName)
	err := r.Update(ctx, instance)
	if err != nil {
		return fmt.Errorf("failed to update idling resource finalizer: %v", err)
	}
	return nil
}

func (r *IdlingResourceReconciler) removeFinalizer(ctx context.Context, instance *kidlev1beta1.IdlingResource) error {
	if array.ContainsString(instance.GetFinalizers(), kidlev1beta1.IdlingResourceFinalizerName) {
		controllerutil.RemoveFinalizer(instance, kidlev1beta1.IdlingResourceFinalizerName)
		err := r.Update(ctx, instance)
		if err != nil {
			return fmt.Errorf("error when removing idling resource finalizer: %v", err)
		}
	}
	return nil
}

func (r *IdlingResourceReconciler) SetupWithManager(mgr ctrl.Manager) error {

	if err := mgr.GetFieldIndexer().IndexField(&v1.Deployment{}, deployOwnerKey, func(rawObj runtime.Object) []string {
		// grab the deployment object, extract the owner...
		deploy := rawObj.(*v1.Deployment)
		owner := metav1.GetControllerOf(deploy)
		if owner == nil {
			return nil
		}
		// ...make sure it's a Deployment...
		if owner.APIVersion != apiGVStr || owner.Kind != "Deployment" {
			return nil
		}

		// ...and if so, return it
		return []string{owner.Name}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&kidlev1beta1.IdlingResource{}).
		//Owns(&v1.Deployment{}).
		Watches(
			&source.Kind{Type: &v1.Deployment{}},
			&handler.EnqueueRequestsFromMapFunc{
				ToRequests: handler.ToRequestsFunc(r.deploymentForIdlingResourceMapper),
			},
		).
		Complete(r)
}

func (r *IdlingResourceReconciler) deploymentForIdlingResourceMapper(object handler.MapObject) []reconcile.Request {
	ref, found := object.Meta.GetAnnotations()[kidlev1beta1.MetadataIdlingResourceReference]
	if !found {
		return nil
	}

	reqs := make([]reconcile.Request, 1)
	reqs[0].NamespacedName.Name = ref
	reqs[0].NamespacedName.Namespace = object.Meta.GetNamespace()
	r.Log.Info("requesting reconciliation", "IdlingResource", reqs[0].NamespacedName)
	return reqs
}
