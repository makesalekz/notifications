package nats

import "time"

const (
	maxAgeDuration     = 3 * 24 * time.Hour
	maxMessagesInQueue = 1000000
	maxMessageSize     = 5 << 20 // one message maximum size 5mb
)

type Option func(*Options)

type Options struct {
	MaxAge             time.Duration
	MaxMessagesInQueue int64
	MaxMessageSize     int32
	Delay              *time.Duration
	BackOff            []time.Duration
}

func NewOptions() *Options {
	return &Options{
		MaxAge:             maxAgeDuration,
		MaxMessagesInQueue: maxMessagesInQueue,
		MaxMessageSize:     maxMessageSize,
		BackOff:            []time.Duration{5 * time.Second, 15 * time.Second, 5 * time.Minute},
	}
}

func WithMaxAge(maxAge time.Duration) Option {
	return func(o *Options) {
		o.MaxAge = maxAge
	}
}

func WithMaxMessagesInQueue(maxMessagesInQueue int64) Option {
	return func(o *Options) {
		o.MaxMessagesInQueue = maxMessagesInQueue
	}
}

func WithMaxMessageSize(maxMessageSize int32) Option {
	return func(o *Options) {
		o.MaxMessageSize = maxMessageSize
	}
}

func WithDelay(delay time.Duration) Option {
	return func(o *Options) {
		o.Delay = &delay
	}
}

func WithBackOff(backOff []time.Duration) Option {
	return func(o *Options) {
		o.BackOff = backOff
	}
}
