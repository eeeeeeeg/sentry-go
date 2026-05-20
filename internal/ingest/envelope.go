package ingest

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

var ErrNoEventItem = errors.New("envelope does not contain an event item")

type decodedEnvelope struct {
	EventID    string
	SDKName    string
	SDKVersion string
	HasEvent   bool
	Items      []EnvelopeItem
	Payload    []byte
}

type EnvelopeItem struct {
	EnvelopeItemMetadata
	Payload json.RawMessage `json:"payload,omitempty"`
}

type EnvelopeItemMetadata struct {
	Type        string `json:"type"`
	Category    string `json:"category"`
	Length      int    `json:"length"`
	ContentType string `json:"content_type,omitempty"`
	Filename    string `json:"filename,omitempty"`
	Attachment  string `json:"attachment_type,omitempty"`
}

func decodeIngestPayload(body []byte) (decodedEnvelope, error) {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return decodedEnvelope{}, fmt.Errorf("request body must not be empty")
	}
	if json.Valid(trimmed) {
		if err := validatePayload(trimmed); err != nil {
			return decodedEnvelope{}, err
		}
		return decodedEnvelope{
			EventID:  extractEventID(trimmed),
			HasEvent: true,
			Items: []EnvelopeItem{{
				EnvelopeItemMetadata: EnvelopeItemMetadata{
					Type:     "event",
					Category: envelopeItemCategory("event"),
					Length:   len(trimmed),
				},
				Payload: json.RawMessage(trimmed),
			}},
			Payload: trimmed,
		}, nil
	}
	return decodeSentryEnvelope(body)
}

func decodeSentryEnvelope(body []byte) (decodedEnvelope, error) {
	reader := bufio.NewReader(bytes.NewReader(body))

	envelopeHeaderLine, err := readJSONLine(reader)
	if err != nil {
		return decodedEnvelope{}, fmt.Errorf("read envelope header: %w", err)
	}
	var envelopeHeader map[string]any
	if err := json.Unmarshal(envelopeHeaderLine, &envelopeHeader); err != nil {
		return decodedEnvelope{}, fmt.Errorf("invalid envelope header")
	}
	envelopeEventID := stringFromHeader(envelopeHeader, "event_id")
	var decoded decodedEnvelope

	for {
		itemHeaderLine, err := readJSONLine(reader)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return decodedEnvelope{}, fmt.Errorf("read envelope item header: %w", err)
		}

		var itemHeader map[string]any
		if err := json.Unmarshal(itemHeaderLine, &itemHeader); err != nil {
			return decodedEnvelope{}, fmt.Errorf("invalid envelope item header")
		}

		itemPayload, err := readEnvelopeItemPayload(reader, itemHeader)
		if err != nil {
			return decodedEnvelope{}, err
		}
		itemMeta := envelopeItemMetadata(itemHeader, len(itemPayload))
		decoded.Items = append(decoded.Items, EnvelopeItem{
			EnvelopeItemMetadata: itemMeta,
			Payload:              json.RawMessage(bytes.TrimSpace(itemPayload)),
		})
		if itemMeta.Type != "event" {
			continue
		}

		itemPayload = bytes.TrimSpace(itemPayload)
		if !json.Valid(itemPayload) {
			return decodedEnvelope{}, fmt.Errorf("event item payload must be valid JSON")
		}
		if err := validatePayload(itemPayload); err != nil {
			return decodedEnvelope{}, err
		}

		sdkName, sdkVersion := sdkFromEnvelopeHeader(envelopeHeader)
		eventID := firstNonEmpty(extractEventID(itemPayload), envelopeEventID)
		return decodedEnvelope{
			EventID:    eventID,
			SDKName:    sdkName,
			SDKVersion: sdkVersion,
			HasEvent:   true,
			Items:      decoded.Items,
			Payload:    itemPayload,
		}, nil
	}

	sdkName, sdkVersion := sdkFromEnvelopeHeader(envelopeHeader)
	return decodedEnvelope{
		EventID:    envelopeEventID,
		SDKName:    sdkName,
		SDKVersion: sdkVersion,
		Items:      decoded.Items,
	}, nil
}

func readJSONLine(reader *bufio.Reader) ([]byte, error) {
	for {
		line, err := reader.ReadBytes('\n')
		line = bytes.TrimSpace(line)
		if len(line) > 0 {
			return line, nil
		}
		if err != nil {
			return line, err
		}
	}
}

func readEnvelopeItemPayload(reader *bufio.Reader, itemHeader map[string]any) ([]byte, error) {
	if length, ok := numericLength(itemHeader["length"]); ok {
		payload := make([]byte, length)
		if _, err := io.ReadFull(reader, payload); err != nil {
			return nil, fmt.Errorf("read envelope item payload: %w", err)
		}
		if next, err := reader.ReadByte(); err == nil && next != '\n' {
			if unreadErr := reader.UnreadByte(); unreadErr != nil {
				return nil, unreadErr
			}
		}
		return payload, nil
	}

	payload, err := reader.ReadBytes('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("read envelope item payload: %w", err)
	}
	return bytes.TrimRight(payload, "\r\n"), nil
}

func itemType(itemHeader map[string]any) string {
	value, _ := itemHeader["type"].(string)
	return strings.ToLower(strings.TrimSpace(value))
}

func envelopeItemMetadata(itemHeader map[string]any, payloadLength int) EnvelopeItemMetadata {
	length := payloadLength
	if headerLength, ok := numericLength(itemHeader["length"]); ok {
		length = headerLength
	}
	return EnvelopeItemMetadata{
		Type:        itemType(itemHeader),
		Category:    envelopeItemCategory(itemType(itemHeader)),
		Length:      length,
		ContentType: stringFromHeader(itemHeader, "content_type"),
		Filename:    stringFromHeader(itemHeader, "filename"),
		Attachment:  stringFromHeader(itemHeader, "attachment_type"),
	}
}

func envelopeItemCategory(itemType string) string {
	switch itemType {
	case "event":
		return "error"
	case "transaction":
		return "transaction"
	case "session", "sessions":
		return "session"
	case "attachment":
		return "attachment"
	case "profile", "profile_chunk":
		return "profile"
	case "replay_event", "replay_recording":
		return "replay"
	case "client_report":
		return "outcome"
	case "check_in":
		return "monitor"
	default:
		return "default"
	}
}

func numericLength(value any) (int, bool) {
	switch typed := value.(type) {
	case float64:
		if typed < 0 || typed != float64(int(typed)) {
			return 0, false
		}
		return int(typed), true
	case json.Number:
		length, err := typed.Int64()
		if err != nil || length < 0 {
			return 0, false
		}
		return int(length), true
	default:
		return 0, false
	}
}

func sdkFromEnvelopeHeader(header map[string]any) (string, string) {
	sdk, _ := header["sdk"].(map[string]any)
	if sdk == nil {
		return "", ""
	}
	name, _ := sdk["name"].(string)
	version, _ := sdk["version"].(string)
	return strings.TrimSpace(name), strings.TrimSpace(version)
}

func stringFromHeader(header map[string]any, key string) string {
	value, _ := header[key].(string)
	return strings.TrimSpace(value)
}
