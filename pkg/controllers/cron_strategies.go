package controllers

import (
	"context"
	"fmt"
	kidlev1beta1 "github.com/orphaner/kidle/pkg/api/v1beta1"
	"github.com/orphaner/kidle/pkg/utils/k8s"
	"github.com/orphaner/kidle/pkg/utils/pointer"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/api/batch/v1beta1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	KidlectlImage        = "kidle/kidlectl:latest"
	CronJobContainerName = "kidlectl"
	CommandIdle          = "idle"
	CommandWakeup        = "wakeup"
)

type CronJobValues struct {
	key          types.NamespacedName
	instanceName string
	strategy     *kidlev1beta1.CronStrategy
	command      string
}

func (r *IdlingResourceReconciler) ReconcileCronStrategies(ctx context.Context, instance *kidlev1beta1.IdlingResource) (ctrl.Result, error) {
	// Create dedicated RBAC for the instance
	if err := r.createRBAC(ctx, instance); err != nil {
		r.Event(instance, corev1.EventTypeWarning, "Adding RBAC", fmt.Sprintf("Failed to add RBAC: %s", err))
		return reconcile.Result{}, fmt.Errorf("error when adding RBAC: %v", err)
	}

	cjIdleKey := types.NamespacedName{
		Namespace: instance.Namespace,
		Name:      k8s.ToDNSName("kidle", instance.Name, CommandIdle),
	}

	// Create or update idle cronjob for the instance
	if instance.Spec.IdlingStrategy != nil && instance.Spec.IdlingStrategy.CronStrategy != nil {
		cjIdleValues := &CronJobValues{
			key:          cjIdleKey,
			instanceName: instance.Name,
			command:      CommandIdle,
			strategy:     instance.Spec.IdlingStrategy.CronStrategy,
		}
		if err := r.createOrUpdateCronJob(ctx, instance, cjIdleValues); err != nil {
			r.Event(instance, corev1.EventTypeWarning, "Creating idle CronJob", fmt.Sprintf("Failed to create CronJob: %s", err))
			return reconcile.Result{}, fmt.Errorf("error when creating idle CronJob: %v", err)
		} else {
			r.Event(instance, corev1.EventTypeNormal, "Creating idle CronJob", "Created")
		}
	} else {
		// Delete the idle cronjob if necessary
		cronJob := &v1beta1.CronJob{}
		if err := r.Get(ctx, cjIdleKey, cronJob); err == nil {
			if err := r.Delete(ctx, cronJob, client.PropagationPolicy(metav1.DeletePropagationBackground)); client.IgnoreNotFound(err) != nil {
				r.Event(instance, corev1.EventTypeWarning, "Deleting idle CronJob", fmt.Sprintf("Failed to delete CronJob: %s", err))
				return reconcile.Result{}, fmt.Errorf("error when deleting idle CronJob: %v", err)
			} else {
				r.Event(instance, corev1.EventTypeNormal, "Deleting idle CronJob", "Deleted")
			}
		}
	}

	cjWakeupKey := types.NamespacedName{
		Namespace: instance.Namespace,
		Name:      k8s.ToDNSName("kidle", instance.Name, CommandWakeup),
	}

	// Create wakeup cronjob RBAC for the instance
	if instance.Spec.WakeupStrategy != nil && instance.Spec.WakeupStrategy.CronStrategy != nil {
		cjValues := &CronJobValues{
			key:          cjWakeupKey,
			instanceName: instance.Name,
			command:      CommandWakeup,
			strategy:     instance.Spec.WakeupStrategy.CronStrategy,
		}
		if err := r.createOrUpdateCronJob(ctx, instance, cjValues); err != nil {
			r.Event(instance, corev1.EventTypeWarning, "Creating wakeup CronJob", fmt.Sprintf("Failed to create CronJob: %s", err))
			return reconcile.Result{}, fmt.Errorf("error when creating wakeup CronJob: %v", err)
		} else {
			r.Event(instance, corev1.EventTypeNormal, "Creating wakeup CronJob", "Created")
		}
	} else {
		// Delete the wakeup cronjob if necessary
		cronJob := &v1beta1.CronJob{}
		if err := r.Get(ctx, cjWakeupKey, cronJob); err == nil {
			if err := r.Delete(ctx, cronJob, client.PropagationPolicy(metav1.DeletePropagationBackground)); client.IgnoreNotFound(err) != nil {
				r.Event(instance, corev1.EventTypeWarning, "Deleting wakeup CronJob", fmt.Sprintf("Failed to delete CronJob: %s", err))
				return reconcile.Result{}, fmt.Errorf("error when deleting wakeup CronJob: %v", err)
			} else {
				r.Event(instance, corev1.EventTypeNormal, "Deleting wakeup CronJob", "Deleted")
			}
		}
	}

	return ctrl.Result{}, nil
}

func (r *IdlingResourceReconciler) createOrUpdateCronJob(ctx context.Context, instance *kidlev1beta1.IdlingResource, cjValues *CronJobValues) error {
	cronJob := &v1beta1.CronJob{}
	if err := r.Get(ctx, cjValues.key, cronJob); err != nil {
		if errors.IsNotFound(err) {
			cj := NewCronJob(cjValues.key)
			setCronjobValues(cj, cjValues)
			if err := controllerutil.SetControllerReference(instance, cj, r.Scheme); err != nil {
				return fmt.Errorf("unable to set controller reference for cronJob: %v", err)
			}
			if err := r.Create(ctx, cj); err != nil {
				return fmt.Errorf("unable to create cronJob: %v", err)
			}
			return nil
		} else {
			return fmt.Errorf("unable to get cronJob: %v", err)
		}
	}

	if cronJobNeedChanges(cronJob, cjValues) {
		setCronjobValues(cronJob, cjValues)
		if err := r.Update(ctx, cronJob); err != nil {
			return fmt.Errorf("unable to update cronJob: %v", err)
		}
	}
	return nil
}

func NewCronJob(key types.NamespacedName) *batchv1beta1.CronJob {
	var cj = &batchv1beta1.CronJob{
		TypeMeta: metav1.TypeMeta{
			Kind:       "CronJob",
			APIVersion: "batch/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      key.Name,
			Namespace: key.Namespace,
		},
		Spec: batchv1beta1.CronJobSpec{
			JobTemplate: batchv1beta1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{},
						Spec: corev1.PodSpec{
							RestartPolicy: corev1.RestartPolicyOnFailure,
							Containers: []corev1.Container{
								{
									Name: CronJobContainerName,
								},
							},
						},
					},
				},
			},
		},
	}
	return cj
}

func cronJobNeedChanges(cronJob *batchv1beta1.CronJob, cjValues *CronJobValues) bool {
	if cronJob.Spec.JobTemplate.Spec.Template.Spec.ServiceAccountName != getSaName(cjValues.instanceName) {
		return true
	}

	if cronJob.Spec.Suspend != pointer.Bool(false) {
		return true
	}
	if cronJob.Spec.Schedule != cjValues.strategy.Schedule {
		return true
	}

	container := k8s.ContainersToMap(cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers)[CronJobContainerName]
	if container.Image != KidlectlImage {
		return true
	}
	if len(container.Args) != 2 ||
		container.Args[0] != cjValues.command ||
		container.Args[1] != cjValues.key.Name {
		return true
	}
	return false
}

func setCronjobValues(cronJob *batchv1beta1.CronJob, cjValues *CronJobValues) {
	cronJob.Spec.JobTemplate.Spec.Template.Spec.ServiceAccountName = getSaName(cjValues.instanceName)

	cronJob.Spec.Suspend = pointer.Bool(false)
	cronJob.Spec.Schedule = cjValues.strategy.Schedule

	container := k8s.ContainersToMap(cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers)[CronJobContainerName]
	container.Image = KidlectlImage
	container.Args = []string{
		cjValues.command,
		cjValues.instanceName,
	}
	k8s.SetContainer(cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers, &container)
}

func getSaName(instanceName string) string {
	return k8s.ToDNSName("kidle", instanceName, "sa")
}

func (r *IdlingResourceReconciler) createRBAC(ctx context.Context, instance *kidlev1beta1.IdlingResource) error {
	saName := getSaName(instance.Name)
	sa := &corev1.ServiceAccount{}
	saKey := types.NamespacedName{Namespace: instance.Namespace, Name: saName}
	if err := r.Get(ctx, saKey, sa); err != nil {
		if errors.IsNotFound(err) {
			sa = &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: instance.Namespace,
					Name:      saName,
				},
			}
			if err := controllerutil.SetControllerReference(instance, sa, r.Scheme); err != nil {
				return fmt.Errorf("unable to set controller reference for sa: %v", err)
			}
			if err := r.Create(ctx, sa); err != nil {
				return fmt.Errorf("unable to create sa: %v", err)
			}
		} else {
			return fmt.Errorf("unable to get sa: %v", err)
		}
	}

	roleName := k8s.ToDNSName("kidle", instance.Name, "role")
	role := &rbacv1.Role{}
	roleKey := types.NamespacedName{Namespace: instance.Namespace, Name: roleName}
	policyRule := rbacv1.PolicyRule{
		Verbs:         []string{"get", "patch", "update"},
		APIGroups:     []string{"kidle.beroot.org"},
		Resources:     []string{"idlingresources"},
		ResourceNames: []string{instance.Name},
	}
	if err := r.Get(ctx, roleKey, role); err != nil {
		if errors.IsNotFound(err) {
			role = &rbacv1.Role{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: instance.Namespace,
					Name:      roleName,
				},
				Rules: []rbacv1.PolicyRule{policyRule},
			}
			if err = controllerutil.SetControllerReference(instance, role, r.Scheme); err != nil {
				return fmt.Errorf("unable to set controller reference for role: %v", err)
			}
			if err = r.Create(ctx, role); err != nil {
				return fmt.Errorf("unable to create role: %v", err)
			}
		} else {
			return fmt.Errorf("unable to get role: %v", err)
		}
	} else {
		role.Rules[0] = policyRule
		if err := r.Update(ctx, role) ; err != nil {
			return fmt.Errorf("unable to update role: %v", err)
		}
	}

	rbName := k8s.ToDNSName("kidle", instance.Name, "rb")
	rb := &rbacv1.RoleBinding{}
	rbKey := types.NamespacedName{Namespace: instance.Namespace, Name: rbName}
	if err := r.Get(ctx, rbKey, rb); err != nil {
		if errors.IsNotFound(err) {
			rb := &rbacv1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: instance.Namespace,
					Name:      rbName,
				},
				Subjects: []rbacv1.Subject{{
					Kind: "ServiceAccount",
					Name: sa.Name,
				}},
				RoleRef: rbacv1.RoleRef{
					APIGroup: role.GroupVersionKind().Group,
					Kind:     "Role",
					Name:     role.Name,
				},
			}
			if err = controllerutil.SetControllerReference(instance, rb, r.Scheme); err != nil {
				return fmt.Errorf("unable to set controller reference for rolebinding: %v", err)
			}
			if err = r.Create(ctx, rb); err != nil {
				return fmt.Errorf("unable to create rolebinding: %v", err)
			}
		} else {
			return fmt.Errorf("unable to get rolebinding: %v", err)
		}
	} else {
		// TODO update if necessary
	}
	return nil
}
