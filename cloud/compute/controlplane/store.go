package controlplane

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

var ErrNotFound = errors.New("instance not found")
var ErrAlreadyExists = errors.New("resource already exists")

type Store interface {
	Create(instances []Instance) error
	Get(id string) (Instance, error)
	List() ([]Instance, error)
	Update(instance Instance) error
	AddSSHKey(key SSHKey) error
	ListSSHKeys() ([]SSHKey, error)
}

type FileStore struct {
	mu      sync.RWMutex
	path    string
	data    map[string]Instance
	sshKeys map[string]SSHKey
}

type fileStoreState struct {
	Instances map[string]Instance `json:"instances"`
	SSHKeys   map[string]SSHKey   `json:"ssh_keys"`
}

func OpenFileStore(path string) (*FileStore, error) {
	if path == "" {
		return nil, errors.New("instance store path is required")
	}

	store := &FileStore{
		path:    path,
		data:    make(map[string]Instance),
		sshKeys: make(map[string]SSHKey),
	}
	data, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("read instance store: %w", err)
	}
	if err == nil {
		var state fileStoreState
		if err := json.Unmarshal(data, &state); err != nil {
			return nil, fmt.Errorf("decode instance store: %w", err)
		}
		store.data = state.Instances
		store.sshKeys = state.SSHKeys
	}
	if store.data == nil {
		store.data = make(map[string]Instance)
	}
	if store.sshKeys == nil {
		store.sshKeys = make(map[string]SSHKey)
	}
	return store, nil
}

func (s *FileStore) Create(instances []Instance) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	seen := make(map[string]bool, len(instances))
	for _, instance := range instances {
		if _, exists := s.data[instance.ID]; exists {
			return fmt.Errorf("instance %q already exists", instance.ID)
		}
		if seen[instance.ID] {
			return fmt.Errorf("duplicate instance ID %q", instance.ID)
		}
		seen[instance.ID] = true
	}

	for _, instance := range instances {
		s.data[instance.ID] = cloneInstance(instance)
	}
	if err := s.persist(); err != nil {
		for _, instance := range instances {
			delete(s.data, instance.ID)
		}
		return err
	}
	return nil
}

func (s *FileStore) Get(id string) (Instance, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	instance, exists := s.data[id]
	if !exists {
		return Instance{}, ErrNotFound
	}
	return cloneInstance(instance), nil
}

func (s *FileStore) List() ([]Instance, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	instances := make([]Instance, 0, len(s.data))
	for _, instance := range s.data {
		instances = append(instances, cloneInstance(instance))
	}
	sort.Slice(instances, func(i, j int) bool {
		return instances[i].CreatedAt.After(instances[j].CreatedAt)
	})
	return instances, nil
}

func (s *FileStore) Update(instance Instance) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	previous, exists := s.data[instance.ID]
	if !exists {
		return ErrNotFound
	}
	s.data[instance.ID] = cloneInstance(instance)
	if err := s.persist(); err != nil {
		s.data[instance.ID] = previous
		return err
	}
	return nil
}

func (s *FileStore) AddSSHKey(key SSHKey) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.sshKeys[key.ID]; exists {
		return ErrAlreadyExists
	}
	for _, current := range s.sshKeys {
		if current.Fingerprint == key.Fingerprint {
			return ErrAlreadyExists
		}
	}
	s.sshKeys[key.ID] = key
	if err := s.persist(); err != nil {
		delete(s.sshKeys, key.ID)
		return err
	}
	return nil
}

func cloneInstance(instance Instance) Instance {
	instance.SSHKeyIDs = append([]string(nil), instance.SSHKeyIDs...)
	instance.Tags = append([]string(nil), instance.Tags...)
	instance.Volumes = append([]VolumeRequest(nil), instance.Volumes...)
	instance.Provisioning = append([]ProvisioningStep(nil), instance.Provisioning...)
	for i := range instance.Provisioning {
		if instance.Provisioning[i].UpdatedAt != nil {
			updatedAt := *instance.Provisioning[i].UpdatedAt
			instance.Provisioning[i].UpdatedAt = &updatedAt
		}
	}
	return instance
}

func (s *FileStore) ListSSHKeys() ([]SSHKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keys := make([]SSHKey, 0, len(s.sshKeys))
	for _, key := range s.sshKeys {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i].Name < keys[j].Name
	})
	return keys, nil
}

func (s *FileStore) persist() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return fmt.Errorf("create instance store directory: %w", err)
	}

	data, err := json.MarshalIndent(fileStoreState{
		Instances: s.data,
		SSHKeys:   s.sshKeys,
	}, "", "  ")
	if err != nil {
		return fmt.Errorf("encode instance store: %w", err)
	}

	file, err := os.CreateTemp(filepath.Dir(s.path), ".instances-*.json")
	if err != nil {
		return fmt.Errorf("create temporary instance store: %w", err)
	}
	tempPath := file.Name()
	defer os.Remove(tempPath)

	if err := file.Chmod(0o600); err != nil {
		file.Close()
		return fmt.Errorf("secure temporary instance store: %w", err)
	}
	if _, err := file.Write(data); err != nil {
		file.Close()
		return fmt.Errorf("write temporary instance store: %w", err)
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return fmt.Errorf("sync temporary instance store: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close temporary instance store: %w", err)
	}
	if err := os.Rename(tempPath, s.path); err != nil {
		return fmt.Errorf("replace instance store: %w", err)
	}
	return nil
}

func NewID(prefix string) (string, error) {
	bytes := make([]byte, 12)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate ID: %w", err)
	}
	return prefix + "_" + hex.EncodeToString(bytes), nil
}
