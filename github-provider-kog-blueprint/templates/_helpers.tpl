{{/*
Return true if the collaborator blueprint should be enabled.
It checks for the existence of the 'restdefinitions' block passed down from the parent chart and if the 'collaborator' key is set to true.
*/}}
{{- define "github-provider-kog-collaborator-blueprint.enabled" -}}
{{- .Values.restdefinitions.collaborator.enabled | default false -}}
{{- end -}}

{{/*
Return true if the repo blueprint should be enabled.
It checks for the existence of the 'restdefinitions' block passed down from the parent chart and if the 'repo' key is set to true.
*/}}
{{- define "github-provider-kog-repo-blueprint.enabled" -}}
{{- .Values.restdefinitions.repo.enabled | default false -}}
{{- end -}}

{{/*
Return true if the runnergroup blueprint should be enabled.
It checks for the existence of the 'restdefinitions' block passed down from the parent chart and if the 'runnergroup' key is set to true.
*/}}
{{- define "github-provider-kog-runnergroup-blueprint.enabled" -}}
{{- .Values.restdefinitions.runnergroup.enabled | default false -}}
{{- end -}}

{{/*
Return true if the teamrepo blueprint should be enabled.
It checks for the existence of the 'restdefinitions' block passed down from the parent chart and if the 'teamrepo' key is set to true.
*/}}
{{- define "github-provider-kog-teamrepo-blueprint.enabled" -}}
{{- .Values.restdefinitions.teamrepo.enabled | default false -}}
{{- end -}}

{{/*
Return true if the workflow blueprint should be enabled.
It checks for the existence of the 'restdefinitions' block passed down from the parent chart and if the 'workflow' key is set to true.
*/}}
{{- define "github-provider-kog-workflow-blueprint.enabled" -}}
{{- .Values.restdefinitions.workflow.enabled | default false -}}
{{- end -}}