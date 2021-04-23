package k8s

import (
	"github.com/orphaner/kidle/pkg/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

func ToNamespacedName(req *ctrl.Request, ref *v1beta1.CrossVersionObjectReference) types.NamespacedName {
	return types.NamespacedName{
		Namespace: req.Namespace,
		Name:      ref.Name,
	}
}

func ContainersToMap(containers []corev1.Container) map[string]corev1.Container {
	result := make(map[string]corev1.Container)
	for _, c := range containers {
		result[c.Name] = c
	}
	return result
}
