package worker

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"
)

type MessageHandler func(context.Context, *nats.Msg) error

type PullWorker struct {
	Name    string
	Subject string
	Durable string
	Stream  string
	Batch   int
	JS      nats.JetStreamContext
	Handle  MessageHandler
}

func (w PullWorker) Run(ctx context.Context) error {
	batch := w.Batch
	if batch <= 0 {
		batch = 10
	}

	sub, err := w.JS.PullSubscribe(w.Subject, w.Durable, nats.BindStream(w.Stream), nats.ManualAck())
	if err != nil {
		return err
	}

	slog.Info("worker started", "name", w.Name, "subject", w.Subject, "durable", w.Durable)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		msgs, err := sub.Fetch(batch, nats.MaxWait(2*time.Second))
		if errors.Is(err, nats.ErrTimeout) {
			continue
		}
		if err != nil {
			slog.Error("fetch messages", "worker", w.Name, "error", err)
			time.Sleep(time.Second)
			continue
		}

		for _, msg := range msgs {
			if err := w.Handle(ctx, msg); err != nil {
				slog.Error("handle message", "worker", w.Name, "error", err)
				_ = msg.Nak()
				continue
			}
			_ = msg.Ack()
		}
	}
}
