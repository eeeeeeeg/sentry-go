package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/nats-io/nats.go"

	"sentry-lite/internal/alert"
	"sentry-lite/internal/config"
	"sentry-lite/internal/grouping"
	"sentry-lite/internal/issue"
	"sentry-lite/internal/normalize"
	"sentry-lite/internal/platform"
	"sentry-lite/internal/worker"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config", "error", err)
		os.Exit(1)
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	deps, err := platform.Connect(ctx, cfg)
	if err != nil {
		slog.Error("connect dependencies", "error", err)
		os.Exit(1)
	}
	defer deps.Close()

	issues := issue.NewRepository(deps.Postgres)
	pull := worker.PullWorker{
		Name:    cfg.GroupingConsumer,
		Subject: cfg.NormalizedEventSubject,
		Durable: cfg.GroupingConsumer,
		Stream:  cfg.RawEventStream,
		Batch:   50,
		JS:      deps.JetStream,
		Handle: func(ctx context.Context, msg *nats.Msg) error {
			var event normalize.NormalizedEvent
			if err := json.Unmarshal(msg.Data, &event); err != nil {
				_ = worker.PublishDLQ(deps.JetStream, cfg.DeadLetterSubject, cfg.GroupingConsumer, msg, err)
				return nil
			}

			fingerprint := grouping.Fingerprint(event)
			upsert, err := issues.UpsertFromEvent(ctx, event, fingerprint)
			if err != nil {
				return err
			}

			event.Fingerprint = fingerprint
			event.IssueID = upsert.IssueID
			body, err := json.Marshal(event)
			if err != nil {
				return err
			}

			out := nats.NewMsg(cfg.GroupedEventSubject)
			out.Data = body
			out.Header.Set("content-type", "application/json")
			out.Header.Set("project-id", event.ProjectID)
			out.Header.Set("event-id", event.EventID)
			out.Header.Set("issue-id", event.IssueID)
			out.Header.Set(nats.MsgIdHdr, event.EventID)
			if _, err = deps.JetStream.PublishMsg(out); err != nil {
				return err
			}

			if err := publishAlert(deps.JetStream, cfg.AlertEventSubject, "issue_seen", event); err != nil {
				return err
			}
			if upsert.AlertType != "" {
				return publishAlert(deps.JetStream, cfg.AlertEventSubject, upsert.AlertType, event)
			}
			return nil
		},
	}

	if err := pull.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		slog.Error("worker stopped", "error", err)
		os.Exit(1)
	}
}

func publishAlert(js nats.JetStreamContext, subject string, alertType string, event normalize.NormalizedEvent) error {
	alertEvent := alert.Event{
		Type:        alertType,
		ProjectID:   event.ProjectID,
		IssueID:     event.IssueID,
		EventID:     event.EventID,
		Level:       event.Level,
		Title:       grouping.Title(event),
		Message:     event.Message,
		Environment: event.Environment,
		Release:     event.Release,
		OccurredAt:  event.Timestamp,
		Event:       event,
	}
	body, err := json.Marshal(alertEvent)
	if err != nil {
		return err
	}
	msg := nats.NewMsg(subject)
	msg.Data = body
	msg.Header.Set("content-type", "application/json")
	msg.Header.Set("project-id", event.ProjectID)
	msg.Header.Set("issue-id", event.IssueID)
	msg.Header.Set("event-id", event.EventID)
	msg.Header.Set("alert-type", alertType)
	msg.Header.Set(nats.MsgIdHdr, alertType+"-"+event.EventID)
	_, err = js.PublishMsg(msg)
	return err
}
