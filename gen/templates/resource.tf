resource "nso_{{snakeCase .Name}}" "example" {
{{- range  .Attributes}}
{{- if and (ne .ExcludeTest true) (ne .ExcludeExample true)}}
{{- if and (eq .Type "List") (ne .ListElement "String")}}
  {{.TfName}} = [
    {
      {{- range  .Attributes}}
      {{- if and (ne .ExcludeTest true) (ne .ExcludeExample true)}}
      {{.TfName}} = {{if eq .Type "String"}}"{{end}}{{.Example}}{{if eq .Type "String"}}"{{end}}
      {{- end}}
      {{- end}}
    }
  ]
{{- else}}
  {{.TfName}} = {{if eq .Type "String"}}"{{end}}{{.Example}}{{if eq .Type "String"}}"{{end}}
{{- end}}
{{- end}}
{{- end}}
}
