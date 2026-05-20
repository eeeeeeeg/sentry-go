package envelope

import "time"

type Envelope struct {
	EventID     string            `json:"event_id"`
	Timestamp   time.Time         `json:"timestamp"`
	Platform    string            `json:"platform"`
	Runtime     Runtime           `json:"runtime"`
	SDK         SDK               `json:"sdk"`
	Level       string            `json:"level"`
	Message     string            `json:"message"`
	Exception   *Exception        `json:"exception,omitempty"`
	Release     string            `json:"release,omitempty"`
	Environment string            `json:"environment,omitempty"`
	Tags        map[string]string `json:"tags,omitempty"`
	User        map[string]any    `json:"user,omitempty"`
	Contexts    map[string]any    `json:"contexts,omitempty"`
	Breadcrumbs []Breadcrumb      `json:"breadcrumbs,omitempty"`
	Fingerprint []string          `json:"fingerprint,omitempty"`
}

type Runtime struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type SDK struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type Exception struct {
	Type       string       `json:"type"`
	Value      string       `json:"value"`
	Stacktrace []StackFrame `json:"stacktrace,omitempty"`
}

type StackFrame struct {
	Filename string `json:"filename,omitempty"`
	Function string `json:"function,omitempty"`
	Module   string `json:"module,omitempty"`
	LineNo   int    `json:"lineno,omitempty"`
	ColNo    int    `json:"colno,omitempty"`
	InApp    bool   `json:"in_app,omitempty"`
}

type Breadcrumb struct {
	Timestamp time.Time      `json:"timestamp,omitempty"`
	Type      string         `json:"type,omitempty"`
	Category  string         `json:"category,omitempty"`
	Level     string         `json:"level,omitempty"`
	Message   string         `json:"message,omitempty"`
	Data      map[string]any `json:"data,omitempty"`
}
