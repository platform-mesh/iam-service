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