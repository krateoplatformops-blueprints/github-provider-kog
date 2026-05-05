{{/*
Expand the name of the chart.
*/}}
{{- define "branchprotection-plugin-chart.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "branchprotection-plugin-chart.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s-plugin" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "branchprotection-plugin-chart.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "branchprotection-plugin-chart.labels" -}}
helm.sh/chart: {{ include "branchprotection-plugin-chart.chart" . }}
{{ include "branchprotection-plugin-chart.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "branchprotection-plugin-chart.selectorLabels" -}}
app.kubernetes.io/name: {{ include "branchprotection-plugin-chart.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "branchprotection-plugin-chart.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "branchprotection-plugin-chart.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{- define "branchprotection.webServiceUrl" -}}
http://{{ include "branchprotection-plugin-chart.fullname" . }}.{{ .Release.Namespace }}.svc.cluster.local:{{ .Values.service.port }}
{{- end -}}
