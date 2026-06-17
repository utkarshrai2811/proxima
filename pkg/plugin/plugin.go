// Package plugin implements a sandboxed JavaScript plugin system (via Goja).
// Plugins export a `meta` object and hook functions (onRequest, onResponse,
// onIntercept, onFuzzResult). Hook errors are recorded and never crash the proxy.
package plugin

import (
	"sync"
	"time"

	"github.com/dop251/goja"
)

// Logger is the subset of the app logger the plugin system uses (zap Sugared
// satisfies it).
type Logger interface {
	Infow(msg string, keysAndValues ...interface{})
}

// Plugin is a single loaded JavaScript plugin.
type Plugin struct {
	Name        string
	Version     string
	Description string
	Author      string // set by the plugin developer in their own meta export
	FilePath    string
	Enabled     bool
	LastError   string
	LoadedAt    time.Time

	mu    sync.Mutex // Goja runtimes are single-threaded; guard all vm access
	vm    *goja.Runtime
	hooks map[string]goja.Callable
}

// HookContext is passed to and returned from hooks.
type HookContext struct {
	Request  *RequestCtx
	Response *ResponseCtx
	Result   *FuzzResultCtx
	Action   string // onIntercept: "forward" | "drop"
}

type RequestCtx struct {
	Method  string
	URL     string
	Headers map[string]string
	Body    string
}

type ResponseCtx struct {
	StatusCode int
	Headers    map[string]string
	Body       string
}

type FuzzResultCtx struct {
	AttackID       string
	RequestIndex   int
	PayloadValues  map[string]string
	StatusCode     int
	ResponseSize   int
	ResponseTimeMs int64
	IsError        bool
}

// Notification is a message raised by a plugin via proxima.alert.
type Notification struct {
	Plugin  string    `json:"plugin"`
	Message string    `json:"message"`
	Time    time.Time `json:"time"`
}

// NotificationQueue collects plugin alerts. It is safe for concurrent use.
type NotificationQueue struct {
	mu    sync.Mutex
	items []Notification
}

func NewNotificationQueue() *NotificationQueue {
	return &NotificationQueue{}
}

func (q *NotificationQueue) Push(plugin, message string) {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.items = append(q.items, Notification{Plugin: plugin, Message: message, Time: time.Now()})
	if len(q.items) > 500 {
		q.items = q.items[len(q.items)-500:]
	}
}

func (q *NotificationQueue) List() []Notification {
	q.mu.Lock()
	defer q.mu.Unlock()

	return append([]Notification(nil), q.items...)
}
