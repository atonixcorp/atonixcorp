package containers

import (
	"context"
	"errors"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestKubernetesProvisionerCreatesRestrictedNamespacedWorkload(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleClientset()
	provisioner := &KubernetesProvisioner{client: client}
	spec := validSpec()
	if err := ValidateSpec(&spec); err != nil {
		t.Fatal(err)
	}
	var stages []string
	stage := func(value string) error {
		stages = append(stages, value)
		return nil
	}
	if _, err := provisioner.Create(ctx, "atonix-t3-p5", 3, 5, "deploymentid", spec, map[string]string{
		"API_TOKEN": "secret-value",
	}, stage); err != nil {
		t.Fatal(err)
	}

	namespace, err := client.CoreV1().Namespaces().Get(ctx, "atonix-t3-p5", metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if namespace.Labels["pod-security.kubernetes.io/enforce"] != "restricted" {
		t.Fatalf("restricted pod security label was not set: %#v", namespace.Labels)
	}
	policy, err := client.NetworkingV1().NetworkPolicies("atonix-t3-p5").Get(ctx, "atonixcorp-default-ingress", metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(policy.Spec.Ingress) != 1 ||
		policy.Spec.Ingress[0].From[0].NamespaceSelector != nil ||
		policy.Spec.Ingress[0].From[0].PodSelector == nil {
		t.Fatalf("network policy is not confined to the same namespace: %+v", policy.Spec)
	}

	deployment, err := client.AppsV1().Deployments("atonix-t3-p5").Get(ctx, "web", metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	created := deployment.Spec.Template.Spec.Containers[0]
	if created.SecurityContext == nil || created.SecurityContext.Privileged != nil ||
		created.SecurityContext.AllowPrivilegeEscalation == nil ||
		*created.SecurityContext.AllowPrivilegeEscalation ||
		created.SecurityContext.RunAsNonRoot == nil || !*created.SecurityContext.RunAsNonRoot {
		t.Fatalf("container security defaults are not restrictive: %+v", created.SecurityContext)
	}
	if deployment.Spec.Template.Spec.AutomountServiceAccountToken == nil ||
		*deployment.Spec.Template.Spec.AutomountServiceAccountToken {
		t.Fatal("service account token automount should be disabled")
	}
	if created.Env[1].Value != "" || created.Env[1].ValueFrom == nil ||
		created.Env[1].ValueFrom.SecretKeyRef == nil {
		t.Fatal("secret environment value must be referenced from a Kubernetes Secret")
	}
	secret, err := client.CoreV1().Secrets("atonix-t3-p5").Get(ctx, "deploymentid-env", metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if secret.StringData["API_TOKEN"] != "secret-value" {
		t.Fatal("Kubernetes secret was not materialized")
	}
	if _, err := client.CoreV1().PersistentVolumeClaims("atonix-t3-p5").Get(ctx, "deploymentid-data", metav1.GetOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := client.CoreV1().Services("atonix-t3-p5").Get(ctx, "web", metav1.GetOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := client.CoreV1().ResourceQuotas("atonix-t3-p5").Get(ctx, "atonixcorp-default-quota", metav1.GetOptions{}); err != nil {
		t.Fatal(err)
	}
	if len(stages) == 0 || stages[0] != "CREATING_NETWORK" {
		t.Fatalf("provisioning stage callback was not used: %v", stages)
	}

	if deployment.Status.AvailableReplicas != 0 {
		t.Fatal("fake Deployment should not be considered running before readiness")
	}
}

func TestMetricsRequiresKubernetesMetricsAPI(t *testing.T) {
	provisioner := &KubernetesProvisioner{client: fake.NewSimpleClientset()}
	_, err := provisioner.Metrics(context.Background(), "project", "deploymentid")
	if !errors.Is(err, ErrMetricsUnavailable) {
		t.Fatalf("expected metrics-unavailable error, got %v", err)
	}
}

func TestWaitReplicasOnlyReturnsAfterAvailability(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleClientset(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "project"},
		Spec:       appsv1.DeploymentSpec{Replicas: int32Pointer(1)},
		Status:     appsv1.DeploymentStatus{Replicas: 1, AvailableReplicas: 1},
	})
	provisioner := &KubernetesProvisioner{client: client}
	if err := provisioner.WaitReplicas(ctx, "project", "web", 1); err != nil {
		t.Fatal(err)
	}
}

func int32Pointer(value int32) *int32 { return &value }
