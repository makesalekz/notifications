package data

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"gitlab.calendaria.team/services/utils/v1/config"

	"github.com/go-kratos/kratos/v2/log"
	_ "github.com/lib/pq"
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
	SendSms(sms Sms) (*Result, error)
}

type smscClient struct {
	SmsEndpoint    string
	SmsCredentials map[string]interface{}
}

func NewSmscClient(config *config.Config) SmscClient {
	repo := &smscClient{}

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

func (s *smscClient) SendSms(sms Sms) (*Result, error) {
	// Check if the SMSC endpoint, login and password are set
	if s.SmsEndpoint == "" {
		return nil, fmt.Errorf("SMSC endpoint is not set")
	}
	login, ok := s.SmsCredentials["login"].(string)
	if !ok {
		return nil, fmt.Errorf("SMSC Login is not set")
	}
	password, ok := s.SmsCredentials["password"].(string)
	if !ok {
		return nil, fmt.Errorf("SMSC Password is not set")
	}

	// Check the required fields
	if sms.Message == "" {
		return nil, fmt.Errorf("message is not set")
	} else if len(sms.Phones) == 0 {
		return nil, fmt.Errorf("phone(s) is (are) not set")
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
		return nil, fmt.Errorf("marshal error: %v", err)
	}

	// Post request to send auth code with sms
	res, err := http.Post(s.SmsEndpoint, "application/json", bytes.NewBuffer(body))
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

	// Check the balance to log/notify if it is low
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
