package data

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

type Localizer struct {
	bundle *i18n.Bundle
}

func NewLocalizerForTest() (*Localizer, error) {
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", "locales"))

	bundle := i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)

	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		bundle.MustLoadMessageFile(filepath.Join(dir, file.Name()))
	}

	return &Localizer{
		bundle: bundle,
	}, nil
}

func NewLocalizer() (*Localizer, error) {
	bundle := i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)
	files, err := os.ReadDir("locales/")

	if err != nil {
		return nil, err
	}

	for _, file := range files {
		bundle.MustLoadMessageFile("locales/" + file.Name())
	}

	return &Localizer{
		bundle: bundle,
	}, nil
}

func (loc *Localizer) GetLocalizedMessage(
	langTag string, id string, templateData map[string]interface{}, pluralCount *int64,
) (string, error) {
	localizer := i18n.NewLocalizer(loc.bundle, langTag)

	locConfig := &i18n.LocalizeConfig{
		MessageID:    id,
		TemplateData: templateData,
	}
	if pluralCount != nil {
		locConfig.PluralCount = *pluralCount
	}

	message, err := localizer.Localize(locConfig)
	if err != nil {
		return "", err
	}

	return message, nil
}
