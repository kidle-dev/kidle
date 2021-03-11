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
	"github.com/prometheus/common/log"
	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kidlev1beta1 "github.com/orphaner/kidle/api/v1beta1"
)

// IdlingResourceReconciler reconciles a IdlingResource object
type IdlingResourceReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=kidle.beroot.org,resources=idlingresources,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kidle.beroot.org,resources=idlingresources/status,verbs=get;update;patch

func (r *IdlingResourceReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	_ = r.Log.WithValues("idlingresource", req.NamespacedName)

	// your logic here
	var ir kidlev1beta1.IdlingResource
	if err := r.Get(ctx, req.NamespacedName, &ir); err != nil {
		log.Error("unable to read IdlingResource")
		return ctrl.Result{}, err
	}

	ref := ir.Spec.IdlingResourceRef
	switch ref.Kind {
	case "Deployment":
		var deploy v1.Deployment
		nn := types.NamespacedName{
			Namespace: req.Namespace,
			Name:      ref.Name,
		}
		if err := r.Get(ctx, nn, &deploy); err != nil {
			log.Error("unable to read Deployment")
			return ctrl.Result{}, err
		}
		if *ir.Spec.Idle && *deploy.Spec.Replicas > 0 {
			zero := int32(0)
			deploy.Spec.Replicas = &zero
			if err := r.Client.Update(ctx, &deploy); err != nil {
				log.Error("unable to downscale deployment")
				return ctrl.Result{}, err
			}
		}
	}

	return ctrl.Result{}, nil
}

func (r *IdlingResourceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kidlev1beta1.IdlingResource{}).
		Complete(r)
}

type Idler struct {
}
