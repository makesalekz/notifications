package biz

import (
	"bytes"
	"fmt"
	"html/template"
)

type LocalizedEmailTemplates struct {
	EmailTemplates map[Lang]map[TemplateType]*template.Template
}

func NewLocalizedEmailTemplates() (*LocalizedEmailTemplates, error) {
	tmpls := make(map[Lang]map[TemplateType]*template.Template)
	langs := []Lang{LangRU, LangEN, LangKK}
	types := []TemplateType{Invite, ConfirmEmail, NewUser}

	for _, lang := range langs {
		tmpls[lang] = make(map[TemplateType]*template.Template)
		for _, tType := range types {
			path := tType.GetTemplatePath(lang)
			if path == "" {
				continue // Skip if no path is returned
			}
			tmpl, err := template.ParseFiles(path)
			if err != nil {
				return nil, err
			}
			tmpls[lang][tType] = tmpl
		}
	}

	return &LocalizedEmailTemplates{EmailTemplates: tmpls}, nil
}

func (t *LocalizedEmailTemplates) ExecuteTemplate(lang Lang, tType TemplateType, templateData map[string]string) (string, error) {
	templatesByLang, ok := t.EmailTemplates[lang]
	if !ok {
		return "", fmt.Errorf("no templates found for language %s", lang)
	}

	tmpl, ok := templatesByLang[tType]
	if !ok {
		return "", fmt.Errorf("template not found for type %v in language %s", tType, lang)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, templateData); err != nil {
		return "", err
	}

	return buf.String(), nil
}
