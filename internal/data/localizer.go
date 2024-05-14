package data

import (
	"encoding/json"
	"os"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

type Localizer struct {
	bundle *i18n.Bundle
}

func NewLocalizer() (*Localizer, error) {
	bundle := i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)
	files, err := os.ReadDir("../locales/")
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		bundle.MustLoadMessageFile("../locales/" + file.Name())
	}

	return &Localizer{
		bundle: bundle,
	}, nil
}

func (loc *Localizer) GetLocalizedMessage(langTag string, id string, templateData map[string]interface{}, plularCount *int64) (string, error) {
	localizer := i18n.NewLocalizer(loc.bundle, langTag)

	locConfig := &i18n.LocalizeConfig{
		MessageID:    id,
		TemplateData: templateData,
	}
	if plularCount != nil {
		locConfig.PluralCount = *plularCount
	}

	message, err := localizer.Localize(locConfig)
	if err != nil {
		return "", err
	}

	return message, nil
}
