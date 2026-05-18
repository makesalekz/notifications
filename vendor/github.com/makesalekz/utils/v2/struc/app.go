package struc

import (
	"os"
)

type ApplicationID string

const (
	AppCalendaria ApplicationID = "calendaria"
	AppPMS        ApplicationID = "pms"
	AppTickets    ApplicationID = "tickets"
	AppKnowledge  ApplicationID = "knowledge"
)

func (p ApplicationID) Value() string {
	return string(p)
}

func (p ApplicationID) IsValid() bool {
	switch p {
	case AppCalendaria:
		return true
	case AppPMS:
		return true
	case AppTickets:
		return true
	case AppKnowledge:
		return true
	}
	return false
}

func (p ApplicationID) BrandName() string {
	name := os.Getenv("BRAND_NAME")
	if name != "" {
		return name
	}

	switch p {
	case AppCalendaria:
		return "AIgenda"
	case AppPMS:
		return "BasQaru"
	case AppTickets:
		return "Vibe"
	case AppKnowledge:
		return "IdeasGen"
	}

	return "AXIO"
}

func (p ApplicationID) CompanyFullName() string {
	name := os.Getenv("COMPANY_FULL_NAME")
	if name != "" {
		return name
	}

	if p == AppCalendaria || p == AppPMS || p == AppTickets {
		return "TOO \"AXIO\""
	}

	return "TOO \"AXIO\""
}
