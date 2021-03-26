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
	"github.com/go-logr/logr"
	"github.com/orphaner/kidle/pkg/utils/pointer"
	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"time"

	kidlev1beta1 "github.com/orphaner/kidle/pkg/api/v1beta1"
)

var (
	deployOwnerKey = ".metadata.controller"
	apiGVStr       = kidlev1beta1.GroupVersion.String()
	finalizerName  = kidlev1beta1.GroupVersion.Group + "/finalizer"
)

// IdlingResourceReconciler reconciles a IdlingResource object
type IdlingResourceReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=kidle.beroot.org,resources=idlingresources,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kidle.beroot.org,resources=idlingresources/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get

func (r *IdlingResourceReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("idlingresource", req.NamespacedName)

	// Load the IdlingResource by name and setup finalizer
	var ir kidlev1beta1.IdlingResource
	if err := r.Get(ctx, req.NamespacedName, &ir); err != nil {
		log.Error(err, "unable to read IdlingResource")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	r.finalizer(ctx, log, &ir)

	// Reconcile
	ref := ir.Spec.IdlingResourceRef
	switch ref.Kind {
	case "Deployment":
		var deploy v1.Deployment
		nn := types.NamespacedName{
			Namespace: req.Namespace,
			Name:      ref.Name,
		}
		if err := r.Get(ctx, nn, &deploy); err != nil {
			if errors.IsNotFound(err) {
				return ctrl.Result{RequeueAfter: 2 * time.Second}, nil
			}
			log.Error(err, "unable to read Deployment")
			return ctrl.Result{}, err
		}
		if err := controllerutil.SetControllerReference(&ir, &deploy, r.Scheme); err != nil {
			return ctrl.Result{}, err
		}
		if err := r.Update(ctx, &deploy); err != nil {
			log.Error(err, "unable to update controller reference")
			return ctrl.Result{}, err
		}

		if ir.Spec.Idle && *deploy.Spec.Replicas > 0 {
			deploy.Spec.Replicas = pointer.Int32(0)
			if err := r.Update(ctx, &deploy); err != nil {
				log.Error(err, "unable to downscale deployment")
				return ctrl.Result{}, err
			}
			log.V(1).Info("deployment idled", "name", ref.Name)
		}
		if !ir.Spec.Idle && *deploy.Spec.Replicas == 0 {
			deploy.Spec.Replicas = pointer.Int32(1)
			if err := r.Update(ctx, &deploy); err != nil {
				log.Error(err, "unable to wakeup deployment")
				return ctrl.Result{}, err
			}
			log.V(1).Info("deployment waked up", "name", ref.Name)
		}

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
		if ir.Spec.Idle && *st.Spec.Replicas > 0 {
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

func (r *IdlingResourceReconciler) finalizer(ctx context.Context, log logr.Logger, ir *kidlev1beta1.IdlingResource) (ctrl.Result, error) {

	controllerutil.AddFinalizer(ir, finalizerName)

	if !ir.ObjectMeta.DeletionTimestamp.IsZero() {
		if containsString(ir.GetFinalizers(), finalizerName) {
			// TODO scale back to previous state
			// TODO remove finalizer only if scaled back is successful
			controllerutil.RemoveFinalizer(ir, finalizerName)
		}
	}
	return ctrl.Result{}, nil
}

func (r *IdlingResourceReconciler) SetupWithManager(mgr ctrl.Manager) error {

	if err := mgr.GetFieldIndexer().IndexField(&v1.Deployment{}, deployOwnerKey, func(rawObj runtime.Object) []string {
		// grab the job object, extract the owner...
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
		Owns(&v1.Deployment{}).
		Complete(r)
}

type Idler struct {
}

// Helper functions to check and remove string from a slice of strings.
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}
