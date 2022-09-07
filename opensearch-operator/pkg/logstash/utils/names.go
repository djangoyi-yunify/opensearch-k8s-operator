package utils

import "strings"

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
