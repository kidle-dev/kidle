package k8s

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AddAnnotation creates or updates an annotation on an object meta.
// The Annotations map is initialized if nil
func AddAnnotation(obj metav1.Object, annotation string, value string) {
	if obj.GetAnnotations() == nil {
		obj.SetAnnotations(make(map[string]string))
	}
	obj.GetAnnotations()[annotation] = value
}

// RemoveAnnotation removes an annotation on an object meta
func RemoveAnnotation(obj metav1.Object, annotation string) {
	delete(obj.GetAnnotations(), annotation)
}

// GetAnnotation safely returns an annotation value if it exists
func GetAnnotation(obj metav1.Object, annotation string) (string, bool) {
	if obj.GetAnnotations() == nil {
		return "", false
	}
	value, found := obj.GetAnnotations()[annotation]
	return value, found
}

// HasAnnotation safely checks if an annotation exists
func HasAnnotation(obj metav1.Object, annotation string) bool {
	if obj.GetAnnotations() == nil {
		return false
	}
	_, found := obj.GetAnnotations()[annotation]
	return found
}
