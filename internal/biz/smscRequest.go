package biz

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/go-kratos/kratos/v2/log"
	"net/http"
	"strconv"
	"strings"
)

type SmsRequest struct {
	Endpoint string
	Login    string
	Password string
	Message  string
	Phones   []string
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

func SendSms(sms SmsRequest) (*Result, error) {
	// Check the required fields
	if sms.Endpoint == "" {
		return nil, fmt.Errorf("endpoint is not set")
	} else if sms.Login == "" || sms.Password == "" {
		return nil, fmt.Errorf("login or password is not set")
	} else if sms.Message == "" {
		return nil, fmt.Errorf("message is not set")
	} else if len(sms.Phones) == 0 {
		return nil, fmt.Errorf("phone(s) is (are) not set")
	}

	// Create the request
	request := &requestSms{
		Login:    sms.Login,
		Password: sms.Password,
		Phones:   strings.Join(sms.Phones, ","),
		Message:  sms.Message,
		Cost:     CostCountBalance,
		Format:   formatJSON,
	}

	return sendRequest(sms.Endpoint, request)
}

func sendRequest(endpoint string, request *requestSms) (*Result, error) {
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
