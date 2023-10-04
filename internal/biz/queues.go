package biz

import (
	"context"
	"notifications/internal/data"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/nats-io/nats.go"
)

type QueueManager struct {
	nc      *data.NatsClient
	log     *log.Helper
	appName string
}

func NewQueueManager(c *data.Config, nc *data.NatsClient, logger log.Logger) *QueueManager {
	return &QueueManager{
		nc:      nc,
		log:     log.NewHelper(logger),
		appName: c.GetAppName(),
	}
}

type queueKey struct{}

func (qm *QueueManager) Create(name string, handler func(ctx context.Context, m *nats.Msg) bool) *Queue {
	subj := qm.appName + "/" + name
	queue := newQueue(qm.nc, subj)

	if handler != nil {
		ctx := context.WithValue(context.Background(), queueKey{}, queue)
		_, err := qm.nc.QueueSubscribe(subj, "workers", func(m *nats.Msg) {
			if !handler(ctx, m) {
				m.Nak()
			}
		})
		if err != nil {
			qm.log.Errorf("nc.QueueSubscribe: %s", err.Error())
		}
	}

	return queue
}

func (qm *QueueManager) CreateRemote(app, name string) *Queue {
	return newQueue(qm.nc, app+"/"+name)
}

type Queue struct {
	nc   *data.NatsClient
	name string
}

func newQueue(nc *data.NatsClient, name string) *Queue {
	return &Queue{
		nc:   nc,
		name: name,
	}
}

func (q *Queue) Pub(data any) error {
	return q.nc.Publish(q.name, data)
}

type Notification struct {
	UsersIds []int64
	Title    string
	Body     string
	Image    string
	Data     map[string]string
}
