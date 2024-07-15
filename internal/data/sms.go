package data

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"gitlab.calendaria.team/services/utils/v1/config"

	"github.com/go-kratos/kratos/v2/log"
)

type Sms struct {
	Phones  []string
	Message string
}

// region ------------------- smsc request -------------------
// requestSms defines the structure of the request to send SMS.
type requestSms struct {
	Login    string `json:"login"`
	Password string `json:"psw"`
	Phones   string `json:"phones"`
	Message  string `json:"mes"`
	Cost     cost   `json:"cost"`
	Format   format `json:"fmt"`
}

// format defines API output format.
type format int

const (
	formatInlineVerbose format = iota
	formatInline
	formatXML
	formatJSON

	lowBalance = 100
)

// cost defines whether API should send a Result with cost information.
type cost int

const (
	CostWithoutSend cost = iota + 1
	CostCount
	CostCountBalance
)

// endregion ------------------- smsc request -------------------

// region ------------------- smsc response -------------------
// MetaResult allows to parse Result and Error from response while structures have different fields.
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

// endregion ------------------- smsc response -------------------

type SmscClient interface {
	SendSms(ctx context.Context, sms Sms) (*Result, error)
}

type smscClient struct {
	smsEndpoint    string
	smsCredentials map[string]interface{}
	log            *log.Helper
	debug          bool
}

func NewSmscClient(config *config.Config, logger log.Logger) SmscClient {
	debug := os.Getenv("DEBUG") != ""
	s := &smscClient{
		debug: debug,
		log:   log.NewHelper(log.With(logger, "module", "data/sms")),
	}

	s.log.Infof("debug: %v", debug)

	if !debug {
		// Get the SMSC endpoint and credentials
		endpoint, err := config.Value("SMSC_ENDPOINT").String()
		if err != nil {
			s.log.Errorf("SMSC endpoint is not set: %v", err)
			return s
		}
		s.smsEndpoint = endpoint

		smscCredentials, err := config.ReadSecretsFor(context.Background(), "smsc")
		if err != nil {
			s.log.Errorf("SMSC credentials are not set: %v", err)
			return s
		}
		s.smsCredentials = smscCredentials
	}

	return s
}

func (s *smscClient) SendSms(ctx context.Context, sms Sms) (*Result, error) {
	if s.debug {
		s.log.Debugf("Sending sms to %s: %s", sms.Phones, sms.Message)
		s.log.Debug("[DEBUG] SMS sent with result: OK - 1 SMS, ID - TEST")
		return &Result{
			ID:    1,
			Count: 1,
		}, nil
	}

	s.log.Infof("Sending sms to %s: <message>", sms.Phones)

	// Check if the SMSC endpoint, login and password are set
	if s.smsEndpoint == "" {
		return nil, errors.New("SMSC endpoint is not set")
	}
	if len(s.smsCredentials) == 0 {
		return nil, errors.New("SMSC credentials are not set")
	}
	login, ok := s.smsCredentials["login"].(string)
	if !ok {
		return nil, errors.New("SMSC Login is not set")
	}
	password, ok := s.smsCredentials["password"].(string)
	if !ok {
		return nil, errors.New("SMSC Password is not set")
	}

	// Check the required fields
	if sms.Message == "" {
		return nil, errors.New("message is not set")
	} else if len(sms.Phones) == 0 {
		return nil, errors.New("phone(s) is (are) not set")
	}

	// Create the request
	request := &requestSms{
		Login:    login,
		Password: password,
		Phones:   strings.Join(sms.Phones, ","),
		Message:  sms.Message,
		Cost:     CostCountBalance,
		Format:   formatJSON,
	}

	// Marshal the request to JSON
	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.smsEndpoint, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("new request error: %w", err)
	}

	// Post request to send auth code with sms
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("post request error: %w", err)
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
		return nil, fmt.Errorf("decode error: %w", err)
	}

	// Check if the response contains an error
	if response.Error != nil {
		return nil, response.Error
	}

	// Check the balance to log/notify if it is low
	if response.Balance != nil {
		balance, err2 := strconv.ParseFloat(*response.Balance, 64)
		if err2 != nil {
			log.Errorf("smsc.kz balance parse error: %v", err2)
		} else if balance < lowBalance {
			log.Warnf("smsc.kz balance is low: %v", balance)
		}
	}

	s.log.Infof("SMS sent with result: %s", response.Result)

	return response.Result, nil
}
