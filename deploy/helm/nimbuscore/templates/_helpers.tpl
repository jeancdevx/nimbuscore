{{- define "nimbuscore.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "nimbuscore.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{- define "nimbuscore.labels" -}}
helm.sh/chart: {{ include "nimbuscore.name" . }}-{{ .Chart.Version | replace "+" "_" }}
{{ include "nimbuscore.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{- define "nimbuscore.selectorLabels" -}}
app.kubernetes.io/name: {{ include "nimbuscore.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "nimbuscore.api.labels" -}}
{{ include "nimbuscore.labels" . }}
app.kubernetes.io/component: api
{{- end }}

{{- define "nimbuscore.operator.labels" -}}
{{ include "nimbuscore.labels" . }}
app.kubernetes.io/component: operator
{{- end }}

{{- define "nimbuscore.manager.labels" -}}
{{ include "nimbuscore.labels" . }}
app.kubernetes.io/component: workspace-manager
{{- end }}

{{- define "nimbuscore.api.fullname" -}}
{{ include "nimbuscore.fullname" . }}-api
{{- end }}

{{- define "nimbuscore.operator.fullname" -}}
{{ include "nimbuscore.fullname" . }}-operator
{{- end }}

{{- define "nimbuscore.manager.fullname" -}}
{{ include "nimbuscore.fullname" . }}-workspace-manager
{{- end }}

{{- define "nimbuscore.db.url" -}}
{{- if .Values.postgresql.enabled -}}
postgres://{{ .Values.postgresql.auth.username }}:{{ .Values.postgresql.auth.password }}@{{ .Release.Name }}-postgresql:5432/{{ .Values.postgresql.auth.database }}?sslmode=disable
{{- else -}}
{{ .Values.config.database.url }}
{{- end -}}
{{- end -}}

{{- define "nimbuscore.redis.url" -}}
{{- if .Values.redis.enabled -}}
redis://:{{ .Values.redis.auth.password }}@{{ .Release.Name }}-redis-master:6379/0
{{- else -}}
{{ .Values.config.redis.url }}
{{- end -}}
{{- end -}}

{{- define "nimbuscore.rmq.url" -}}
{{- if .Values.rabbitmq.enabled -}}
amqp://{{ .Values.rabbitmq.auth.username }}:{{ .Values.rabbitmq.auth.password }}@{{ .Release.Name }}-rabbitmq:5672
{{- else -}}
{{ .Values.config.rabbitmq.url }}
{{- end -}}
{{- end -}}
