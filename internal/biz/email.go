package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ses/types"
	"github.com/go-kratos/kratos/v2/log"
	nnats "github.com/nats-io/nats.go"
	"gitlab.calendaria.team/services/utils/v1/config"
	"gitlab.calendaria.team/services/utils/v1/nats"
)

type InviteEmail struct {
	AppId    string
	Email    string
	UserId   int64
	InviteId string
}

// EmailUsecase is a Greeter usecase.
type EmailUsecase struct {
	client *ses.Client
	log    *log.Helper
	config *config.Config
	qm     *nats.QueueManager
}

func NewEmailUsecase(config *config.Config, logger log.Logger, qm *nats.QueueManager) (*EmailUsecase, error) {
	uc := &EmailUsecase{
		config: config,
		log:    log.NewHelper(logger),
		qm:     qm,
	}

	if os.Getenv("DEBUG") == "" {
		awsCfg, err := loadAWSConfig(config)
		if err != nil {
			return nil, err
		}
		client := ses.NewFromConfig(awsCfg)
		uc.client = client
	}

	qm.AddConsumer(QueueEmail, uc.sendNotifications)

	return uc, nil
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

func (uc *EmailUsecase) sendNotifications(ctx context.Context, m *nnats.Msg) bool {
	inviteEmail := InviteEmail{}
	err := json.Unmarshal(m.Data, &inviteEmail)
	if err != nil {
		uc.log.Errorf("sendNotifications: json.Unmarshal: %s", err.Error())
		return true
	}

	uc.log.Debugf("sendNotifications: %v", inviteEmail)

	err = uc.SendInviteEmail(ctx, &inviteEmail)
	if err != nil {
		uc.log.Errorf("sendNotifications: SendInviteEmail: %s", err.Error())
		return false
	}

	return true
}

func (uc *EmailUsecase) SendInviteEmail(ctx context.Context, b *InviteEmail) error {
	sourceEmail, err := uc.config.Value("SES_SOURCE_EMAIL").String()
	if err != nil {
		uc.log.Errorf("failed to load source email from configs: %v", err)
		return err
	}

	subject, err := uc.config.Value("SES_EMAIL_SUBJECT").String()
	if err != nil {
		uc.log.Errorf("failed to load email subject from configs: %v", err)
		return err
	}

	baseURL, err := uc.config.Value("BASE_URL").String()
	if err != nil {
		uc.log.Errorf("failed to load base URL from configs: %v", err)
		return err
	}

	templatePath := filepath.Join("templates", "invite_email_template.html")
	body, err := loadEmailTemplate(templatePath, b, baseURL)
	if err != nil {
		uc.log.Errorf("failed to load email template: %v", err)
		return err
	}

	input := &ses.SendEmailInput{
		Source: aws.String(sourceEmail),
		Destination: &types.Destination{
			ToAddresses: []string{b.Email},
		},
		Message: &types.Message{
			Subject: &types.Content{
				Data: aws.String(subject),
			},
			Body: &types.Body{
				Html: &types.Content{
					Data: aws.String(body),
				},
			},
		},
	}

	_, err = uc.client.SendEmail(ctx, input)
	if err != nil {
		uc.log.Errorf("failed to send email: %v", err)
		return err
	}

	uc.log.Infof("email sent successfully to %s", b.Email)
	return nil
}

func loadEmailTemplate(templatePath string, data *InviteEmail, baseURL string) (string, error) {
	templateBytes, err := os.ReadFile(templatePath)
	if err != nil {
		return "", fmt.Errorf("failed to read email template file: %v", err)
	}

	template := string(templateBytes)
	template = strings.ReplaceAll(template, "{{.Email}}", data.Email)
	template = strings.ReplaceAll(template, "{{.UserId}}", fmt.Sprintf("%d", data.UserId))
	template = strings.ReplaceAll(template, "{{.InviteId}}", data.InviteId)
	template = strings.ReplaceAll(template, "{{.InviteLink}}", fmt.Sprintf("%s/a/%s/%d", baseURL, data.InviteId, data.UserId))

	return template, nil
}
