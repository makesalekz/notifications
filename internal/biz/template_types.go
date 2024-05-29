package biz

import (
	"fmt"
)

type Lang string

const (
	LangRU Lang = "ru"
	LangEN Lang = "en"
	LangKK Lang = "kk"
)

type TemplateType int

const (
	Invite TemplateType = iota
	ConfirmEmail
	NewUser
)

var typeMap = map[string]TemplateType{
	"invite":        Invite,
	"confirm_email": ConfirmEmail,
	"new_user":      NewUser,
}

func (t TemplateType) GetTemplatePath(lang Lang) string {
	basePath := "templates/%s/%s.html"
	switch t {
	case Invite:
		return fmt.Sprintf(basePath, lang, "invite_email_template")
	case ConfirmEmail:
		return fmt.Sprintf(basePath, lang, "confirm_email_template")
	case NewUser:
		return fmt.Sprintf(basePath, lang, "new_user_email_template")
	default:
		return ""
	}
}

func getTemplateTypeFromString(typeStr string) (TemplateType, error) {
	tType, ok := typeMap[typeStr]
	if !ok {
		return 0, fmt.Errorf("invalid email type: %s", typeStr)
	}
	return tType, nil
}
