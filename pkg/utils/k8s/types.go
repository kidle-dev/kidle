package k8s

import (
	"github.com/orphaner/kidle/pkg/api/v1beta1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

func ToNamespacedName(req *ctrl.Request, ref *v1beta1.CrossVersionObjectReference) types.NamespacedName {
	return types.NamespacedName{
		Namespace: req.Namespace,
		Name:      ref.Name,
	}
}
