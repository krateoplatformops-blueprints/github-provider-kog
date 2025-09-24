{{/*
Return true if the collaborator plugin should be enabled.
It checks for the existence of the 'restdefinitions' block passed down from the parent chart.
*/}}
{{- define "plugins-chart.collaborator.enabled" -}}
{{- .Values.restdefinitions.collaborator.enabled | default false -}}
{{- end -}}

{{/*
Return true if the teamrepo plugin should be enabled.
*/}}
{{- define "plugins-chart.teamrepo.enabled" -}}
{{- .Values.restdefinitions.teamrepo.enabled | default false -}}
{{- end -}}
