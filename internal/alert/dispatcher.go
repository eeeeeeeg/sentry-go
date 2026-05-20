package alert

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

type Dispatcher struct {
	repo     *Repository
	redis    *redis.Client
	client   *http.Client
	fallback time.Duration
}

func NewDispatcher(repo *Repository, redis *redis.Client, fallback time.Duration) *Dispatcher {
	return &Dispatcher{
		repo:     repo,
		redis:    redis,
		fallback: fallback,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (d *Dispatcher) Dispatch(ctx context.Context, event Event) error {
	if event.Type == "issue_seen" {
		return d.dispatchFrequency(ctx, event)
	}

	rules, err := d.repo.ListActiveRules(ctx, event.ProjectID, event.Type)
	if err != nil {
		return err
	}

	for _, rule := range rules {
		if !levelAtLeast(event.Level, rule.MinLevel) {
			continue
		}
		if rule.Channel != "webhook" || rule.WebhookURL == "" {
			continue
		}

		ok, err := d.reserveSuppression(ctx, rule, event)
		if err != nil {
			return err
		}
		if !ok {
			if err := d.repo.RecordDelivery(ctx, rule.ID, event, "suppressed", rule.Channel, "cooldown window active"); err != nil {
				return err
			}
			continue
		}

		if err := d.sendWebhook(ctx, rule, event); err != nil {
			_ = d.repo.RecordDelivery(ctx, rule.ID, event, "failed", rule.Channel, err.Error())
			return err
		}
		if err := d.repo.RecordDelivery(ctx, rule.ID, event, "sent", rule.Channel, ""); err != nil {
			return err
		}
	}
	return nil
}

func (d *Dispatcher) dispatchFrequency(ctx context.Context, event Event) error {
	rules, err := d.repo.ListActiveRules(ctx, event.ProjectID, "frequency")
	if err != nil {
		return err
	}

	for _, rule := range rules {
		if !levelAtLeast(event.Level, rule.MinLevel) {
			continue
		}
		count, err := d.incrementFrequency(ctx, rule, event)
		if err != nil {
			return err
		}
		if count < int64(rule.ThresholdCount) {
			continue
		}
		frequencyEvent := event
		frequencyEvent.Type = "frequency"
		frequencyEvent.Message = fmt.Sprintf("%s occurred %d times in %d seconds", event.Title, count, rule.WindowSeconds)

		ok, err := d.reserveSuppression(ctx, rule, frequencyEvent)
		if err != nil {
			return err
		}
		if !ok {
			if err := d.repo.RecordDelivery(ctx, rule.ID, frequencyEvent, "suppressed", rule.Channel, "cooldown window active"); err != nil {
				return err
			}
			continue
		}
		if err := d.sendWebhook(ctx, rule, frequencyEvent); err != nil {
			_ = d.repo.RecordDelivery(ctx, rule.ID, frequencyEvent, "failed", rule.Channel, err.Error())
			return err
		}
		if err := d.repo.RecordDelivery(ctx, rule.ID, frequencyEvent, "sent", rule.Channel, ""); err != nil {
			return err
		}
	}
	return nil
}

func (d *Dispatcher) reserveSuppression(ctx context.Context, rule Rule, event Event) (bool, error) {
	window := d.fallback
	if rule.CooldownSeconds > 0 {
		window = time.Duration(rule.CooldownSeconds) * time.Second
	}
	key := fmt.Sprintf("alert:suppress:%s:%s:%s", rule.ID, event.Type, event.IssueID)
	return d.redis.SetNX(ctx, key, "1", window).Result()
}

func (d *Dispatcher) incrementFrequency(ctx context.Context, rule Rule, event Event) (int64, error) {
	window := rule.WindowSeconds
	if window <= 0 {
		window = 300
	}
	bucket := event.OccurredAt.Unix() / int64(window)
	key := fmt.Sprintf("alert:frequency:%s:%s:%d", rule.ID, event.IssueID, bucket)
	count, err := d.redis.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	if count == 1 {
		if err := d.redis.Expire(ctx, key, time.Duration(window)*time.Second).Err(); err != nil {
			return 0, err
		}
	}
	return count, nil
}

func (d *Dispatcher) sendWebhook(ctx context.Context, rule Rule, event Event) error {
	body, err := json.Marshal(WebhookPayload{
		Type:        event.Type,
		ProjectID:   event.ProjectID,
		IssueID:     event.IssueID,
		EventID:     event.EventID,
		Level:       event.Level,
		Title:       event.Title,
		Message:     event.Message,
		Environment: event.Environment,
		Release:     event.Release,
		OccurredAt:  event.OccurredAt,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, rule.WebhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "sentry-lite-alert/0.1")

	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned %s", resp.Status)
	}
	return nil
}

func levelAtLeast(level string, min string) bool {
	return levelWeight(level) >= levelWeight(min)
}

func levelWeight(level string) int {
	switch level {
	case "fatal":
		return 50
	case "error":
		return 40
	case "warning":
		return 30
	case "info":
		return 20
	case "debug":
		return 10
	default:
		return 40
	}
}
