package data

import (
	"context"
	"fmt"

	"gitlab.calendaria.team/services/notifications/ent"
	"gitlab.calendaria.team/services/utils/v1/config"

	_ "github.com/lib/pq"
)

type SmsRepo interface {
	GetSmsEndpoint() (string, error)
	GetSmsLogin() (string, error)
	GetSmsPassword() (string, error)
}

type smsRepo struct {
	db             *ent.Client
	SmsEndpoint    string
	SmsCredentials map[string]interface{}
}

func NewSmsRepo(config *config.Config, d *Data) SmsRepo {
	repo := &smsRepo{
		db: d.db,
	}

	// Get the SMSC endpoint and credentials
	endpoint, err := config.Value("SMSC_ENDPOINT").String()
	if err != nil {
		return repo
	}
	repo.SmsEndpoint = endpoint

	smscCredentials, err := config.ReadSecretsFor(context.Background(), "smsc")
	if err != nil {
		return repo
	}
	repo.SmsCredentials = smscCredentials

	return repo
}

func (s *smsRepo) GetSmsEndpoint() (string, error) {
	if s.SmsEndpoint == "" {
		return "", fmt.Errorf("SMSC endpoint is not set")
	}

	return s.SmsEndpoint, nil
}

func (s *smsRepo) GetSmsLogin() (string, error) {
	login, ok := s.SmsCredentials["login"].(string)
	if !ok {
		return "", fmt.Errorf("SMSC Login is not set")
	}

	return login, nil
}

func (s *smsRepo) GetSmsPassword() (string, error) {
	password, ok := s.SmsCredentials["password"].(string)
	if !ok {
		return "", fmt.Errorf("SMSC Password is not set")
	}

	return password, nil
}
