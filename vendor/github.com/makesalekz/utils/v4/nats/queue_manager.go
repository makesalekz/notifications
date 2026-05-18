package nats

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/makesalekz/utils/v4/config"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type IQueueManager interface {
	// GetLocal retrieves or initializes a local queue by concat appName (service name) and queue name.
	// If the queue already exists, it's returned; otherwise, a new one is created.
	GetLocal(name string) IQueue

	// GetRemote retrieves or initializes a queue based on the given subject.
	// If the queue already exists, it's returned; otherwise, a new one is created.
	GetRemote(subj string) IQueue

	// AddConsumer adds a new consumer to the specified queue.
	// It creates or updates a JetStream stream and consumer,
	// then starts consuming messages and handling them with the provided handler.
	AddConsumer(queueName string, handler func(ctx context.Context, m jetstream.Msg) bool, opts ...Option)

	// AddRemoteConsumer adds a new consumer to a remote queue.
	// It creates or updates a JetStream stream and consumer,
	// then starts consuming messages and handling them with the provided handler.
	AddRemoteConsumer(appName, queueName string, handler func(ctx context.Context, m jetstream.Msg) bool, opts ...Option)
}

type QueueManager struct {
	js      jetstream.JetStream
	log     *log.Helper
	appName string
	ql      sync.RWMutex
	queues  map[string]*Queue
}

// NewQueueManager initializes a new QueueManager instance,
// establishing a JetStream connection and configuring the manager.
func NewQueueManager(c config.IConfig, nc *nats.Conn, logger log.Logger) IQueueManager {
	js, err := jetstream.New(nc)
	if err != nil {
		log.NewHelper(logger).Errorf("failed to initialize JetStream: %s", err.Error())
		return nil
	}

	return &QueueManager{
		js:      js,
		log:     log.NewHelper(logger),
		appName: c.GetAppName(),
		queues:  make(map[string]*Queue),
	}
}

// getQueue retrieves a queue from the manager's queue map.
func (qm *QueueManager) getQueue(subj string) *Queue {
	qm.ql.RLock()
	defer qm.ql.RUnlock()
	return qm.queues[subj]
}

// initQueue retrieves or initializes a queue.
// If the queue doesn't exist, it's created and added to the map.
func (qm *QueueManager) initQueue(subj string) *Queue {
	qm.ql.Lock()
	defer qm.ql.Unlock()

	queue, ok := qm.queues[subj]
	if !ok {
		queue = newQueue(qm.js, qm.log, subj)
		qm.queues[subj] = queue
	}

	return queue
}

func (qm *QueueManager) GetLocal(queueName string) IQueue {
	names := getNames(qm.appName, qm.appName, queueName)

	return qm.GetRemote(names.Subject)
}

func (qm *QueueManager) GetRemote(subject string) IQueue {
	queue := qm.getQueue(subject)
	if queue == nil {
		queue = qm.initQueue(subject)
	}

	return queue
}

func (qm *QueueManager) AddConsumer(
	queueName string,
	handler func(ctx context.Context, m jetstream.Msg) bool,
	opts ...Option,
) {
	qm.AddRemoteConsumer(qm.appName, queueName, handler, opts...)
}

func (qm *QueueManager) AddRemoteConsumer(
	appName, queueName string,
	handler func(ctx context.Context, m jetstream.Msg) bool,
	opts ...Option,
) {
	// define names
	names := getNames(qm.appName, appName, queueName)

	// parse options
	options := NewOptions()
	for _, opt := range opts {
		opt(options)
	}

	// initial variables
	queue := qm.GetRemote(names.Subject)
	ctx := context.WithValue(context.Background(), queueKey{}, queue)

	// add stream with service name
	stream, err := qm.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:              appName,
		Description:       fmt.Sprintf("Stream for \"%s\" service", appName),
		Subjects:          []string{fmt.Sprintf("%s.*", appName)},
		MaxAge:            options.MaxAge,
		MaxMsgsPerSubject: options.MaxMessagesInQueue,
		MaxMsgSize:        options.MaxMessageSize,
	})
	if err != nil {
		qm.log.Errorf("failed to add stream: %s", err.Error())
		return
	}

	maxDeliver := len(options.BackOff) + 1

	if options.Delay != nil {
		maxDeliver += 1
	}

	// add consumer to stream with subject
	consumer, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Name:          names.ConsumerName,
		Durable:       names.ConsumerName,
		Description:   fmt.Sprintf("Consumer for queue \"%s\"", names.Subject),
		DeliverPolicy: jetstream.DeliverAllPolicy,
		AckPolicy:     jetstream.AckAllPolicy,
		ReplayPolicy:  jetstream.ReplayInstantPolicy,
		AckWait:       30 * time.Second,
		MaxDeliver:    maxDeliver,
		BackOff:       options.BackOff,
		FilterSubject: names.Subject,
	})
	if err != nil {
		qm.log.Errorf("failed to add consumer: %s", err.Error())
		return
	}

	// actions on consume message
	_, err = consumer.Consume(func(msg jetstream.Msg) {
		md, err := msg.Metadata()
		if err != nil {
			qm.log.Errorf("failed to get msg metadata: %s", err.Error())
			return
		}

		qm.log.Infof("delivered %d times", md.NumDelivered)

		// check delay
		if options.Delay != nil {
			if md.NumDelivered == 1 {
				err := msg.NakWithDelay(*options.Delay)
				if err != nil {
					qm.log.Errorf("failed to nak msg: %s", err.Error())
				}
				return
			}
		} else if md.NumDelivered == 1 {
			notBefore := msg.Headers().Get("not-before")
			if notBefore != "" {
				notBeforeTime, err := time.Parse(time.RFC3339, notBefore)
				if err != nil {
					qm.log.Errorf("failed to parse not-before header: %s", err.Error())
					return
				}

				if time.Now().Before(notBeforeTime) {
					err := msg.NakWithDelay(notBeforeTime.Sub(time.Now()))
					if err != nil {
						qm.log.Errorf("failed to nak msg: %s", err.Error())
					}
					return
				}
			}
		}

		var errAck error
		// check handler for success reply
		if handler(ctx, msg) {
			errAck = msg.Ack()
		} else {
			if md.NumDelivered < uint64(len(options.BackOff))+1 {
				errAck = msg.NakWithDelay(options.BackOff[md.NumDelivered-1])
			} else {
				errAck = msg.Nak()
			}
		}
		if errAck != nil {
			qm.log.Errorf("failed to ack msg: %s", errAck.Error())
		}
	})
	if err != nil {
		qm.log.Errorf("failed to consume: %s", err.Error())
		return
	}
}
