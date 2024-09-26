{{- define "sql.instanceName" -}}
{{ .Values.databaseInstanceName }}-{{ .Values.databaseRegion }}
{{- end }}
{{- define "sql.sqlUser" -}}
{{ .Values.databaseInstanceName }}-{{ .Release.Name }}
{{- end }}
{{- define "sql.sqlDatabase" -}}
{{ .Values.databaseInstanceName }}-{{ .Release.Name }}
{{- end }}
{{- define "sql.sqlSSLCert" -}}
{{ .Values.databaseInstanceName }}-{{ .Release.Name }}
{{- end }}
{{- define "sql.computeAddress" -}}
{{ .Values.databaseInstanceName }}-{{ .Release.Name }}
{{- end }}
{{- define "sql.serviceNetworkConnection" -}}
{{ .Values.databaseInstanceName }}-{{ .Release.Name }}
{{- end }}
{{- define "deployment.iamSubscriberName" -}}
{{ .Release.Name }}-subscriber
{{- end }}

{{/*postgres*/}}
{{- define "postgresql.host" -}}
  {{ .Release.Name }}-postgresql
{{- end }}
{{- define "postgresql.secretName" -}}
    {{- if .Values.global.postgresql.auth.existingSecret -}}
      {{ .Values.global.postgresql.auth.existingSecret }}
    {{- else -}}
      {{ .Release.Name }}-postgresql
    {{- end -}}
{{- end }}



{{- define "env.database" }}
- name: DATABASE_USER
  value: {{ ((((.Values).global).postgresql).auth).username }}
- name: DATABASE_PASSWORD
  valueFrom:
    secretKeyRef:
      name: {{ include "postgresql.secretName" . }}
      key: {{ (((((.Values).global).postgresql).auth).secretKeys).adminPasswordKey }}
- name: DATABASE_NAME
  value: {{ ((((.Values).global).postgresql).auth).database }}
- name: DATABASE_INSTANCE_NAMESPACE
  value: {{ .Values.databaseInstanceNamespace | default "dxp-system" }}
- name: DATABASE_TMP_DIR
  value: "/app"
- name: DATABASE_INSTANCE
  value: {{ include "sql.instanceName" . }}
- name: DATABASE_SSL_CERT_NAME
  value: {{ include "sql.sqlSSLCert" . }}
- name: DATABASE_SSL_CERT_NAMESPACE
  value: {{ .Values.databaseSSLCertNamespace | default "dxp-system" }}
{{- end }}

{{- define "env.gcp" }}
- name: EVENTING_GCP_PROJECT_ID
  value: {{ .Values.eventing.projectId }}
- name: EVENTING_GCP_CREDENTIALS_FILE
  value: /var/opt/gcp/key.json
- name: EVENTING_TOPIC
  value: {{ .Values.eventing.topicId }}
- name: EVENTING_ENABLED
  value: "{{ .Values.eventing.enabled }}"
{{- end }}

{{- define "env.openfga" }}
- name: OPENFGA_ENABLED
  value: "{{ index .Values.features "open-fga-enabled" }}"
{{- if eq (index .Values.features "open-fga-enabled") true }}
- name: OPENFGA_EVENTING_ENABLED
  value: "{{ ((.Values.openfga).eventing).enabled | default false }}"
- name: OPENFGA_GRPC_ADDR
  value: "{{ .Values.openfga.grpcAddr }}"
{{- end }}
- name: OPENFGA_DISABLE_LEGACY_QUERIES
  value: "{{ .Values.openfga.disableLegacyQueries }}"
{{- end }}

{{- define "port.fga" }}
{{- if eq (index .Values.features "open-fga-enabled") true }}
- containerPort: {{ .Values.openfga.port }}
  name: grpc
  protocol: TCP
{{- end }}
{{- end }}

{{- define "port.fga.service" }}
{{- if eq (index .Values.features "open-fga-enabled") true }}
- port: {{ .Values.openfga.port }}
  name: grpc
  protocol: TCP
{{- end }}
{{- end }}


