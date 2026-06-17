package plugin

import (
	"fmt"
	"sync"
	"time"
)

// Info is a snapshot of a plugin's public state (safe to hand to the API).
type Info struct {
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	Description string    `json:"description"`
	Author      string    `json:"author"`
	Enabled     bool      `json:"enabled"`
	LastError   string    `json:"lastError"`
	LoadedAt    time.Time `json:"loadedAt"`
}

// Manager loads and coordinates plugins.
type Manager struct {
	pluginDir     string
	mu            sync.RWMutex
	plugins       []*Plugin
	logger        Logger
	notifications *NotificationQueue
}

func NewManager(pluginDir string, logger Logger, notifications *NotificationQueue) *Manager {
	if notifications == nil {
		notifications = NewNotificationQueue()
	}

	return &Manager{pluginDir: pluginDir, logger: logger, notifications: notifications}
}

func (m *Manager) PluginDir() string                 { return m.pluginDir }
func (m *Manager) Notifications() *NotificationQueue { return m.notifications }

// LoadAll (re)loads every plugin from the plugin directory. Successfully loaded
// plugins are enabled by default.
func (m *Manager) LoadAll() error {
	plugins, err := LoadAll(m.pluginDir, m.logger, m.notifications)
	if err != nil {
		return err
	}

	for _, p := range plugins {
		if p.LastError == "" {
			p.Enabled = true
		}
	}

	m.mu.Lock()
	m.plugins = plugins
	m.mu.Unlock()

	return nil
}

func (m *Manager) List() []Info {
	m.mu.RLock()
	plugins := append([]*Plugin(nil), m.plugins...)
	m.mu.RUnlock()

	out := make([]Info, 0, len(plugins))

	for _, p := range plugins {
		p.mu.Lock()
		out = append(out, Info{
			Name: p.Name, Version: p.Version, Description: p.Description, Author: p.Author,
			Enabled: p.Enabled, LastError: p.LastError, LoadedAt: p.LoadedAt,
		})
		p.mu.Unlock()
	}

	return out
}

func (m *Manager) find(name string) *Plugin {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, p := range m.plugins {
		if p.Name == name {
			return p
		}
	}

	return nil
}

func (m *Manager) Enable(name string) error  { return m.setEnabled(name, true) }
func (m *Manager) Disable(name string) error { return m.setEnabled(name, false) }

func (m *Manager) setEnabled(name string, enabled bool) error {
	p := m.find(name)
	if p == nil {
		return fmt.Errorf("plugin: %q not found", name)
	}

	p.mu.Lock()
	p.Enabled = enabled
	p.mu.Unlock()

	return nil
}

// Reload re-reads and re-runs a plugin from its file, preserving enabled state.
func (m *Manager) Reload(name string) error {
	p := m.find(name)
	if p == nil {
		return fmt.Errorf("plugin: %q not found", name)
	}

	p.mu.Lock()
	enabled := p.Enabled
	p.mu.Unlock()

	reloaded, err := LoadPlugin(p.FilePath, m.logger, m.notifications)
	if err != nil {
		p.mu.Lock()
		p.LastError = err.Error()
		p.mu.Unlock()

		return err
	}

	reloaded.Enabled = enabled

	m.mu.Lock()
	for i, pp := range m.plugins {
		if pp.Name == name {
			m.plugins[i] = reloaded

			break
		}
	}
	m.mu.Unlock()

	return nil
}

func (m *Manager) enabledPlugins() []*Plugin {
	m.mu.RLock()
	plugins := append([]*Plugin(nil), m.plugins...)
	m.mu.RUnlock()

	var out []*Plugin

	for _, p := range plugins {
		p.mu.Lock()
		ok := p.Enabled && len(p.hooks) > 0
		p.mu.Unlock()

		if ok {
			out = append(out, p)
		}
	}

	return out
}
