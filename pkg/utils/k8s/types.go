package k8s

import (
	corev1 "k8s.io/api/core/v1"
)

// ContainersToMap maps an array of containers indexed by the container name
func ContainersToMap(containers []corev1.Container) map[string]corev1.Container {
	result := make(map[string]corev1.Container)
	if len(containers) > 0 {
		for _, c := range containers {
			result[c.Name] = c
		}
	}
	return result
}

// SetContainer replace a container in an array by finding it by its name
func SetContainer(containers []corev1.Container, container *corev1.Container) {
	for k := range containers {
		if containers[k].Name == container.Name {
			containers[k] = *container
			return
		}
	}
}
