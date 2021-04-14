package controllers

import (
	"context"
	"fmt"
	kidlev1beta1 "github.com/orphaner/kidle/pkg/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *IdlingResourceReconciler) createRBAC(ctx context.Context, instance *kidlev1beta1.IdlingResource) error {
	saName := fmt.Sprintf("kidle-sa-%s", instance.Name)
	sa := &corev1.ServiceAccount{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: instance.Namespace, Name: saName}, sa); err != nil {
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
	if err := r.Get(ctx, types.NamespacedName{Namespace: instance.Namespace, Name: roleName}, role); err != nil {
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
	if err := r.Get(ctx, types.NamespacedName{Namespace: instance.Namespace, Name: rbName}, rb); err != nil {
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
