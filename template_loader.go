package main

type TemplateLoader interface {
	LoadTemplate(templateName string) (string, error)
	LoadAllTemplates() (map[string]string, error)
}
