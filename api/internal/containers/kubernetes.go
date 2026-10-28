package containers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Provisioner interface {
	Create(context.Context, string, uint, uint, string, DeploymentSpec, map[string]string, func(string) error) (string, error)
	WaitReady(context.Context, string, string, int32) error
	WaitReplicas(context.Context, string, string, int32) error
	Scale(context.Context, string, string, int32) error
	Restart(context.Context, string, string) error
	Delete(context.Context, string, string, string, bool) error
	Logs(context.Context, string, string) (string, error)
	Events(context.Context, string, string) ([]corev1.Event, error)
	Instances(context.Context, string, string) ([]corev1.Pod, error)
	Status(context.Context, string, string) (appsv1.DeploymentStatus, error)
	Metrics(context.Context, string, string) (DeploymentMetrics, error)
}

type KubernetesProvisioner struct {
	client kubernetes.Interface
}

func NewKubernetesProvisioner() (*KubernetesProvisioner, error) {
	var config *rest.Config
	var err error
	if path := os.Getenv("KUBECONFIG"); path != "" {
		config, err = clientcmd.BuildConfigFromFlags("", path)
	} else {
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, nil
		}
	}
	if err != nil {
		return nil, fmt.Errorf("load Kubernetes credentials: %w", err)
	}
	config.UserAgent = "atonixcorp-container-control-plane"
	config.Timeout = 15 * time.Second
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("create Kubernetes client: %w", err)
	}
	return &KubernetesProvisioner{client: client}, nil
}

func (k *KubernetesProvisioner) Create(
	ctx context.Context,
	namespace string,
	tenantID, projectID uint,
	deploymentID string,
	spec DeploymentSpec,
	secrets map[string]string,
	stage func(string) error,
) (string, error) {
	if err := stage("CREATING_NETWORK"); err != nil {
		return "", err
	}
	if err := k.ensureNamespace(ctx, namespace, tenantID, projectID); err != nil {
		return "", err
	}

	labels := map[string]string{
		"app.kubernetes.io/managed-by": "atonixcorp",
		"atonixcorp.io/deployment-id":  deploymentID,
	}
	if len(secrets) > 0 {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:   deploymentID + "-env",
				Labels: labels,
			},
			Type:       corev1.SecretTypeOpaque,
			StringData: secrets,
		}
		if _, err := k.client.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{}); err != nil {
			if !apierrors.IsAlreadyExists(err) {
				return "", fmt.Errorf("create workload secrets: %w", err)
			}
			existing, getErr := k.client.CoreV1().Secrets(namespace).Get(ctx, secret.Name, metav1.GetOptions{})
			if getErr != nil {
				return "", fmt.Errorf("get existing workload secrets: %w", getErr)
			}
			secret.ResourceVersion = existing.ResourceVersion
			if _, err := k.client.CoreV1().Secrets(namespace).Update(ctx, secret, metav1.UpdateOptions{}); err != nil {
				return "", fmt.Errorf("update workload secrets: %w", err)
			}
		}
	}

	volumeMounts := make([]corev1.VolumeMount, 0, len(spec.Volumes))
	volumes := make([]corev1.Volume, 0, len(spec.Volumes))
	if len(spec.Volumes) > 0 {
		if err := stage("CREATING_STORAGE"); err != nil {
			return "", err
		}
	}
	for _, volume := range spec.Volumes {
		claimName := deploymentID + "-" + volume.Name
		storageClass := volume.StorageClass
		claim := &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{Name: claimName, Labels: labels},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{corev1.PersistentVolumeAccessMode(volume.AccessMode)},
				Resources: corev1.VolumeResourceRequirements{
					Requests: corev1.ResourceList{corev1.ResourceStorage: resource.MustParse(fmt.Sprintf("%dGi", volume.SizeGiB))},
				},
				StorageClassName: optionalString(storageClass),
			},
		}
		if _, err := k.client.CoreV1().PersistentVolumeClaims(namespace).Create(ctx, claim, metav1.CreateOptions{}); err != nil && !apierrors.IsAlreadyExists(err) {
			return "", fmt.Errorf("create persistent volume %q: %w", volume.Name, err)
		}
		volumes = append(volumes, corev1.Volume{
			Name: volume.Name,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: claimName},
			},
		})
		volumeMounts = append(volumeMounts, corev1.VolumeMount{Name: volume.Name, MountPath: volume.MountPath})
	}

	environment := make([]corev1.EnvVar, 0, len(spec.EnvironmentVars))
	for _, variable := range spec.EnvironmentVars {
		if variable.Secret {
			environment = append(environment, corev1.EnvVar{
				Name: variable.Name,
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: deploymentID + "-env"},
						Key:                  variable.Name,
					},
				},
			})
			continue
		}
		environment = append(environment, corev1.EnvVar{Name: variable.Name, Value: variable.Value})
	}

	containerPorts := make([]corev1.ContainerPort, 0, len(spec.Ports))
	servicePorts := make([]corev1.ServicePort, 0, len(spec.Ports))
	for _, port := range spec.Ports {
		protocol := corev1.Protocol(port.Protocol)
		containerPorts = append(containerPorts, corev1.ContainerPort{
			Name: port.Name, ContainerPort: port.Container, Protocol: protocol,
		})
		servicePorts = append(servicePorts, corev1.ServicePort{
			Name: port.Name, Port: port.Service, TargetPort: intstr.FromInt32(port.Container), Protocol: protocol,
		})
	}

	probePort := func(port int32) intstr.IntOrString {
		return intstr.FromInt32(port)
	}
	container := corev1.Container{
		Name:            "application",
		Image:           spec.Image,
		ImagePullPolicy: spec.ImagePullPolicy,
		Ports:           containerPorts,
		Env:             environment,
		VolumeMounts:    volumeMounts,
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse(spec.CPURequest),
				corev1.ResourceMemory: resource.MustParse(spec.MemoryRequest),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse(spec.CPULimit),
				corev1.ResourceMemory: resource.MustParse(spec.MemoryLimit),
			},
		},
		SecurityContext: &corev1.SecurityContext{
			RunAsNonRoot:             boolPointer(true),
			RunAsUser:                int64Pointer(10001),
			AllowPrivilegeEscalation: boolPointer(false),
			ReadOnlyRootFilesystem:   boolPointer(true),
			Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
			SeccompProfile:           &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault},
		},
	}
	if spec.ReadinessProbe != nil {
		container.ReadinessProbe = buildProbe(*spec.ReadinessProbe, probePort)
	}
	if spec.LivenessProbe != nil {
		container.LivenessProbe = buildProbe(*spec.LivenessProbe, probePort)
	}
	if err := stage("CONFIGURING_HEALTH_CHECKS"); err != nil {
		return "", err
	}
	if err := stage("PULLING_IMAGE"); err != nil {
		return "", err
	}
	if err := stage("CREATING_COMPUTE"); err != nil {
		return "", err
	}
	if err := stage("CREATING_CONTAINER"); err != nil {
		return "", err
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: spec.Name, Labels: labels},
		Spec: appsv1.DeploymentSpec{
			Replicas: &spec.Replicas,
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"atonixcorp.io/deployment-id": deploymentID}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{
					"app.kubernetes.io/managed-by": "atonixcorp",
					"atonixcorp.io/deployment-id":  deploymentID,
				}},
				Spec: corev1.PodSpec{
					AutomountServiceAccountToken: boolPointer(false),
					RestartPolicy:                corev1.RestartPolicyAlways,
					SecurityContext:              &corev1.PodSecurityContext{RunAsNonRoot: boolPointer(true), SeccompProfile: &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault}},
					Containers:                   []corev1.Container{container},
					Volumes:                      volumes,
				},
			},
		},
	}
	if _, err := k.client.AppsV1().Deployments(namespace).Create(ctx, deployment, metav1.CreateOptions{}); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return "", fmt.Errorf("create Kubernetes deployment: %w", err)
		}
		existing, getErr := k.client.AppsV1().Deployments(namespace).Get(ctx, spec.Name, metav1.GetOptions{})
		if getErr != nil {
			return "", fmt.Errorf("get existing Kubernetes deployment: %w", getErr)
		}
		deployment.ResourceVersion = existing.ResourceVersion
		if _, err := k.client.AppsV1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{}); err != nil {
			return "", fmt.Errorf("update Kubernetes deployment: %w", err)
		}
	}
	if len(servicePorts) > 0 {
		service := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{Name: spec.Name, Labels: labels},
			Spec: corev1.ServiceSpec{
				Type:     corev1.ServiceTypeClusterIP,
				Selector: map[string]string{"atonixcorp.io/deployment-id": deploymentID},
				Ports:    servicePorts,
			},
		}
		if _, err := k.client.CoreV1().Services(namespace).Create(ctx, service, metav1.CreateOptions{}); err != nil {
			if !apierrors.IsAlreadyExists(err) {
				return "", fmt.Errorf("create Kubernetes service: %w", err)
			}
			existing, getErr := k.client.CoreV1().Services(namespace).Get(ctx, service.Name, metav1.GetOptions{})
			if getErr != nil {
				return "", fmt.Errorf("get existing Kubernetes service: %w", getErr)
			}
			service.ResourceVersion = existing.ResourceVersion
			service.Spec.ClusterIP = existing.Spec.ClusterIP
			if _, err := k.client.CoreV1().Services(namespace).Update(ctx, service, metav1.UpdateOptions{}); err != nil {
				return "", fmt.Errorf("update Kubernetes service: %w", err)
			}
		}
		return fmt.Sprintf("%s.%s.svc.cluster.local:%d", spec.Name, namespace, servicePorts[0].Port), nil
	}
	return "", nil
}

func (k *KubernetesProvisioner) WaitReady(ctx context.Context, namespace, name string, desired int32) error {
	return k.WaitReplicas(ctx, namespace, name, desired)
}

func (k *KubernetesProvisioner) WaitReplicas(ctx context.Context, namespace, name string, desired int32) error {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	for {
		deployment, err := k.client.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		if desired == 0 && deployment.Status.Replicas == 0 {
			return nil
		}
		if desired > 0 && deployment.Status.AvailableReplicas >= desired {
			return nil
		}
		if deployment.Status.ObservedGeneration >= deployment.Generation {
			for _, condition := range deployment.Status.Conditions {
				if condition.Type == appsv1.DeploymentProgressing &&
					condition.Status == corev1.ConditionFalse &&
					condition.Reason == "ProgressDeadlineExceeded" {
					return fmt.Errorf("deployment did not become ready: %s", condition.Message)
				}
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (k *KubernetesProvisioner) ensureNamespace(ctx context.Context, namespace string, tenantID, projectID uint) error {
	labels := map[string]string{
		"app.kubernetes.io/managed-by":       "atonixcorp",
		"atonixcorp.io/tenant-id":            fmt.Sprint(tenantID),
		"atonixcorp.io/project-id":           fmt.Sprint(projectID),
		"pod-security.kubernetes.io/enforce": "restricted",
		"pod-security.kubernetes.io/audit":   "restricted",
		"pod-security.kubernetes.io/warn":    "restricted",
	}
	existingNamespace, err := k.client.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		if _, createErr := k.client.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: namespace, Labels: labels},
		}, metav1.CreateOptions{}); createErr != nil && !apierrors.IsAlreadyExists(createErr) {
			return fmt.Errorf("create tenant namespace: %w", createErr)
		}
		existingNamespace, err = k.client.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("verify tenant namespace: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("check tenant namespace: %w", err)
	} else if existingNamespace.Labels["app.kubernetes.io/managed-by"] != "atonixcorp" ||
		existingNamespace.Labels["atonixcorp.io/tenant-id"] != fmt.Sprint(tenantID) ||
		existingNamespace.Labels["atonixcorp.io/project-id"] != fmt.Sprint(projectID) {
		return errors.New("Kubernetes namespace is not owned by this tenant and project")
	}

	quota := &corev1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{Name: "atonixcorp-default-quota", Namespace: namespace},
		Spec: corev1.ResourceQuotaSpec{Hard: corev1.ResourceList{
			corev1.ResourcePods:            resource.MustParse("100"),
			corev1.ResourceRequestsCPU:     resource.MustParse("100"),
			corev1.ResourceRequestsMemory:  resource.MustParse("256Gi"),
			corev1.ResourceRequestsStorage: resource.MustParse("2Ti"),
			corev1.ResourceLimitsCPU:       resource.MustParse("200"),
			corev1.ResourceLimitsMemory:    resource.MustParse("512Gi"),
		}},
	}
	if _, err := k.client.CoreV1().ResourceQuotas(namespace).Create(ctx, quota, metav1.CreateOptions{}); err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("apply namespace resource quota: %w", err)
	}
	policy := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "atonixcorp-default-ingress", Namespace: namespace},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
			Ingress: []networkingv1.NetworkPolicyIngressRule{{
				From: []networkingv1.NetworkPolicyPeer{{PodSelector: &metav1.LabelSelector{}}},
			}},
		},
	}
	if _, err := k.client.NetworkingV1().NetworkPolicies(namespace).Create(ctx, policy, metav1.CreateOptions{}); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("apply namespace network policy: %w", err)
		}
	}
	limitRange := &corev1.LimitRange{
		ObjectMeta: metav1.ObjectMeta{Name: "atonixcorp-default-limits", Namespace: namespace},
		Spec: corev1.LimitRangeSpec{Limits: []corev1.LimitRangeItem{{
			Type: corev1.LimitTypeContainer,
			Default: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("1"),
				corev1.ResourceMemory: resource.MustParse("1Gi"),
			},
			DefaultRequest: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("100m"),
				corev1.ResourceMemory: resource.MustParse("128Mi"),
			},
		}}},
	}
	if _, err := k.client.CoreV1().LimitRanges(namespace).Create(ctx, limitRange, metav1.CreateOptions{}); err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("apply namespace limits: %w", err)
	}
	return nil
}

func (k *KubernetesProvisioner) Scale(ctx context.Context, namespace, name string, replicas int32) error {
	_, err := k.client.AppsV1().Deployments(namespace).Patch(
		ctx, name, types.MergePatchType, []byte(fmt.Sprintf(`{"spec":{"replicas":%d}}`, replicas)), metav1.PatchOptions{},
	)
	return err
}

func (k *KubernetesProvisioner) Restart(ctx context.Context, namespace, name string) error {
	_, err := k.client.AppsV1().Deployments(namespace).Patch(
		ctx, name, types.MergePatchType,
		[]byte(fmt.Sprintf(`{"spec":{"template":{"metadata":{"annotations":{"atonixcorp.io/restartedAt":%q}}}}}`, time.Now().UTC().Format(time.RFC3339Nano))),
		metav1.PatchOptions{},
	)
	return err
}

func (k *KubernetesProvisioner) Delete(ctx context.Context, namespace, name, deploymentID string, retainVolumes bool) error {
	if _, err := k.client.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{}); apierrors.IsNotFound(err) {
		return nil
	} else if err != nil {
		return err
	}
	propagation := metav1.DeletePropagationForeground
	if err := k.client.AppsV1().Deployments(namespace).Delete(ctx, name, metav1.DeleteOptions{PropagationPolicy: &propagation}); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	if err := k.client.CoreV1().Services(namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	if err := k.client.CoreV1().Secrets(namespace).Delete(ctx, deploymentID+"-env", metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	if !retainVolumes {
		claims, err := k.client.CoreV1().PersistentVolumeClaims(namespace).List(ctx, metav1.ListOptions{
			LabelSelector: "atonixcorp.io/deployment-id=" + deploymentID,
		})
		if err != nil {
			return err
		}
		for _, claim := range claims.Items {
			if err := k.client.CoreV1().PersistentVolumeClaims(namespace).Delete(ctx, claim.Name, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
				return err
			}
		}
	}
	return nil
}

func (k *KubernetesProvisioner) Logs(ctx context.Context, namespace, deploymentID string) (string, error) {
	pods, err := k.Instances(ctx, namespace, deploymentID)
	if err != nil {
		return "", err
	}
	if len(pods) == 0 {
		return "", nil
	}
	tail := int64(200)
	var output strings.Builder
	for _, pod := range pods {
		if len(output.String()) >= 1024*1024 {
			break
		}
		request := k.client.CoreV1().Pods(namespace).GetLogs(pod.Name, &corev1.PodLogOptions{
			Container: "application", TailLines: &tail, LimitBytes: int64Pointer(262144),
		})
		stream, err := request.Stream(ctx)
		if err != nil {
			return "", fmt.Errorf("read pod logs: %w", err)
		}
		_, copyErr := io.Copy(&output, io.LimitReader(stream, 262144))
		closeErr := stream.Close()
		if copyErr != nil {
			return "", copyErr
		}
		if closeErr != nil {
			return "", closeErr
		}
	}
	return output.String(), nil
}

func (k *KubernetesProvisioner) Events(ctx context.Context, namespace, deploymentID string) ([]corev1.Event, error) {
	result, err := k.client.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{Limit: 200})
	if err != nil {
		return nil, err
	}
	filtered := make([]corev1.Event, 0, len(result.Items))
	for _, event := range result.Items {
		if event.InvolvedObject.Name == deploymentID || strings.HasPrefix(event.InvolvedObject.Name, deploymentID+"-") {
			filtered = append(filtered, event)
		}
	}
	return filtered, nil
}

func (k *KubernetesProvisioner) Instances(ctx context.Context, namespace, deploymentID string) ([]corev1.Pod, error) {
	pods, err := k.client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "atonixcorp.io/deployment-id=" + deploymentID,
		Limit:         100,
	})
	if err != nil {
		return nil, err
	}
	return pods.Items, nil
}

func (k *KubernetesProvisioner) Status(ctx context.Context, namespace, name string) (appsv1.DeploymentStatus, error) {
	deployment, err := k.client.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return appsv1.DeploymentStatus{}, err
	}
	return deployment.Status, nil
}

func (k *KubernetesProvisioner) Metrics(ctx context.Context, namespace, deploymentID string) (DeploymentMetrics, error) {
	if _, err := k.client.Discovery().ServerResourcesForGroupVersion("metrics.k8s.io/v1beta1"); err != nil {
		return DeploymentMetrics{}, fmt.Errorf("%w: %v", ErrMetricsUnavailable, err)
	}
	pods, err := k.Instances(ctx, namespace, deploymentID)
	if err != nil {
		return DeploymentMetrics{}, err
	}
	result := DeploymentMetrics{SampledAt: time.Now().UTC().Format(time.RFC3339)}
	for _, pod := range pods {
		var payload struct {
			Containers []struct {
				Name  string            `json:"name"`
				Usage map[string]string `json:"usage"`
			} `json:"containers"`
		}
		raw, err := k.client.Discovery().RESTClient().Get().
			AbsPath("/apis/metrics.k8s.io/v1beta1/namespaces/" + namespace + "/pods/" + pod.Name).
			Do(ctx).Raw()
		if apierrors.IsNotFound(err) {
			continue
		}
		if err != nil {
			return DeploymentMetrics{}, fmt.Errorf("read Kubernetes pod metrics: %w", err)
		}
		if err := json.Unmarshal(raw, &payload); err != nil {
			return DeploymentMetrics{}, fmt.Errorf("decode Kubernetes pod metrics: %w", err)
		}
		for _, container := range payload.Containers {
			if value := container.Usage["cpu"]; value != "" {
				quantity, parseErr := resource.ParseQuantity(value)
				if parseErr != nil {
					return DeploymentMetrics{}, fmt.Errorf("parse Kubernetes CPU metric: %w", parseErr)
				}
				result.CPUUsageMillis += quantity.MilliValue()
			}
			if value := container.Usage["memory"]; value != "" {
				quantity, parseErr := resource.ParseQuantity(value)
				if parseErr != nil {
					return DeploymentMetrics{}, fmt.Errorf("parse Kubernetes memory metric: %w", parseErr)
				}
				result.MemoryUsageBytes += quantity.Value()
			}
		}
	}
	return result, nil
}

func buildProbe(probe Probe, port func(int32) intstr.IntOrString) *corev1.Probe {
	httpGet := &corev1.HTTPGetAction{Path: probe.Path, Port: port(probe.Port)}
	return &corev1.Probe{
		ProbeHandler:        corev1.ProbeHandler{HTTPGet: httpGet},
		InitialDelaySeconds: probe.InitialDelaySeconds,
		PeriodSeconds:       probe.PeriodSeconds,
		TimeoutSeconds:      probe.TimeoutSeconds,
		FailureThreshold:    probe.FailureThreshold,
	}
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func boolPointer(value bool) *bool    { return &value }
func int64Pointer(value int64) *int64 { return &value }
