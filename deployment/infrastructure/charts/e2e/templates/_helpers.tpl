{{/*
Expand the name of the chart.
*/}}
{{- define "e2e.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "e2e.fullname" -}}
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

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "e2e.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "e2e.labels" -}}
helm.sh/chart: {{ include "e2e.chart" . }}
{{ include "e2e.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "e2e.selectorLabels" -}}
app.kubernetes.io/name: {{ include "e2e.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app: e2e
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "e2e.serviceAccountName" -}}
{{- default (include "e2e.fullname" .) .Values.serviceAccount.name }}
{{- end }}

{{/*
Image pull policy for the specified image tag.
*/}}
{{- define "e2e.imagePullPolicy" -}}
{{- if .Values.image.pullPolicy -}}
{{ .Values.image.pullPolicy }}
{{- else if eq .Values.image.tag "latest" -}}
Always
{{- else -}}
IfNotPresent
{{- end -}}
{{- end }}
