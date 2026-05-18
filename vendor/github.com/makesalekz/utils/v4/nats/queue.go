package nats

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type IQueue interface {
	// Pub publishes the provided data to the queue.
	// It marshals the data to JSON and sends it to the JetStream server.
	Pub(data any)
	PubDelayed(data any, notBefore time.Time)
}

type Queue struct {
	js   jetstream.JetStream
	log  *log.Helper
	name string
}

// newQueue creates a new Queue instance.
func newQueue(js jetstream.JetStream, log *log.Helper, name string) *Queue {
	return &Queue{
		js:   js,
		log:  log,
		name: name,
	}
}

func (q *Queue) Pub(data any) {
	// marshal data
	msg, err := json.Marshal(data) // JetStream messages need to be []byte
	if err != nil {
		q.log.Errorf("failed to marshal data for %s: %v", q.name, err)
		return
	}

	// initialize context
	ctx := context.WithValue(context.Background(), queueKey{}, q)

	// publish message
	_, err = q.js.Publish(ctx, q.name, msg)
	if err != nil {
		q.log.Errorf("js.Publish for %s: %v", q.name, err)
	}
}

func (q *Queue) PubDelayed(data any, notBefore time.Time) {
	// marshal data
	msgData, err := json.Marshal(data) // JetStream messages need to be []byte
	if err != nil {
		q.log.Errorf("failed to marshal data for %s: %v", q.name, err)
		return
	}

	// initialize context
	ctx := context.WithValue(context.Background(), queueKey{}, q)

	msg := &nats.Msg{
		Subject: q.name,
		Data:    msgData,
		Header: nats.Header{
			"not-before": []string{notBefore.Format(time.RFC3339)},
		},
	}

	// publish message
	_, err = q.js.PublishMsg(ctx, msg)
	if err != nil {
		q.log.Errorf("js.Publish for %s: %v", q.name, err)
	}
}
