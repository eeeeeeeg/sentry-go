package platform

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	clickhouse "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"

	"sentry-lite/internal/config"
)

type Dependencies struct {
	Postgres   *pgxpool.Pool
	ClickHouse *sql.DB
	Redis      *redis.Client
	NATS       *nats.Conn
	JetStream  nats.JetStreamContext
	Config     config.Config
}

type CheckResult struct {
	Name  string `json:"name"`
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

func Connect(ctx context.Context, cfg config.Config) (*Dependencies, error) {
	pg, err := pgxpool.New(ctx, cfg.PostgresDSN)
	if err != nil {
		return nil, fmt.Errorf("postgres: %w", err)
	}

	ch := clickhouse.OpenDB(&clickhouse.Options{
		Addr: []string{cfg.ClickHouseAddr},
		Auth: clickhouse.Auth{
			Database: cfg.ClickHouseDatabase,
			Username: cfg.ClickHouseUsername,
			Password: cfg.ClickHousePassword,
		},
		DialTimeout: 5 * time.Second,
	})

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	nc, err := nats.Connect(cfg.NATSURL, nats.Name("sentry-lite-api"), nats.Timeout(5*time.Second))
	if err != nil {
		pg.Close()
		_ = ch.Close()
		_ = rdb.Close()
		return nil, fmt.Errorf("nats: %w", err)
	}
	js, err := nc.JetStream()
	if err != nil {
		pg.Close()
		_ = ch.Close()
		_ = rdb.Close()
		nc.Close()
		return nil, fmt.Errorf("jetstream: %w", err)
	}
	if err := ensureStreams(js, cfg); err != nil {
		pg.Close()
		_ = ch.Close()
		_ = rdb.Close()
		nc.Close()
		return nil, fmt.Errorf("ensure streams: %w", err)
	}

	return &Dependencies{
		Postgres:   pg,
		ClickHouse: ch,
		Redis:      rdb,
		NATS:       nc,
		JetStream:  js,
		Config:     cfg,
	}, nil
}

func (d *Dependencies) Close() {
	if d == nil {
		return
	}
	if d.Postgres != nil {
		d.Postgres.Close()
	}
	if d.ClickHouse != nil {
		_ = d.ClickHouse.Close()
	}
	if d.Redis != nil {
		_ = d.Redis.Close()
	}
	if d.NATS != nil {
		d.NATS.Drain()
	}
}

func (d *Dependencies) Check(ctx context.Context) []CheckResult {
	checks := []CheckResult{
		{Name: "postgres"},
		{Name: "clickhouse"},
		{Name: "redis"},
		{Name: "nats"},
	}

	if err := d.Postgres.Ping(ctx); err != nil {
		checks[0].OK = false
		checks[0].Error = err.Error()
	} else {
		checks[0].OK = true
	}

	if err := d.ClickHouse.PingContext(ctx); err != nil {
		checks[1].OK = false
		checks[1].Error = err.Error()
	} else {
		checks[1].OK = true
	}

	if err := d.Redis.Ping(ctx).Err(); err != nil {
		checks[2].OK = false
		checks[2].Error = err.Error()
	} else {
		checks[2].OK = true
	}

	if d.NATS == nil || !d.NATS.IsConnected() {
		checks[3].OK = false
		checks[3].Error = "not connected"
	} else if err := d.ensureJetStream(); err != nil {
		checks[3].OK = false
		checks[3].Error = err.Error()
	} else {
		checks[3].OK = true
	}

	return checks
}

func (d *Dependencies) Ready(ctx context.Context) error {
	var errs []error
	for _, check := range d.Check(ctx) {
		if !check.OK {
			errs = append(errs, fmt.Errorf("%s: %s", check.Name, check.Error))
		}
	}
	return errors.Join(errs...)
}

func (d *Dependencies) ensureJetStream() error {
	if d.JetStream == nil {
		return errors.New("jetstream is not initialized")
	}
	_, err := d.JetStream.StreamInfo(d.Config.RawEventStream)
	return err
}

func ensureStreams(js nats.JetStreamContext, cfg config.Config) error {
	subjects := []string{
		cfg.RawEventSubject,
		cfg.RawTransactionSubject,
		cfg.RawSessionSubject,
		cfg.RawAttachmentSubject,
		cfg.RawProfileSubject,
		cfg.RawReplaySubject,
		cfg.RawOutcomeSubject,
		cfg.UnsupportedItemSubject,
		cfg.NormalizedEventSubject,
		cfg.GroupedEventSubject,
		cfg.AlertEventSubject,
		cfg.DeadLetterSubject,
	}

	stream, err := js.StreamInfo(cfg.RawEventStream)
	if err == nil {
		needsUpdate := false
		existing := make(map[string]struct{}, len(stream.Config.Subjects))
		for _, subject := range stream.Config.Subjects {
			existing[subject] = struct{}{}
		}
		for _, subject := range subjects {
			if _, ok := existing[subject]; !ok {
				needsUpdate = true
				stream.Config.Subjects = append(stream.Config.Subjects, subject)
			}
		}
		if !needsUpdate {
			return nil
		}
		_, err = js.UpdateStream(&stream.Config)
		return err
	}
	if !errors.Is(err, nats.ErrStreamNotFound) {
		return err
	}

	_, err = js.AddStream(&nats.StreamConfig{
		Name:      cfg.RawEventStream,
		Subjects:  subjects,
		Retention: nats.LimitsPolicy,
		Storage:   nats.FileStorage,
		MaxAge:    7 * 24 * time.Hour,
	})
	return err
}
