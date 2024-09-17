{{- define "deploymentBasics" }}
strategy:
  rollingUpdate:
    maxSurge: {{ (and .Values.deployment .Values.deployment.maxSurge) | default 5 }}
    maxUnavailable: {{ (and .Values.deployment .Values.deployment.maxUnavailable) | default 0 }}
  type: {{ .Values.deployment.strategy }}
revisionHistoryLimit: 3
selector:
  matchLabels:
    app: {{ .Release.Name }}
{{- end }}

{{- define "deploymentInitContainers" }}
{{- if and (eq .Values.db.inMemory false) (eq .Values.db.postgresNeedsProvision true) }}
initContainers:
- name: check-postgres-connection
  image: busybox
  command:
    - sh
    - -c
    - |
      echo "Checking PostgreSQL connection..."
      until nc -z -v -w30 {{ include "postgresql.host" . }} 5432
      do
        echo "Waiting for PostgreSQL connection..." && sleep 5
      done
      echo "PostgreSQL is up and running."
{{ end }}
{{- end }}