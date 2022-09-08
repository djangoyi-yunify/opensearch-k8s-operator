package utils

import (
	"crypto/sha1"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
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

func BuildPortArray(instance *opsterv1.Logstash) []corev1.ServicePort {
	res := make([]corev1.ServicePort, 0)
	portlist := instance.Spec.Config.Ports
	for _, p := range portlist {
		res = append(res, corev1.ServicePort{
			Name:     fmt.Sprintf("p%d", p),
			Protocol: "TCP",
			Port:     p,
			TargetPort: intstr.IntOrString{
				IntVal: p,
			},
		})
	}
	return res
}

func BuildService(instance *opsterv1.Logstash) *corev1.Service {
	svname := GetServiceName(instance.Name)
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      svname,
			Namespace: instance.Namespace,
		},
		Spec: corev1.ServiceSpec{
			PublishNotReadyAddresses: true,
			Selector:                 GetSelectors(instance.Name),
			Ports:                    BuildPortArray(instance),
		},
	}
}
