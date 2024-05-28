package biz

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"
)

type Lang string
type TemplateType struct {
	Pattern string
}

const (
	LangRU Lang = "ru"
	LangEN Lang = "en"
	LangKK Lang = "kk"
)

var (
	Invite TemplateType = TemplateType{Pattern: "templates/{lang}/invite_email_template.html"}
	// Можно добавить другие типы шаблонов здесь
)

type LocalizedEmailTemplates struct {
	EmailTemplates map[Lang]map[TemplateType]*template.Template
}

func NewLocalizedEmailTemplates() (*LocalizedEmailTemplates, error) {
	tmpls := make(map[Lang]map[TemplateType]*template.Template)
	langs := []Lang{LangRU, LangEN, LangKK}
	types := []TemplateType{Invite} // Добавьте другие типы шаблонов здесь

	for _, lang := range langs {
		tmpls[lang] = make(map[TemplateType]*template.Template)
		for _, tType := range types {
			path := strings.Replace(tType.Pattern, "{lang}", string(lang), 1)
			tmpl, err := template.ParseFiles(path)
			if err != nil {
				return nil, err
			}
			tmpls[lang][tType] = tmpl
		}
	}

	return &LocalizedEmailTemplates{EmailTemplates: tmpls}, nil
}

func (t *LocalizedEmailTemplates) ExecuteTemplate(lang Lang, tType TemplateType, templateData map[string]interface{}) (string, error) {
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
