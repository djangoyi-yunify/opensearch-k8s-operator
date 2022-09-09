package utils

import (
	corev1 "k8s.io/api/core/v1"
)

// merge two maps together, new item is added or updated
func MergeStringMap(orgmap map[string]string, newmap map[string]string) map[string]string {
	if orgmap == nil {
		orgmap = make(map[string]string)
	}
	for k, v := range newmap {
		orgmap[k] = v
	}
	return orgmap
}

func MergeEnvVarArray(orgarr []corev1.EnvVar, newarr []corev1.EnvVar) []corev1.EnvVar {
	if orgarr == nil {
		orgarr = make([]corev1.EnvVar, 0)
	}
	res := orgarr
	var find bool
	for _, a := range newarr {
		find = false
		for k, b := range orgarr {
			if a.Name == b.Name {
				find = true
				res[k] = a
				break
			}
		}
		if !find {
			res = append(res, a)
		}
	}
	return res
}

func MergeVolumeArrayWithConfigMap(orgarr []corev1.Volume, cmname string) []corev1.Volume {
	if orgarr == nil {
		orgarr = make([]corev1.Volume, 0)
	}
	res := orgarr
	var find bool = false
	for _, v := range orgarr {
		if v.Name == PipelineConfigVolumeName {
			find = true
			break
		}
	}
	if !find {
		res = append(res, corev1.Volume{
			Name: PipelineConfigVolumeName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: cmname,
					},
				},
			},
		})
	}

	return res
}

func MergeVolumeMountArrayWithConfigMap(orgarr []corev1.VolumeMount) []corev1.VolumeMount {
	if orgarr == nil {
		orgarr = make([]corev1.VolumeMount, 0)
	}
	res := orgarr
	var find bool = false
	for _, v := range orgarr {
		if v.Name == PipelineConfigVolumeName {
			find = true
			break
		}
	}
	if !find {
		res = append(res, corev1.VolumeMount{
			Name:      PipelineConfigVolumeName,
			MountPath: PipelineConfigVolumeMountPath,
		})
	}
	return res
}

func GetLogstashImageUrl(oldurl string) string {
	if len(oldurl) == 0 {
		return DEFAULTLOGSTASHIMAGEURL
	}

	return oldurl
}
