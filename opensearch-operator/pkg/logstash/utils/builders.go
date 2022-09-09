package utils

import (
	"crypto/sha1"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	opsterv1 "opensearch.opster.io/api/v1"
)

const (
	LogstashUser                  = "admin"
	EnvLogstashUser               = "LOGSTASHUSER"
	EnvLogstashUserKey            = "useranme"
	EnvLogstashPass               = "LOGSTASHPASS"
	EnvLogstashPassKey            = "password"
	PipelineConfigVolumeName      = "pipeline"
	PipelineConfigVolumeMountPath = "/usr/share/logstash/pipeline"
	DEFAULTLOGSTASHIMAGEURL       = "opensearchproject/logstash-oss-with-opensearch-output-plugin:8.4.0"
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
			EnvLogstashUserKey: LogstashUser,
			EnvLogstashPassKey: password,
		},
	}
}

func buildPipelineInputs(inputs string) string {
	if len(inputs) != 0 {
		return inputs
	}

	var builder strings.Builder
	builder.WriteString("http {\n")
	builder.WriteString("  port => 8080\n")
	builder.WriteString("}\n")
	return builder.String()
}

func buildPipelineOutputs(instance *opsterv1.Logstash) string {
	var builder strings.Builder
	if len(instance.Spec.Config.PipelineConfig.Outputs) == 0 && instance.Spec.Config.OpenSearchClusterRef == nil {
		return "stdout {}\n"
	}
	builder.WriteString(instance.Spec.Config.PipelineConfig.Outputs)
	builder.WriteString("\n")
	builder.WriteString("")
	return builder.String()
}

func BuildConfigMap(instance *opsterv1.Logstash) (*corev1.ConfigMap, string) {
	var builder strings.Builder
	builder.WriteString("input {\n")
	builder.WriteString(buildPipelineInputs(instance.Spec.Config.PipelineConfig.Inputs))
	builder.WriteString("\n}\n")

	builder.WriteString("filter {\n")
	builder.WriteString(instance.Spec.Config.PipelineConfig.Filters)
	builder.WriteString("\n}\n")

	builder.WriteString("output {\n")
	builder.WriteString(buildPipelineOutputs(instance))
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
			"logstash.conf": builder.String(),
		},
	}, hashStr
}

func buildPortArray(instance *opsterv1.Logstash) []corev1.ServicePort {
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
			Ports:                    buildPortArray(instance),
		},
	}
}

func buildEnvValArrayFromSecret(instance *opsterv1.Logstash) []corev1.EnvVar {
	if instance.Spec.Config.OpenSearchClusterRef == nil {
		return make([]corev1.EnvVar, 0)
	}

	secname := GetSecretName(instance.Name)
	res := make([]corev1.EnvVar, 0)

	// username
	tmp := corev1.EnvVar{
		Name: EnvLogstashUser,
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: secname,
				},
				Key: EnvLogstashUserKey,
			},
		},
	}
	res = append(res, tmp)

	// password
	tmp = corev1.EnvVar{
		Name: EnvLogstashPass,
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: secname,
				},
				Key: EnvLogstashPassKey,
			},
		},
	}
	res = append(res, tmp)

	return res
}

func buildContainerPostArray(instance *opsterv1.Logstash) []corev1.ContainerPort {
	res := make([]corev1.ContainerPort, 0)
	portlist := instance.Spec.Config.Ports
	for _, p := range portlist {
		res = append(res, corev1.ContainerPort{
			Name:          fmt.Sprintf("p%d", p),
			ContainerPort: p,
		})
	}
	return res
}

func buildTemplateSpec(instance *opsterv1.Logstash) *corev1.PodTemplateSpec {
	res := instance.Spec.PodTemplate.DeepCopy()

	// labels
	res.Labels = MergeStringMap(res.Labels, GetLabels(instance.Name))
	if len(res.Spec.Containers) == 0 {
		res.Spec.Containers = append(res.Spec.Containers, corev1.Container{})
	}

	// Env
	tmparray := MergeEnvVarArray(instance.Spec.Config.LogstashConfig, buildEnvValArrayFromSecret(instance))
	res.Spec.Containers[0].Env = MergeEnvVarArray(res.Spec.Containers[0].Env, tmparray)

	// Resource.Limits
	if res.Spec.Containers[0].Resources.Limits == nil {
		res.Spec.Containers[0].Resources.Limits = corev1.ResourceList{
			"cpu": resource.Quantity{
				Format: "500m",
			},
			"memory": resource.Quantity{
				Format: "0.5Gi",
			},
		}
	}

	// Resource.Requests
	if res.Spec.Containers[0].Resources.Requests == nil {
		res.Spec.Containers[0].Resources.Requests = corev1.ResourceList{
			"cpu": resource.Quantity{
				Format: "500m",
			},
			"memory": resource.Quantity{
				Format: "0.5Gi",
			},
		}
	}

	// Mounts for configmap
	res.Spec.Volumes = MergeVolumeArrayWithConfigMap(res.Spec.Volumes, GetConfigMapName(instance.Name))
	res.Spec.Containers[0].VolumeMounts = MergeVolumeMountArrayWithConfigMap(res.Spec.Containers[0].VolumeMounts)

	// ports
	res.Spec.Containers[0].Ports = buildContainerPostArray(instance)

	// image
	res.Spec.Containers[0].Image = GetLogstashImageUrl(res.Spec.Containers[0].Image)

	return res
}

func BuildDeployment(instance *opsterv1.Logstash) *appsv1.Deployment {
	dname := GetDeploymentName(instance.Name)
	template := buildTemplateSpec(instance)
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dname,
			Namespace: instance.Namespace,
			Labels:    GetLabels(instance.Name),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &instance.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: GetSelectors(instance.Name),
			},
			Template: *template,
		},
	}

	return deploy
}
