{{- define "skills"}}
{{- if .Workspace.SkillsXML }}

## Available Skills

You have access to the following skills in the current workspace ({{.Workspace.CurrentPath}}):

{{.Workspace.SkillsXML}}

Available skills in this workspace:
{{- range .Workspace.Skills }}
- **{{.Name}}**: {{.Description}}
{{- end }}
{{- end }}
{{- end }}
