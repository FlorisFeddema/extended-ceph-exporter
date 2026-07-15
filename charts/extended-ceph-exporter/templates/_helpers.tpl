{{- define "extended-ceph-exporter.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "extended-ceph-exporter.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{- define "extended-ceph-exporter.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "extended-ceph-exporter.labels" -}}
helm.sh/chart: {{ include "extended-ceph-exporter.chart" . }}
app.kubernetes.io/name: {{ include "extended-ceph-exporter.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{- define "extended-ceph-exporter.selectorLabels" -}}
app.kubernetes.io/name: {{ include "extended-ceph-exporter.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{- define "extended-ceph-exporter.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
{{- default (include "extended-ceph-exporter.fullname" .) .Values.serviceAccount.name -}}
{{- else -}}
{{- default "default" .Values.serviceAccount.name -}}
{{- end -}}
{{- end -}}

{{- define "extended-ceph-exporter.credentialsSecretName" -}}
{{- if .Values.rgw.credentials.existingSecret.name -}}
{{- .Values.rgw.credentials.existingSecret.name -}}
{{- else if .Values.rgw.credentials.secretName -}}
{{- .Values.rgw.credentials.secretName -}}
{{- else if .Values.rook.objectStoreUser.secretName -}}
{{- .Values.rook.objectStoreUser.secretName -}}
{{- else if and .Values.rook.objectStoreUser.enabled .Values.rook.objectStoreUser.store (include "extended-ceph-exporter.objectStoreUserName" .) -}}
{{- printf "rook-ceph-object-user-%s-%s" .Values.rook.objectStoreUser.store (include "extended-ceph-exporter.objectStoreUserName" .) -}}
{{- else -}}
{{- printf "%s-rgw-credentials" (include "extended-ceph-exporter.fullname" .) -}}
{{- end -}}
{{- end -}}

{{- define "extended-ceph-exporter.objectStoreUserName" -}}
{{- default (include "extended-ceph-exporter.fullname" .) .Values.rook.objectStoreUser.name -}}
{{- end -}}

{{- define "extended-ceph-exporter.objectStoreUserNamespace" -}}
{{- default .Release.Namespace .Values.rook.objectStoreUser.namespace -}}
{{- end -}}
