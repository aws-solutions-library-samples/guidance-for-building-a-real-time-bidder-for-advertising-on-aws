{{/*
Expand the name of the chart.
*/}}
{{- define "load-generator.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "load-generator.fullname" -}}
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
{{- define "load-generator.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "load-generator.labels" -}}
helm.sh/chart: {{ include "load-generator.chart" . }}
{{ include "load-generator.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "load-generator.selectorLabels" -}}
app.kubernetes.io/name: {{ include "load-generator.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app: load-generator
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "load-generator.serviceAccountName" -}}
{{- default (include "load-generator.fullname" .) .Values.serviceAccount.name }}
{{- end }}

{{/*
Image pull policy for the specified image tag.
*/}}
{{- define "load-generator.imagePullPolicy" -}}
{{- if .Values.image.pullPolicy -}}
{{ .Values.image.pullPolicy }}
{{- else if eq .Values.image.tag "latest" -}}
Always
{{- else -}}
IfNotPresent
{{- end -}}
{{- end }}

{{/*
Evaluates and creates a list of targets from the first available source of configuration:
- dynamic targets (.Values.targets.dynamic)
- static targets (.Values.targets.static)
- single target (.Values.target)

The list is exposed as $.targets.
*/}}
{{- define "load-generator.evaluateTargets" -}}
{{- if gt (.Values.targets.dynamic.count | int) 0 -}}
    {{- $list := list -}}

    {{- range $i, $e := until (.Values.targets.dynamic.count | int) -}}
        {{- $_ := set $ "suffix" (tpl $.Values.targets.dynamic.suffixTemplate (dict "index" $i "Template" $.Template)) -}}
        {{- $list = append $list (tpl $.Values.targets.dynamic.template $) -}}
    {{- end -}}

    {{- $_ := set $ "targets" $list -}}
{{- else if .Values.targets.static -}}
    {{- $_ := set $ "targets" .Values.targets.static -}}
{{- else -}}
    {{- $_ := set $ "targets" (list .Values.target) -}}
{{- end -}}
{{- end -}}

{{/*
Target URL for wait-for-service initial container.
*/}}
{{- define "load-generator.waitForServiceTargets" -}}
{{- range $.targets  -}}
{{ " " }}{{ include "load-generator.replacePathInUrl" (dict "url" . "path" $.Values.waitForService.healthCheckPath) -}}
{{- end -}}
{{- end -}}

{{/*
Replace path in given URL.
*/}}
{{- define "load-generator.replacePathInUrl" -}}
{{- $url := urlParse .url -}}
{{- $_ := set $url "path" .path -}}
{{- urlJoin $url -}}
{{- end -}}

{{/*
Expand to the GOMAXPROCS config key if it's absent from values config and the
load-generator has a corresponding CPU limit.
*/}}
{{- define "load-generator.gomaxprocs" -}}
{{- if .Values.resources.limits -}}
{{- if regexFind "\\d+(\\d{3}m)?$" .Values.resources.limits.cpu -}}
{{ regexReplaceAll "\\d{3}m$" .Values.resources.limits.cpu "" }}
{{- else -}}
""
{{- end -}}
{{- else -}}
""
{{- end -}}
{{- end }}
