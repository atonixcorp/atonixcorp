package storage

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"atonixcorp/api/internal/cloud"
)

var ErrDriverNameConflict = errors.New("storage driver name is already registered")

var driverNamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]{0,63}$`)

type ScopedBlockDriverFactory interface {
	Name() string
	ForAccount(context.Context, cloud.CloudAccountCredentials) (BlockDriver, error)
}

type BlockDriverRegistry struct {
	mu                 sync.RWMutex
	drivers            map[string]BlockDriver
	factories          map[string]ScopedBlockDriverFactory
	defaultFactoryName string
}

func NewBlockDriverRegistry() *BlockDriverRegistry {
	return &BlockDriverRegistry{
		drivers:   make(map[string]BlockDriver),
		factories: make(map[string]ScopedBlockDriverFactory),
	}
}

func (r *BlockDriverRegistry) Register(driver BlockDriver) error {
	if r == nil || driver == nil {
		return fmt.Errorf("%w: storage driver is required", ErrInvalid)
	}
	name := driver.Name()
	if name != strings.TrimSpace(name) || !driverNamePattern.MatchString(name) {
		return fmt.Errorf("%w: storage driver name must be a lowercase identifier", ErrInvalid)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.drivers == nil {
		r.drivers = make(map[string]BlockDriver)
	}
	if r.factories == nil {
		r.factories = make(map[string]ScopedBlockDriverFactory)
	}
	if _, exists := r.drivers[name]; exists {
		return fmt.Errorf("%w: %s", ErrDriverNameConflict, name)
	}
	if _, exists := r.factories[name]; exists {
		return fmt.Errorf("%w: %s", ErrDriverNameConflict, name)
	}
	r.drivers[name] = driver
	return nil
}

func (r *BlockDriverRegistry) RegisterFactory(factory ScopedBlockDriverFactory) error {
	if r == nil || factory == nil {
		return fmt.Errorf("%w: scoped storage driver factory is required", ErrInvalid)
	}
	name := factory.Name()
	if name != strings.TrimSpace(name) || !driverNamePattern.MatchString(name) {
		return fmt.Errorf("%w: storage driver name must be a lowercase identifier", ErrInvalid)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.drivers == nil {
		r.drivers = make(map[string]BlockDriver)
	}
	if r.factories == nil {
		r.factories = make(map[string]ScopedBlockDriverFactory)
	}
	if _, exists := r.drivers[name]; exists {
		return fmt.Errorf("%w: %s", ErrDriverNameConflict, name)
	}
	if _, exists := r.factories[name]; exists {
		return fmt.Errorf("%w: %s", ErrDriverNameConflict, name)
	}
	r.factories[name] = factory
	if r.defaultFactoryName == "" {
		r.defaultFactoryName = name
	}
	return nil
}

func (r *BlockDriverRegistry) Driver(name string) (BlockDriver, error) {
	if r == nil {
		return nil, ErrDriverUnavailable
	}
	r.mu.RLock()
	driver, exists := r.drivers[name]
	r.mu.RUnlock()
	if !exists {
		return nil, ErrDriverUnavailable
	}
	return driver, nil
}

func (r *BlockDriverRegistry) DefaultFactoryName() string {
	if r == nil {
		return ""
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.defaultFactoryName
}

func (r *BlockDriverRegistry) ForAccount(
	ctx context.Context,
	name string,
	account cloud.CloudAccountCredentials,
) (BlockDriver, error) {
	if r == nil {
		return nil, ErrDriverUnavailable
	}
	r.mu.RLock()
	factory, exists := r.factories[name]
	r.mu.RUnlock()
	if !exists {
		return nil, ErrDriverUnavailable
	}
	driver, err := factory.ForAccount(ctx, account)
	if err != nil {
		return nil, err
	}
	if driver == nil || driver.Name() != name {
		return nil, errors.New("storage driver factory returned an invalid adapter")
	}
	return driver, nil
}
