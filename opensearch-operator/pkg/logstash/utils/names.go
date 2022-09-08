package utils

import "strings"

const (
	ClusterNameKey = "opensearch.opster.io/logstash"
)

func GetSecretName(lstname string) string {
	var builder strings.Builder
	builder.WriteString("logstash-")
	builder.WriteString(lstname)
	builder.WriteString("-user")
	return builder.String()
}

func GetConfigMapName(lstname string) string {
	var builder strings.Builder
	builder.WriteString("logstash-")
	builder.WriteString(lstname)
	builder.WriteString("-pipelines")
	return builder.String()
}

func GetServiceName(lstname string) string {
	var builder strings.Builder
	builder.WriteString("logstash-")
	builder.WriteString(lstname)
	builder.WriteString("-network")
	return builder.String()
}

func GetLabels(lstname string) map[string]string {
	return map[string]string{
		ClusterNameKey: lstname,
	}
}

func GetSelectors(lstname string) map[string]string {
	return map[string]string{
		ClusterNameKey: lstname,
	}
}
