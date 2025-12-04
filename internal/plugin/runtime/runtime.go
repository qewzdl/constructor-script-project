package runtime

import "sync"

// Feature defines the activation lifecycle for a runtime plugin feature.
type Feature interface {
	Activate() error
	Deactivate() error
}

// Runtime coordinates activation and deactivation of runtime features.
type Runtime struct {
	mu        sync.RWMutex
	features  map[string]Feature
	activated map[string]bool
}

// New creates a new runtime registry.
func New() *Runtime {
	return &Runtime{
		features:  make(map[string]Feature),
		activated: make(map[string]bool),
	}
}

// Register adds a feature implementation for the provided slug.
func (r *Runtime) Register(slug string, feature Feature) {
	if r == nil || feature == nil || slug == "" {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.features == nil {
		r.features = make(map[string]Feature)
	}
	if r.activated == nil {
		r.activated = make(map[string]bool)
	}

	r.features[slug] = feature
	// Reset activation state on re-registration
	delete(r.activated, slug)
}

// Activate enables the feature identified by slug if it exists.
func (r *Runtime) Activate(slug string) error {
	if r == nil || slug == "" {
		return nil
	}

	r.mu.Lock()
	feature, ok := r.features[slug]
	isActivated := r.activated[slug]
	r.mu.Unlock()

	if !ok || feature == nil {
		return nil
	}

	// Skip if already activated
	if isActivated {
		return nil
	}

	err := feature.Activate()
	if err != nil {
		return err
	}

	r.mu.Lock()
	r.activated[slug] = true
	r.mu.Unlock()

	return nil
}

// Deactivate disables the feature identified by slug if it exists.
func (r *Runtime) Deactivate(slug string) error {
	if r == nil || slug == "" {
		return nil
	}

	r.mu.Lock()
	feature, ok := r.features[slug]
	isActivated := r.activated[slug]
	r.mu.Unlock()

	if !ok || feature == nil {
		return nil
	}

	// Skip if not activated
	if !isActivated {
		return nil
	}

	err := feature.Deactivate()
	if err != nil {
		return err
	}

	r.mu.Lock()
	r.activated[slug] = false
	r.mu.Unlock()

	return nil
}

// Unregister removes a feature implementation for the provided slug.
// This cleans up memory by removing references to the feature.
func (r *Runtime) Unregister(slug string) error {
	if r == nil || slug == "" {
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	feature, ok := r.features[slug]
	if !ok {
		return nil
	}

	// Deactivate if currently active
	if r.activated[slug] && feature != nil {
		if err := feature.Deactivate(); err != nil {
			return err
		}
	}

	// Remove from both maps
	delete(r.features, slug)
	delete(r.activated, slug)

	return nil
}

// Clear removes all registered features and cleans up memory.
// Should be called during shutdown to ensure proper cleanup.
func (r *Runtime) Clear() error {
	if r == nil {
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Deactivate all activated features
	for slug, feature := range r.features {
		if r.activated[slug] && feature != nil {
			if err := feature.Deactivate(); err != nil {
				// Log but continue cleanup
				_ = err
			}
		}
	}

	// Clear all references
	r.features = make(map[string]Feature)
	r.activated = make(map[string]bool)

	return nil
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
