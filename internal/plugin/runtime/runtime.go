package runtime

import "sync"

// Feature defines the activation lifecycle for a runtime plugin feature.
type Feature interface {
	Activate() error
	Deactivate() error
}

// Runtime coordinates activation and deactivation of runtime features.
type Runtime struct {
	mu       sync.Mutex
	features map[string]Feature
}

// New creates a new runtime registry.
func New() *Runtime {
	return &Runtime{features: make(map[string]Feature)}
}

// Register adds a feature implementation for the provided slug.
func (r *Runtime) Register(slug string, feature Feature) {
	if r == nil || feature == nil {
		return
	}

	r.mu.Lock()
	if r.features == nil {
		r.features = make(map[string]Feature)
	}
	r.features[slug] = feature
	r.mu.Unlock()
}

// Activate enables the feature identified by slug if it exists.
func (r *Runtime) Activate(slug string) error {
	if r == nil {
		return nil
	}

	r.mu.Lock()
	feature, ok := r.features[slug]
	r.mu.Unlock()
	if !ok || feature == nil {
		return nil
	}

	return feature.Activate()
}

// Deactivate disables the feature identified by slug if it exists.
func (r *Runtime) Deactivate(slug string) error {
	if r == nil {
		return nil
	}

	r.mu.Lock()
	feature, ok := r.features[slug]
	r.mu.Unlock()
	if !ok || feature == nil {
		return nil
	}

	return feature.Deactivate()
}

// FeatureFunc is a helper to adapt plain functions to the Feature interface.
type FeatureFunc struct {
	ActivateFunc   func() error
	DeactivateFunc func() error
}

// Activate executes the configured activate callback if present.
func (f FeatureFunc) Activate() error {
	if f.ActivateFunc == nil {
		return nil
	}
	return f.ActivateFunc()
}

// Deactivate executes the configured deactivate callback if present.
func (f FeatureFunc) Deactivate() error {
	if f.DeactivateFunc == nil {
		return nil
	}
	return f.DeactivateFunc()
}
