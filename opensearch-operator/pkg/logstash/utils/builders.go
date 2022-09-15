package utils

import (
	"crypto/sha1"
	"encoding/hex"
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
	EnvLogstashUser               = "LOGSTASHUSER"
	EnvLogstashPass               = "LOGSTASHPASS"
	SecLogstashUserKey            = "username"
	SecLogstashPassKey            = "password"
	PipelineConfigVolumeName      = "pipeline"
	PipelineConfigVolumeMountPath = "/usr/share/logstash/pipeline"
	DEFAULTLOGSTASHIMAGEURL       = "opensearchproject/logstash-oss-with-opensearch-output-plugin:8.4.0"

	ExtOpenSearchUrlProtocol = "https"
	ExtOpenSearchUrlPort     = "9200"

	EnvCmHashKey = "CMHASH"
)

// global info: external opensearch's url
var ExtOpenSearchUrl string
var ExtOpenSearchLogstashUserSecret *corev1.Secret

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

func buildOpenSearchOutput(instance *opsterv1.Logstash) string {
	var builder strings.Builder
	builder.WriteString("opensearch {\n")
	builder.WriteString(fmt.Sprintf("  hosts => [\"%s\"]\n", ExtOpenSearchUrl))
	if ExtOpenSearchLogstashUserSecret != nil {
		builder.WriteString(fmt.Sprintf("  user => \"${%s}\"\n", EnvLogstashUser))
		builder.WriteString(fmt.Sprintf("  password => \"${%s}\"\n", EnvLogstashPass))
		builder.WriteString("  ssl => true\n")
		builder.WriteString("  ssl_certificate_verification => false\n")
	}
	if len(instance.Spec.Config.PipelineConfig.OpenSearchIndex) == 0 {
		builder.WriteString("  index => \"opensearch-logstash-%{+YYYY.MM.dd}\"")
	} else {
		builder.WriteString(instance.Spec.Config.PipelineConfig.OpenSearchIndex)
	}
	builder.WriteString("\n}\n")
	return builder.String()
}

func buildPipelineOutputs(instance *opsterv1.Logstash) string {
	var builder strings.Builder
	// default output: stdout
	if len(instance.Spec.Config.PipelineConfig.Outputs) == 0 && instance.Spec.Config.OpenSearchInfo == nil {
		return "stdout {}\n"
	}
	builder.WriteString(instance.Spec.Config.PipelineConfig.Outputs)
	builder.WriteString("\n")
	if instance.Spec.Config.OpenSearchInfo != nil && len(ExtOpenSearchUrl) != 0 {
		builder.WriteString(buildOpenSearchOutput(instance))
	}
	builder.WriteString("\n")
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
	hashStr := hex.EncodeToString(hash.Sum(nil))

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
	if instance.Spec.Config.OpenSearchInfo == nil {
		return make([]corev1.EnvVar, 0)
	}

	secname := ExtOpenSearchLogstashUserSecret.Name
	res := make([]corev1.EnvVar, 0)

	// username
	tmp := corev1.EnvVar{
		Name: EnvLogstashUser,
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: secname,
				},
				Key: SecLogstashUserKey,
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
				Key: SecLogstashPassKey,
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

func buildTemplateSpec(instance *opsterv1.Logstash, hash string) *corev1.PodTemplateSpec {
	res := instance.Spec.PodTemplate.DeepCopy()

	// labels
	res.Labels = MergeStringMap(res.Labels, GetLabels(instance.Name))
	if len(res.Spec.Containers) == 0 {
		res.Spec.Containers = append(res.Spec.Containers, corev1.Container{})
	}

	// Env
	tmparray := MergeEnvVarArray(instance.Spec.Config.LogstashConfig, buildEnvValArrayFromSecret(instance))
	tmparray = append(tmparray, corev1.EnvVar{
		Name:  EnvCmHashKey,
		Value: hash,
	})

	if len(instance.Spec.Config.Jvm) == 0 {
		tmparray = append(tmparray, corev1.EnvVar{
			Name:  "LS_JAVA_OPTS",
			Value: "-Xms512m -Xmx512m",
		})
	} else {
		tmparray = append(tmparray, corev1.EnvVar{
			Name:  "LS_JAVA_OPTS",
			Value: instance.Spec.Config.Jvm,
		})
	}

	res.Spec.Containers[0].Env = MergeEnvVarArray(res.Spec.Containers[0].Env, tmparray)

	// Resource.Limits
	var reslist corev1.ResourceList
	if res.Spec.Containers[0].Resources.Limits == nil {
		reslist = make(corev1.ResourceList)
		reslist[corev1.ResourceCPU] = resource.MustParse("500m")
		reslist[corev1.ResourceMemory] = resource.MustParse("1024Mi")
		res.Spec.Containers[0].Resources.Limits = reslist
	}

	// Resource.Requests
	if res.Spec.Containers[0].Resources.Requests == nil {
		reslist = make(corev1.ResourceList)
		reslist[corev1.ResourceCPU] = resource.MustParse("500m")
		reslist[corev1.ResourceMemory] = resource.MustParse("1024Mi")
		res.Spec.Containers[0].Resources.Requests = reslist
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

func BuildDeployment(instance *opsterv1.Logstash, hash string) *appsv1.Deployment {
	dname := GetDeploymentName(instance.Name)
	template := buildTemplateSpec(instance, hash)
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

func BuildExtOpenSearchUrl(srv string, ns string) {
	if len(srv) == 0 {
		ExtOpenSearchUrl = ""
	}
	ExtOpenSearchUrl = fmt.Sprintf("%s://%s.%s.svc.cluster.local:%s", ExtOpenSearchUrlProtocol, srv, ns, ExtOpenSearchUrlPort)
}
