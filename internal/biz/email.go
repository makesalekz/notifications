package biz

import (
	"context"
	"encoding/json"

	v1 "gitlab.calendaria.team/services/notifications/api/notifications/v1"
	"gitlab.calendaria.team/services/notifications/internal/data"
	u_struc "gitlab.calendaria.team/services/utils/v2/struc"
	"gitlab.calendaria.team/services/utils/v4/config"
	u_nats "gitlab.calendaria.team/services/utils/v4/nats"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ses/types"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/nats-io/nats.go/jetstream"
)

type EmailUsecase struct {
	client      *ses.Client
	log         *log.Helper
	config      config.IConfig
	templates   *LocalizedEmailTemplates
	localizer   *data.Localizer
	qm          u_nats.IQueueManager
	sourceEmail string
}

func NewEmailUsecase(
	config config.IConfig,
	logger log.Logger,
	qm u_nats.IQueueManager,
	templates *LocalizedEmailTemplates,
	localizer *data.Localizer,
) (*EmailUsecase, error) {
	uc := &EmailUsecase{
		config:    config,
		log:       log.NewHelper(logger),
		qm:        qm,
		templates: templates,
		localizer: localizer,
	}

	sourceEmail, err := uc.config.GetValue("SES_SOURCE_EMAIL")
	if err == nil {
		err = uc.setupAWSClient()
		if err != nil {
			return nil, err
		}
		uc.sourceEmail = sourceEmail
	} else {
		uc.log.Infof("loading email configuration failed: %v", err)
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

func loadAWSConfig(c config.IConfig) (aws.Config, error) {
	secrets, err := c.ReadSecretsFor(context.Background(), "aws")
	if err != nil {
		return aws.Config{}, v1.ErrorInternal("failed to read AWS secrets: %v", err)
	}
	accessKeyID, ok := secrets["access_key_id"].(string)
	if !ok {
		return aws.Config{}, v1.ErrorInternal("failed to load access_key_id")
	}
	secretAccessKey, ok := secrets["secret_access_key"].(string)
	if !ok {
		return aws.Config{}, v1.ErrorInternal("failed to load secret_access_key")
	}

	region, err := c.GetValue("AWS_REGION")
	if err != nil {
		return aws.Config{}, v1.ErrorInternal("failed to load AWS_REGION: %v", err)
	}

	awsCfg, err := awsConfig.LoadDefaultConfig(
		context.TODO(),
		awsConfig.WithRegion(region),
		awsConfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, "")),
	)
	if err != nil {
		return aws.Config{}, v1.ErrorInternal("failed to load AWS configs: %v", err)
	}

	return awsCfg, nil
}

func (uc *EmailUsecase) handleEmailRequest(ctx context.Context, m jetstream.Msg) bool {
	var request u_struc.EmailDetails
	if err := json.Unmarshal(m.Data(), &request); err != nil {
		uc.log.Errorf("handleEmailRequest: json.Unmarshal: %uc", err)
		return true
	}
	if err := uc.SendEmail(ctx, &request); err != nil {
		uc.log.Errorf("handleEmailRequest: SendEmail: %uc", err)
		return false
	}
	return true
}

func (uc *EmailUsecase) SendEmail(ctx context.Context, emailDetails *u_struc.EmailDetails) error {
	messageID := "email.subject." + emailDetails.Type
	subject, err := uc.localizer.GetLocalizedMessage(emailDetails.Language, messageID, nil, nil)
	if err != nil {
		uc.log.Errorf("localizing email subject failed: %v", err)
		return err
	}

	templateType, err := getTemplateTypeFromString(emailDetails.Type)
	if err != nil {
		uc.log.Errorf("resolving template type failed: %v", err)
		return err
	}

	body, err := uc.templates.ExecuteTemplate(Lang(emailDetails.Language), templateType, emailDetails.Data)
	if err != nil {
		uc.log.Errorf("executing email template failed: %v", err)
		return err
	}

	return uc.sendSESEmail(ctx, emailDetails.Emails, subject, body)
}

func (uc *EmailUsecase) sendSESEmail(ctx context.Context, recipients []string, subject, body string) error {
	if uc.client == nil {
		return v1.ErrorInternal("SES client is not initialized")
	}

	input := &ses.SendEmailInput{
		Source: aws.String(uc.sourceEmail),
		Destination: &types.Destination{
			BccAddresses: recipients,
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

	uc.log.Infof("Email sent successfully to %s", recipients)
	return nil
}
