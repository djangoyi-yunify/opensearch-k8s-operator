package utils

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	opsterv1 "opensearch.opster.io/api/v1"
)

const (
	LogstashUser = "admin"
)

func BuildSecret(instance *opsterv1.Logstash) *corev1.Secret {
	// for test
	// will find a way to create password
	password := "admin"

	secname := GetSecretName(instance.Name)
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secname,
			Namespace: instance.Namespace,
		},
		StringData: map[string]string{
			"username": LogstashUser,
			"password": password,
		},
	}
}

func BuildConfigMap(instance *opsterv1.Logstash) *corev1.ConfigMap {
	return nil
}
