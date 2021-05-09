package controllers

import corev1 "k8s.io/api/core/v1"

func IsPodReady(pod *corev1.Pod) bool {
	if pod.DeletionTimestamp != nil {
		return false
	}

	if len(pod.Status.ContainerStatuses) < 1 {
		return false
	}

	if !pod.Status.ContainerStatuses[0].Ready {
		return false
	}

	for _, v := range pod.Status.Conditions {
		if v.Type != corev1.PodReady {
			continue
		}
		if v.Status != corev1.ConditionTrue {
			return false
		}
	}

	return true
}
