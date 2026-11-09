package core

import "errors"

var ErrProvisioningFailed = errors.New("database infrastructure provisioning failed")

type ProvisioningProgress struct {
	InfrastructureReady bool
}

type DatabaseProvisioner interface {
	Name() string
	Supports(engine, version, plan, region string) bool
	Provision(instance *DBInstance, checkpoint func(stage string) error) (ProvisioningProgress, error)
}
