package biz

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-kratos/kratos/v2/log"
	"gitlab.calendaria.team/services/utils/v1/config"
)

type Sms struct {
	Phone   string
	Message string
}

// smsc request structures
// format defines API output format.
type format int

const (
	formatInlineVerbose format = iota
	formatInline
	formatXML
	formatJSON
)

// cost defines whether API should send a Result with cost information.
type cost int

const (
	CostWithoutSend cost = iota + 1
	CostCount
	CostCountBalance
)

type requestSms struct {
	Login    string `json:"login"`
	Password string `json:"psw"`
	Phones   string `json:"phones"`
	Message  string `json:"mes"`
	Cost     cost   `json:"cost"`
	Format   format `json:"fmt"`
}

// smsc response structures

// MetaResult allows to parse Result and Error from response while
// structures have different fields.
type MetaResult struct {
	*Result
	*Error
}

type Result struct {
	ID      int     `json:"id"`
	Count   int     `json:"cnt"`
	Cost    *string `json:"cost"`
	Balance *string `json:"balance"`
	Phones  []Phone `json:"phones"`
}

func (r *Result) String() string {
	return fmt.Sprintf("OK - %d SMS, ID - %d", r.Count, r.ID)
}

type Phone struct {
	Phone  string  `json:"phone"`
	Mccmnc string  `json:"mccmnc"`
	Cost   string  `json:"cost"`
	Status *string `json:"status"`
	Error  *string `json:"error"`
}

type Error struct {
	Code int    `json:"error_code"`
	Desc string `json:"error"`
	ID   *int   `json:"id"`
}

func (e *Error) Error() string {
	s := fmt.Sprintf("ERROR = %d (%s)", e.Code, e.Desc)
	if e.ID != nil {
		s += fmt.Sprintf(", ID - %d", *e.ID)
	}
	return s
}

// SmsUsecase is a Greeter usecase.
type SmsUsecase struct {
	config *config.Config
	log    *log.Helper
}

func NewSmsUsecase(c *config.Config, logger log.Logger) (*SmsUsecase, error) {
	return &SmsUsecase{
		config: c,
		log:    log.NewHelper(logger),
	}, nil
}

func (uc *SmsUsecase) sendSms(ctx context.Context, message string, phones []string) (*Result, error) {
	// Get the SMSC endpoint and credentials
	endpoint, err := uc.config.Value("SMSC_ENDPOINT").String()
	if err != nil {
		return nil, err
	}
	smscCredentials, err := uc.config.ReadSecretsFor(context.Background(), "smsc")
	if err != nil {
		return nil, err
	}

	// Get the SMSC login and password from credentials
	login, ok := smscCredentials["login"].(string)
	if !ok {
		return nil, fmt.Errorf("SMSC Login is not set: %v", smscCredentials)
	}
	password, ok := smscCredentials["password"].(string)
	if !ok {
		return nil, fmt.Errorf("SMSC Password is not set: %v", smscCredentials)
	}

	// Create the request
	request := &requestSms{
		Login:    login,
		Password: password,
		Phones:   strings.Join(phones, ","),
		Message:  message,
		Cost:     CostCountBalance,
		Format:   formatJSON,
	}

	uc.log.WithContext(ctx).Infof("Send sms to %s: %s", phones, message)
	return uc.sendRequest(ctx, endpoint, request)
}

func (uc *SmsUsecase) sendRequest(_ context.Context, endpoint string, request *requestSms) (*Result, error) {
	// Marshal the request to JSON
	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("marshal error: %v", err)
	}

	// Post request to send auth code with sms
	res, err := http.Post(endpoint, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("post request error: %v", err)
	}
	defer res.Body.Close()

	// Check the response status code
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status code error: %v", res.Status)
	}

	// Decode the JSON response
	response := &MetaResult{}
	err = json.NewDecoder(res.Body).Decode(response)
	if err != nil {
		return nil, fmt.Errorf("decode error: %v", err)
	}

	// Check if the response contains an error
	if response.Error != nil {
		return nil, response.Error
	}

	if response.Balance != nil {
		balance, err := strconv.ParseFloat(*response.Balance, 64)
		if err != nil {
			log.Errorf("smsc.kz balance parse error: %v", err)
		} else if balance < 100 {
			log.Warnf("smsc.kz balance is low: %v", balance)
		}
	}

	return response.Result, nil
}

func (uc *SmsUsecase) SendSms(ctx context.Context, sms *Sms) error {
	result, err := uc.sendSms(ctx, sms.Message, []string{sms.Phone})
	if err != nil {
		return err
	}

	uc.log.Debugf("SMS sent with result: %s", result)

	return err
}
