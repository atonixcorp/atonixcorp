package cloud

import (
	"atonixcorp/api/internal/models"
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
)

var (
	ErrProviderNotRegistered = errors.New("infrastructure provider is not registered")
	ErrProviderNameConflict  = errors.New("infrastructure provider name is already registered")
)

var providerNamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]{0,31}$`)

type ComputeProviderFactory interface {
	Name() string
	ForAccount(context.Context, CloudAccountCredentials) (ComputeProvider, error)
}

func (r *ComputeProviderRegistry) DecodeStoredAccount(record models.CloudAccount, plaintext []byte) (CloudAccountCredentials, error) {
	if r == nil {
		return nil, ErrProviderNotRegistered
	}
	r.mu.RLock()
	factory := r.factories[record.Provider]
	r.mu.RUnlock()
	decoder, ok := factory.(AccountCredentialDecoder)
	if !ok {
		return nil, ErrProviderUnavailable
	}
	return decoder.DecodeStoredAccount(record, plaintext)
}

type ComputeProviderRegistry struct {
	mu          sync.RWMutex
	factories   map[string]ComputeProviderFactory
	defaultName string
}

func NewComputeProviderRegistry() *ComputeProviderRegistry {
	return &ComputeProviderRegistry{factories: make(map[string]ComputeProviderFactory)}
}

func (r *ComputeProviderRegistry) Register(factory ComputeProviderFactory) error {
	if r == nil || factory == nil {
		return fmt.Errorf("%w: provider factory is required", ErrInvalid)
	}
	name := factory.Name()
	if name != strings.TrimSpace(name) || !providerNamePattern.MatchString(name) {
		return fmt.Errorf("%w: provider name must be a lowercase identifier", ErrInvalid)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.factories == nil {
		r.factories = make(map[string]ComputeProviderFactory)
	}
	if _, exists := r.factories[name]; exists {
		return fmt.Errorf("%w: %s", ErrProviderNameConflict, name)
	}
	r.factories[name] = factory
	if r.defaultName == "" {
		r.defaultName = name
	}
	return nil
}

func (r *ComputeProviderRegistry) DefaultName() string {
	if r == nil {
		return ""
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.defaultName
}

func (r *ComputeProviderRegistry) ForAccount(
	ctx context.Context,
	name string,
	account CloudAccountCredentials,
) (ComputeProvider, error) {
	if r == nil {
		return nil, ErrProviderNotRegistered
	}
	name = strings.TrimSpace(name)
	r.mu.RLock()
	factory, exists := r.factories[name]
	r.mu.RUnlock()
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrProviderNotRegistered, name)
	}
	provider, err := factory.ForAccount(ctx, account)
	if err != nil {
		return nil, err
	}
	if provider == nil || provider.Name() != name {
		return nil, fmt.Errorf("%w: provider factory returned an invalid adapter", ErrProviderRequest)
	}
	return provider, nil
}
