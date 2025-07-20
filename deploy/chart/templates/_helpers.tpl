{{/*
Expand the name of the chart.
*/}}
{{- define "xds-cp-operator.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "xds-cp-operator.fullname" -}}
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
{{- define "xds-cp-operator.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "xds-cp-operator.labels" -}}
helm.sh/chart: {{ include "xds-cp-operator.chart" . }}
{{ include "xds-cp-operator.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- with .Values.commonLabels }}
{{ toYaml . }}
{{- end }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "xds-cp-operator.selectorLabels" -}}
app.kubernetes.io/name: {{ include "xds-cp-operator.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "xds-cp-operator.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "xds-cp-operator.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Create image name with tag
*/}}
{{- define "xds-cp-operator.image" -}}
{{- printf "%s:%s" .Values.image.repository (.Values.image.tag | default .Chart.AppVersion) }}
{{- end }}

{{/*
Common annotations
*/}}
{{- define "xds-cp-operator.annotations" -}}
{{- with .Values.commonAnnotations }}
{{ toYaml . }}
{{- end }}
{{- end }}

{{/*
Create ClusterRole name
*/}}
{{- define "xds-cp-operator.clusterRoleName" -}}
{{- printf "%s-manager-role" (include "xds-cp-operator.fullname" .) }}
{{- end }}

{{/*
Create ClusterRoleBinding name
*/}}
{{- define "xds-cp-operator.clusterRoleBindingName" -}}
{{- printf "%s-manager-rolebinding" (include "xds-cp-operator.fullname" .) }}
{{- end }}

{{/*
Create leader election Role name
*/}}
{{- define "xds-cp-operator.leaderElectionRoleName" -}}
{{- printf "%s-leader-election-role" (include "xds-cp-operator.fullname" .) }}
{{- end }}

{{/*
Create leader election RoleBinding name
*/}}
{{- define "xds-cp-operator.leaderElectionRoleBindingName" -}}
{{- printf "%s-leader-election-rolebinding" (include "xds-cp-operator.fullname" .) }}
{{- end }} 