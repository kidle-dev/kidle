package k8s

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AddAnnotation creates or updates an annotation on an object meta.
// The Annotations map is initialized if nil
func AddAnnotation(obj *metav1.ObjectMeta, annotation string, value string) {
	if obj.Annotations == nil {
		obj.Annotations = make(map[string]string)
	}
	obj.Annotations[annotation] = value
}

// RemoveAnnotation removes an annotation on an object meta
func RemoveAnnotation(obj *metav1.ObjectMeta, annotation string) {
	delete(obj.Annotations, annotation)
}

// GetAnnotation safely returns an annotation value if it exists
func GetAnnotation(obj *metav1.ObjectMeta, annotation string) (string, bool) {
	if obj.Annotations == nil {
		return "", false
	}
	value, found := obj.GetAnnotations()[annotation]
	return value, found
}

// HasAnnotation safely checks if an annotation exists
func HasAnnotation(obj *metav1.ObjectMeta, annotation string) bool {
	if obj.Annotations == nil {
		return false
	}
	_, found := obj.GetAnnotations()[annotation]
	return found
}
