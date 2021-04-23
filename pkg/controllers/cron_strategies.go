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
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	CronJobContainerName = "kidlectl"
)

func (r *IdlingResourceReconciler) ReconcileCronStrategies(ctx context.Context, instance *kidlev1beta1.IdlingResource) (ctrl.Result, error) {
	if !hasCronStrategy(instance) {
		return ctrl.Result{}, nil
	}

	// Create dedicated RBAC for the instance
	if err := r.createRBAC(ctx, instance); err != nil {
		r.Event(instance, corev1.EventTypeWarning, "Adding RBAC", fmt.Sprintf("Failed to add RBAC: %s", err))
		return reconcile.Result{}, fmt.Errorf("error when adding RBAC: %v", err)
	}

	// Create idle cronjob RBAC for the instance
	if instance.Spec.IdlingStrategy != nil && instance.Spec.IdlingStrategy.CronStrategy != nil {
		cjName := fmt.Sprintf("kidle-%s-idle", instance.Name)
		key := types.NamespacedName{Namespace: instance.Namespace, Name: cjName}

		if err := r.createCronJob(ctx, instance, key, "idle"); err != nil {
			r.Event(instance, corev1.EventTypeWarning, "Creating idle CronJob", fmt.Sprintf("Failed to create CronJob: %s", err))
			return reconcile.Result{}, fmt.Errorf("error when creating idle CronJob: %v", err)
		}
	}

	// Create wakeup cronjob RBAC for the instance
	if instance.Spec.WakeupStrategy != nil && instance.Spec.WakeupStrategy.CronStrategy != nil {
		cjName := fmt.Sprintf("kidle-%s-wakeup", instance.Name)
		key := types.NamespacedName{Namespace: instance.Namespace, Name: cjName}

		if err := r.createCronJob(ctx, instance, key, "wakeup"); err != nil {
			r.Event(instance, corev1.EventTypeWarning, "Creating wakeup CronJob", fmt.Sprintf("Failed to create CronJob: %s", err))
			return reconcile.Result{}, fmt.Errorf("error when creating wakeup CronJob: %v", err)
		}
	}

	return ctrl.Result{}, nil
}

func (r *IdlingResourceReconciler) createCronJob(ctx context.Context, instance *kidlev1beta1.IdlingResource, key types.NamespacedName, command string) error {
	cronJob := &v1beta1.CronJob{}
	if err := r.Get(ctx, key, cronJob); err != nil {
		if errors.IsNotFound(err) {
			cj := NewCronJob(key)
			setCronjobValues(cj, instance, "idle")
			if err := controllerutil.SetControllerReference(instance, cj, r.Scheme); err != nil {
				return fmt.Errorf("unable to set controller reference for cronJob: %v", err)
			}
			if err := r.Create(ctx, cj); err != nil {
				return fmt.Errorf("unable to create cronJob: %v", err)
			}
		} else {
			return fmt.Errorf("unable to get cronJob: %v", err)
		}
	}

	if needCronjobValues(cronJob, instance, command) {
		setCronjobValues(cronJob, instance, command)
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

func needCronjobValues(cronJob *batchv1beta1.CronJob, instance *kidlev1beta1.IdlingResource, command string) bool {
	if cronJob.Spec.Suspend != pointer.Bool(false) {
		return true
	}

	if cronJob.Spec.Schedule != instance.Spec.IdlingStrategy.CronStrategy.Schedule {
		return true
	}

	container := k8s.ContainersToMap(cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers)[CronJobContainerName]
	if container.Image != "kidle/kidlectl:latest" {
		return true
	}
	if len(container.Args) != 5 ||
		container.Args[0] != command ||
		container.Args[1] != "--namespace" ||
		container.Args[2] != instance.Namespace ||
		container.Args[3] != "--name" ||
		container.Args[4] != instance.Name {
		return true
	}

	return false
}

func setCronjobValues(cronJob *batchv1beta1.CronJob, instance *kidlev1beta1.IdlingResource, command string) {
	cronJob.Spec.Suspend = pointer.Bool(false)
	cronJob.Spec.Schedule = instance.Spec.IdlingStrategy.CronStrategy.Schedule
	container := k8s.ContainersToMap(cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers)["kidlectl"]

	container.Image = "kidle/kidlectl:latest"
	container.Args = []string{
		command,
		"--namespace",
		instance.Namespace,
		"--name",
		instance.Name,
	}
}

func (r *IdlingResourceReconciler) createRBAC(ctx context.Context, instance *kidlev1beta1.IdlingResource) error {
	saName := fmt.Sprintf("kidle-sa-%s", instance.Name)
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

	roleName := fmt.Sprintf("kidle-role-%s", instance.Name)
	role := &rbacv1.Role{}
	roleKey := types.NamespacedName{Namespace: instance.Namespace, Name: roleName}
	if err := r.Get(ctx, roleKey, role); err != nil {
		if errors.IsNotFound(err) {
			role = &rbacv1.Role{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: instance.Namespace,
					Name:      roleName,
				},
				Rules: []rbacv1.PolicyRule{{
					Verbs:         []string{"get", "patch"},
					APIGroups:     []string{"kidle.beroot.org"},
					Resources:     []string{"idlingresources"},
					ResourceNames: []string{instance.Name},
				}},
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
	}

	rbName := fmt.Sprintf("kidle-rb-%s", instance.Name)
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
					APIGroup: sa.GroupVersionKind().Group,
					Kind:     "ServiceAccount",
					Name:     sa.Name,
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
	}

	return nil
}
