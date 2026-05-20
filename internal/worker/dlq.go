package worker

import (
	"encoding/json"
	"time"

	"github.com/nats-io/nats.go"
)

type DLQMessage struct {
	Subject   string    `json:"subject"`
	Error     string    `json:"error"`
	FailedAt  time.Time `json:"failed_at"`
	Payload   string    `json:"payload"`
	Worker    string    `json:"worker"`
	MessageID string    `json:"message_id,omitempty"`
}

func PublishDLQ(js nats.JetStreamContext, subject string, workerName string, source *nats.Msg, cause error) error {
	message := DLQMessage{
		Subject:  source.Subject,
		Error:    cause.Error(),
		FailedAt: time.Now().UTC(),
		Payload:  string(source.Data),
		Worker:   workerName,
	}
	if source.Header != nil {
		message.MessageID = source.Header.Get(nats.MsgIdHdr)
	}

	body, err := json.Marshal(message)
	if err != nil {
		return err
	}
	_, err = js.Publish(subject, body)
	return err
}
