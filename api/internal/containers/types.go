package containers

import (
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"regexp"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

var (
	deploymentNamePattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$`)
	envNamePattern        = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	imageDigestPattern    = regexp.MustCompile(`@sha256:[a-f0-9]{64}$`)
)

type EnvironmentVariable struct {
	Name   string `json:"name"`
	Value  string `json:"value,omitempty"`
	Secret bool   `json:"secret,omitempty"`
}

type Port struct {
	Name      string `json:"name,omitempty"`
	Container int32  `json:"container_port"`
	Service   int32  `json:"service_port,omitempty"`
	Protocol  string `json:"protocol,omitempty"`
}

type Volume struct {
	Name         string `json:"name"`
	SizeGiB      int32  `json:"size_gib"`
	MountPath    string `json:"mount_path"`
	StorageClass string `json:"storage_class,omitempty"`
	AccessMode   string `json:"access_mode,omitempty"`
}

type Probe struct {
	Path                string `json:"path,omitempty"`
	Port                int32  `json:"port,omitempty"`
	InitialDelaySeconds int32  `json:"initial_delay_seconds,omitempty"`
	PeriodSeconds       int32  `json:"period_seconds,omitempty"`
	TimeoutSeconds      int32  `json:"timeout_seconds,omitempty"`
	FailureThreshold    int32  `json:"failure_threshold,omitempty"`
}

type DeploymentSpec struct {
	Name            string                `json:"name"`
	Description     string                `json:"description,omitempty"`
	Environment     string                `json:"environment"`
	Image           string                `json:"image"`
	ImagePullPolicy corev1.PullPolicy     `json:"image_pull_policy,omitempty"`
	Replicas        int32                 `json:"replicas"`
	CPURequest      string                `json:"cpu_request"`
	CPULimit        string                `json:"cpu_limit"`
	MemoryRequest   string                `json:"memory_request"`
	MemoryLimit     string                `json:"memory_limit"`
	Ports           []Port                `json:"ports,omitempty"`
	EnvironmentVars []EnvironmentVariable `json:"environment_variables,omitempty"`
	Volumes         []Volume              `json:"volumes,omitempty"`
	RetainVolumes   bool                  `json:"retain_volumes,omitempty"`
	ReadinessProbe  *Probe                `json:"readiness_probe,omitempty"`
	LivenessProbe   *Probe                `json:"liveness_probe,omitempty"`
}

type DeploymentResponse struct {
	ID                string         `json:"id"`
	TenantID          uint           `json:"tenant_id"`
	ProjectID         uint           `json:"project_id"`
	Name              string         `json:"name"`
	State             string         `json:"state"`
	CurrentRevision   uint           `json:"current_revision"`
	DesiredReplicas   int32          `json:"desired_replicas"`
	ReadyReplicas     int32          `json:"ready_replicas,omitempty"`
	Spec              DeploymentSpec `json:"spec"`
	Endpoint          string         `json:"endpoint,omitempty"`
	CorrelationID     string         `json:"correlation_id"`
	ProvisioningJobID string         `json:"provisioning_job_id"`
	ErrorCode         string         `json:"error_code,omitempty"`
	ErrorDetails      string         `json:"error_details,omitempty"`
	CreatedAt         string         `json:"created_at"`
	UpdatedAt         string         `json:"updated_at"`
}

type RevisionResponse struct {
	Revision  uint           `json:"revision"`
	SpecHash  string         `json:"spec_hash"`
	ActorID   uint           `json:"actor_id"`
	CreatedAt string         `json:"created_at"`
	Spec      DeploymentSpec `json:"spec"`
}

type DeploymentMetrics struct {
	CPUUsageMillis   int64  `json:"cpu_usage_millis"`
	MemoryUsageBytes int64  `json:"memory_usage_bytes"`
	SampledAt        string `json:"sampled_at"`
}

func ValidateSpec(spec *DeploymentSpec) error {
	spec.Name = strings.ToLower(strings.TrimSpace(spec.Name))
	spec.Image = strings.TrimSpace(spec.Image)
	spec.Environment = strings.ToLower(strings.TrimSpace(spec.Environment))
	if !deploymentNamePattern.MatchString(spec.Name) {
		return errors.New("name must be a lowercase DNS label with up to 63 characters")
	}
	if spec.Image == "" || len(spec.Image) > 512 || strings.ContainsAny(spec.Image, " \t\r\n") {
		return errors.New("a valid container image reference is required")
	}
	if spec.Environment != "development" && spec.Environment != "staging" && spec.Environment != "production" {
		return errors.New("environment must be development, staging, or production")
	}
	if spec.Environment == "production" && !imageDigestPattern.MatchString(spec.Image) {
		return errors.New("production images must be pinned by an immutable sha256 digest")
	}
	if spec.Replicas < 1 || spec.Replicas > 50 {
		return errors.New("replicas must be between 1 and 50")
	}
	if spec.Description != "" && len(spec.Description) > 2000 {
		return errors.New("description must not exceed 2000 characters")
	}
	if spec.ImagePullPolicy == "" {
		spec.ImagePullPolicy = corev1.PullIfNotPresent
	}
	if spec.ImagePullPolicy != corev1.PullAlways &&
		spec.ImagePullPolicy != corev1.PullIfNotPresent &&
		spec.ImagePullPolicy != corev1.PullNever {
		return errors.New("image pull policy must be Always, IfNotPresent, or Never")
	}
	if err := validateResourcePair("cpu", spec.CPURequest, spec.CPULimit, "10m", "16"); err != nil {
		return err
	}
	if err := validateResourcePair("memory", spec.MemoryRequest, spec.MemoryLimit, "16Mi", "64Gi"); err != nil {
		return err
	}
	if len(spec.Ports) > 32 {
		return errors.New("at most 32 container ports are allowed")
	}
	portNumbers := make(map[int32]struct{}, len(spec.Ports))
	for i := range spec.Ports {
		port := &spec.Ports[i]
		if port.Container < 1 || port.Container > 65535 {
			return errors.New("container ports must be between 1 and 65535")
		}
		if _, exists := portNumbers[port.Container]; exists {
			return errors.New("container ports must be unique")
		}
		portNumbers[port.Container] = struct{}{}
		if port.Service == 0 {
			port.Service = port.Container
		}
		if port.Service < 1 || port.Service > 65535 {
			return errors.New("service ports must be between 1 and 65535")
		}
		if port.Protocol == "" {
			port.Protocol = string(corev1.ProtocolTCP)
		}
		if port.Protocol != string(corev1.ProtocolTCP) && port.Protocol != string(corev1.ProtocolUDP) {
			return errors.New("port protocol must be TCP or UDP")
		}
		if port.Name == "" {
			port.Name = fmt.Sprintf("port-%d", port.Container)
		}
		if !deploymentNamePattern.MatchString(port.Name) {
			return errors.New("port names must be valid lowercase DNS labels")
		}
	}
	if len(spec.EnvironmentVars) > 128 {
		return errors.New("at most 128 environment variables are allowed")
	}
	envNames := make(map[string]struct{}, len(spec.EnvironmentVars))
	for _, variable := range spec.EnvironmentVars {
		if !envNamePattern.MatchString(variable.Name) {
			return fmt.Errorf("invalid environment variable name %q", variable.Name)
		}
		if _, exists := envNames[variable.Name]; exists {
			return errors.New("environment variable names must be unique")
		}
		envNames[variable.Name] = struct{}{}
		if len(variable.Value) > 4096 {
			return fmt.Errorf("environment variable %q exceeds 4096 characters", variable.Name)
		}
	}
	if len(spec.Volumes) > 16 {
		return errors.New("at most 16 persistent volumes are allowed")
	}
	volumeNames := make(map[string]struct{}, len(spec.Volumes))
	mountPaths := make(map[string]struct{}, len(spec.Volumes))
	for i := range spec.Volumes {
		volume := &spec.Volumes[i]
		if !deploymentNamePattern.MatchString(volume.Name) || volume.SizeGiB < 1 || volume.SizeGiB > 1000 {
			return errors.New("volumes need a valid name and size between 1 and 1000 GiB")
		}
		if !path.IsAbs(volume.MountPath) || path.Clean(volume.MountPath) != volume.MountPath ||
			volume.MountPath == "/" || strings.HasPrefix(volume.MountPath, "/proc") ||
			strings.HasPrefix(volume.MountPath, "/sys") || strings.HasPrefix(volume.MountPath, "/dev") {
			return fmt.Errorf("volume %q has an unsafe mount path", volume.Name)
		}
		if _, exists := volumeNames[volume.Name]; exists {
			return errors.New("volume names must be unique")
		}
		if _, exists := mountPaths[volume.MountPath]; exists {
			return errors.New("volume mount paths must be unique")
		}
		volumeNames[volume.Name] = struct{}{}
		mountPaths[volume.MountPath] = struct{}{}
		if volume.AccessMode == "" {
			volume.AccessMode = string(corev1.ReadWriteOnce)
		}
		if volume.AccessMode != string(corev1.ReadWriteOnce) {
			return errors.New("only ReadWriteOnce storage is supported by this runtime")
		}
		if len(volume.StorageClass) > 253 {
			return errors.New("storage class name is too long")
		}
	}
	if err := validateProbe(spec.ReadinessProbe, portNumbers); err != nil {
		return fmt.Errorf("readiness probe: %w", err)
	}
	if err := validateProbe(spec.LivenessProbe, portNumbers); err != nil {
		return fmt.Errorf("liveness probe: %w", err)
	}
	if _, err := json.Marshal(spec); err != nil {
		return errors.New("deployment configuration is invalid")
	}
	return nil
}

func validateResourcePair(name, request, limit, minimum, maximum string) error {
	requestQuantity, err := resource.ParseQuantity(request)
	if err != nil {
		return fmt.Errorf("%s request is invalid", name)
	}
	limitQuantity, err := resource.ParseQuantity(limit)
	if err != nil {
		return fmt.Errorf("%s limit is invalid", name)
	}
	minQuantity := resource.MustParse(minimum)
	maxQuantity := resource.MustParse(maximum)
	if requestQuantity.Sign() <= 0 || requestQuantity.Cmp(minQuantity) < 0 ||
		requestQuantity.Cmp(limitQuantity) > 0 || limitQuantity.Cmp(maxQuantity) > 0 {
		return fmt.Errorf("%s request and limit are outside supported limits", name)
	}
	return nil
}

func validateProbe(probe *Probe, portNumbers map[int32]struct{}) error {
	if probe == nil {
		return nil
	}
	if probe.Path != "" && (!strings.HasPrefix(probe.Path, "/") || len(probe.Path) > 1024) {
		return errors.New("path must be an absolute URL path")
	}
	if probe.Port == 0 || probe.Path == "" {
		return errors.New("path and a configured container port are required")
	}
	if _, exists := portNumbers[probe.Port]; !exists {
		return errors.New("probe port must match a configured container port")
	}
	if probe.InitialDelaySeconds < 0 || probe.InitialDelaySeconds > 3600 ||
		probe.PeriodSeconds < 0 || probe.PeriodSeconds > 300 ||
		probe.TimeoutSeconds < 0 || probe.TimeoutSeconds > 300 ||
		probe.FailureThreshold < 0 || probe.FailureThreshold > 100 {
		return errors.New("probe timings are outside supported limits")
	}
	return nil
}

func hideSecretValues(spec DeploymentSpec) (DeploymentSpec, map[string]string) {
	secrets := make(map[string]string)
	for i := range spec.EnvironmentVars {
		if spec.EnvironmentVars[i].Secret {
			secrets[spec.EnvironmentVars[i].Name] = spec.EnvironmentVars[i].Value
			spec.EnvironmentVars[i].Value = ""
		}
	}
	return spec, secrets
}

func preserveSecretValues(spec *DeploymentSpec, previous DeploymentSpec, secrets map[string]string) {
	for i := range spec.EnvironmentVars {
		current := &spec.EnvironmentVars[i]
		if !current.Secret || current.Value != "" {
			continue
		}
		for _, prior := range previous.EnvironmentVars {
			if prior.Secret && prior.Name == current.Name {
				current.Value = secrets[current.Name]
				break
			}
		}
	}
}
