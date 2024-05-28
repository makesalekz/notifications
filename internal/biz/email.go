package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ses/types"
	"github.com/go-kratos/kratos/v2/log"
	nnats "github.com/nats-io/nats.go"
	"gitlab.calendaria.team/services/notifications/messages"
	"gitlab.calendaria.team/services/utils/v1/config"
	"gitlab.calendaria.team/services/utils/v1/nats"
	"os"
)

type EmailUsecase struct {
	client    *ses.Client
	log       *log.Helper
	config    *config.Config
	templates *LocalizedEmailTemplates
	qm        *nats.QueueManager
}

func NewEmailUsecase(config *config.Config, logger log.Logger, qm *nats.QueueManager, templates *LocalizedEmailTemplates) (*EmailUsecase, error) {
	uc := &EmailUsecase{
		config:    config,
		log:       log.NewHelper(logger),
		qm:        qm,
		templates: templates,
	}
	if os.Getenv("DEBUG") == "" {
		if err := uc.setupAWSClient(); err != nil {
			return nil, err
		}
	}
	qm.AddConsumer(QueueEmail, uc.handleEmailRequest)
	return uc, nil
}

func (uc *EmailUsecase) setupAWSClient() error {
	awsCfg, err := loadAWSConfig(uc.config)
	if err != nil {
		return err
	}
	client := ses.NewFromConfig(awsCfg)
	uc.client = client
	return nil
}

func loadAWSConfig(c *config.Config) (aws.Config, error) {
	accessKeyID, err := c.Value("AWS_ACCESS_KEY_ID").String()
	if err != nil {
		return aws.Config{}, fmt.Errorf("failed to load AWS_ACCESS_KEY_ID: %v", err)
	}

	secretAccessKey, err := c.Value("AWS_SECRET_ACCESS_KEY").String()
	if err != nil {
		return aws.Config{}, fmt.Errorf("failed to load AWS_SECRET_ACCESS_KEY: %v", err)
	}

	region, err := c.Value("AWS_REGION").String()
	if err != nil {
		return aws.Config{}, fmt.Errorf("failed to load AWS_REGION: %v", err)
	}

	awsCfg, err := awsConfig.LoadDefaultConfig(context.TODO(),
		awsConfig.WithRegion(region),
		awsConfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, "")),
	)
	if err != nil {
		return aws.Config{}, fmt.Errorf("failed to load AWS configs: %v", err)
	}

	return awsCfg, nil
}

func (uc *EmailUsecase) handleEmailRequest(ctx context.Context, m *nnats.Msg) bool {
	var request messages.EmailDetails
	if err := json.Unmarshal(m.Data, &request); err != nil {
		uc.log.Errorf("handleEmailRequest: json.Unmarshal: %uc", err)
		return true
	}
	if err := uc.SendEmail(ctx, &request); err != nil {
		uc.log.Errorf("handleEmailRequest: SendEmail: %uc", err)
		return false
	}
	return true
}

func (uc *EmailUsecase) SendEmail(ctx context.Context, emailDetails *messages.EmailDetails) error {
	sourceEmail, subject, err := uc.loadEmailConfig()
	if err != nil {
		uc.log.Errorf("loading email configuration failed: %v", err)
		return err
	}

	templateType, err := getTypeFromString(emailDetails.Type)
	if err != nil {
		uc.log.Errorf("resolving template type failed: %v", err)
		return err
	}

	body, err := uc.templates.ExecuteTemplate(Lang(emailDetails.Language), templateType, emailDetails.Data)
	if err != nil {
		uc.log.Errorf("executing email template failed: %v", err)
		return err
	}

	return uc.sendSESEmail(ctx, emailDetails.Email, sourceEmail, subject, body)
}

func (uc *EmailUsecase) loadEmailConfig() (sourceEmail, subject string, err error) {
	sourceEmail, err = uc.config.Value("SES_SOURCE_EMAIL").String()
	if err != nil {
		return
	}
	subject, err = uc.config.Value("SES_EMAIL_SUBJECT").String()
	return sourceEmail, subject, err
}

func (uc *EmailUsecase) sendSESEmail(ctx context.Context, recipient, sourceEmail, subject, body string) error {
	if uc.client == nil {
		return fmt.Errorf("SES client is not initialized")
	}

	input := &ses.SendEmailInput{
		Source: aws.String(sourceEmail),
		Destination: &types.Destination{
			ToAddresses: []string{recipient},
		},
		Message: &types.Message{
			Subject: &types.Content{Data: aws.String(subject)},
			Body:    &types.Body{Html: &types.Content{Data: aws.String(body)}},
		},
	}

	_, err := uc.client.SendEmail(ctx, input)
	if err != nil {
		uc.log.Errorf("failed to send email: %v", err)
		return err
	}

	uc.log.Infof("Email sent successfully to %s", recipient)
	return nil
}
