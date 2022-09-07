package utils

import (
	"crypto/sha1"
	"strings"

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

func BuildConfigMap(instance *opsterv1.Logstash) (*corev1.ConfigMap, string) {
	var builder strings.Builder
	builder.WriteString("input {\n")
	builder.WriteString(instance.Spec.Config.PipelineConfig.Inputs)
	builder.WriteString("\n}\n")

	builder.WriteString("filter {\n")
	builder.WriteString(instance.Spec.Config.PipelineConfig.Filters)
	builder.WriteString("\n}\n")

	builder.WriteString("output {\n")
	builder.WriteString(instance.Spec.Config.PipelineConfig.Outputs)
	builder.WriteString("\n}")

	hash := sha1.New()
	hash.Write([]byte(builder.String()))
	hashStr := string(hash.Sum([]byte("")))

	cmname := GetConfigMapName(instance.Name)
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cmname,
			Namespace: instance.Namespace,
		},
		Data: map[string]string{
			"pipelines.yml": builder.String(),
		},
	}, hashStr
}
