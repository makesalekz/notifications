package nats

import (
	"fmt"
)

type queueKey struct{}

type naming struct {
	Subject      string `json:"subject"`
	ConsumerName string `json:"consumer_name"`
}

// getNames function uses to generate standard general name
func getNames(serviceAppName, appName, queueName string) naming {
	name := naming{
		Subject:      fmt.Sprintf("%s.%s", appName, queueName),
		ConsumerName: fmt.Sprintf("%s_%s", serviceAppName, queueName), // consumer name can't contain ., *, >, /, \
	}

	if serviceAppName != appName {
		name.ConsumerName = fmt.Sprintf("%s_%s_%s", serviceAppName, appName, queueName)
	}

	return name
}
