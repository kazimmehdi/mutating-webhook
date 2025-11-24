{{/*
Expand the name of the chart.
*/}}
{{- define "mutating-webhook.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "mutating-webhook.fullname" -}}
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
{{- define "mutating-webhook.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "mutating-webhook.labels" -}}
helm.sh/chart: {{ include "mutating-webhook.chart" . }}
{{ include "mutating-webhook.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "mutating-webhook.selectorLabels" -}}
app.kubernetes.io/name: {{ include "mutating-webhook.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "mutating-webhook.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "mutating-webhook.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Generate or load the CA cert.
Returned object: dict with key "Cert"
*/}}
{{- define "mutating-webhook.tls" -}}
  {{- if .Values.selfSignedCert.enabled }}

    {{- $ca := include "mutating-webhook.ca" . | fromYaml }}
    {{- $cn := tpl .Values.selfSignedCert.commonName . }}

    {{- /* Normalize altNames into a list */}}
    {{- $altNames := list }}
    {{- if kindIs "slice" .Values.selfSignedCert.altNames }}
      {{- $altNames = .Values.selfSignedCert.altNames }}
    {{- else if .Values.selfSignedCert.altNames }}
      {{- $altNames = list .Values.selfSignedCert.altNames }}
    {{- end }}

    {{- /* Call genSignedCert safely */}}
    {{- $cert := genSignedCert $cn $altNames (list) $ca 365 }}

    {{- $cert }}

  {{- else }}

    {{- dict
          "Cert" .Values.tls.crt
          "Key"  .Values.tls.key
    }}

  {{- end }}
{{- end }}


{{/*
Generate certificates for mutating-webhook api server 
*/}}
{{- define "mutating-webhook.gen-certs" -}}
{{- $altNames := list ( printf "%s.%s" (include "mutating-webhook.name" .) .Release.Namespace ) ( printf "%s.%s.svc" (include "mutating-webhook.name" .) .Release.Namespace ) -}}
{{- $ca := genCA "mutating-webhook-ca" 365 -}}
{{- $cert := genSignedCert ( include "mutating-webhook.name" . ) nil $altNames 365 $ca -}}
tls.crt: {{ $cert.Cert | b64enc }}
tls.key: {{ $cert.Key | b64enc }}
ca: {{ $ca.Cert | b64enc }}
{{- end -}}