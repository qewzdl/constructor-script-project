package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
)

// SubtitleManager coordinates subtitle generators and exposes a provider-agnostic
// interface to the rest of the application. It is safe for concurrent use.
type SubtitleManager struct {
	mu               sync.RWMutex
	defaultProvider  string
	generators       map[string]SubtitleGenerator
	providerPriority []string
}

// NewSubtitleManager constructs a new SubtitleManager instance.
func NewSubtitleManager(defaultProvider string) *SubtitleManager {
	manager := &SubtitleManager{
		generators: make(map[string]SubtitleGenerator),
	}
	manager.SetDefaultProvider(defaultProvider)
	return manager
}

// Register attaches a subtitle generator to the manager using the supplied name.
// Names are case-insensitive. Registering the same name twice replaces the
// previous generator.
func (m *SubtitleManager) Register(name string, generator SubtitleGenerator) error {
	if m == nil {
		return fmt.Errorf("subtitle manager is nil")
	}
	trimmed := strings.ToLower(strings.TrimSpace(name))
	if trimmed == "" {
		return fmt.Errorf("subtitle provider name is required")
	}
	if generator == nil {
		return fmt.Errorf("subtitle generator for provider %q is nil", trimmed)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.generators == nil {
		m.generators = make(map[string]SubtitleGenerator)
	}

	_, exists := m.generators[trimmed]
	m.generators[trimmed] = generator
	if !exists {
		m.providerPriority = append(m.providerPriority, trimmed)
		sort.Strings(m.providerPriority)
	}

	return nil
}

// SetDefaultProvider configures the preferred provider. The name is normalised
// to lowercase. The provider does not need to exist at the time of invocation.
func (m *SubtitleManager) SetDefaultProvider(name string) {
	if m == nil {
		return
	}
	m.mu.Lock()
	m.defaultProvider = strings.ToLower(strings.TrimSpace(name))
	m.mu.Unlock()
}

// Providers returns the list of registered provider identifiers sorted in
// ascending order.
func (m *SubtitleManager) Providers() []string {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()

	providers := make([]string, 0, len(m.providerPriority))
	providers = append(providers, m.providerPriority...)
	return providers
}

// Generate delegates the request to the resolved provider. If the request does
// not specify a provider the manager attempts to use the configured default
// provider. When no default is set the first registered provider is used.
func (m *SubtitleManager) Generate(ctx context.Context, request SubtitleGenerationRequest) (*SubtitleResult, error) {
	if m == nil {
		return nil, ErrSubtitleProviderNotConfigured
	}

	provider := strings.ToLower(strings.TrimSpace(request.Provider))

	m.mu.RLock()
	generator, ok := m.generators[provider]
	if !ok {
		if provider == "" {
			if m.defaultProvider != "" {
				generator, ok = m.generators[m.defaultProvider]
				provider = m.defaultProvider
			}
			if !ok && len(m.providerPriority) > 0 {
				provider = m.providerPriority[0]
				generator = m.generators[provider]
				ok = true
			}
		}
	}
	m.mu.RUnlock()

	if !ok || generator == nil {
		return nil, fmt.Errorf("subtitle provider %q is not registered", provider)
	}

	resolvedRequest := request
	resolvedRequest.Provider = provider

	result, err := generator.Generate(ctx, resolvedRequest)
	if err != nil {
		return nil, err
	}

	if result != nil && result.Name == "" && request.PreferredName != "" {
		result.Name = request.PreferredName
	}

	return result, nil
}
