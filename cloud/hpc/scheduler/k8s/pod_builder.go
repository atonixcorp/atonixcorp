package k8s

import (
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type JobTemplate struct {
	Name       string
	Image      string
	Command    []string
	Args       []string
	CPUs       string // e.g. "4"
	Memory     string // e.g. "8Gi"
	GPUs       string // e.g. "1"
	Env        map[string]string
	WorkingDir string
	Labels     map[string]string
}

func BuildJob(template JobTemplate) *batchv1.Job {
	envVars := []corev1.EnvVar{}
	for k, v := range template.Env {
		envVars = append(envVars, corev1.EnvVar{
			Name:  k,
			Value: v,
		})
	}

	labels := map[string]string{
		"app":      "atonix-hpc-job",
		"job-name": template.Name,
	}
	for k, v := range template.Labels {
		labels[k] = v
	}

	resources := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{},
		Limits:   corev1.ResourceList{},
	}

	if template.CPUs != "" {
		cpuQty := resourceMustParse(template.CPUs)
		resources.Requests[corev1.ResourceCPU] = cpuQty
		resources.Limits[corev1.ResourceCPU] = cpuQty
	}
	if template.Memory != "" {
		memQty := resourceMustParse(template.Memory)
		resources.Requests[corev1.ResourceMemory] = memQty
		resources.Limits[corev1.ResourceMemory] = memQty
	}
	if template.GPUs != "" {
		gpuQty := resourceMustParse(template.GPUs)
		resources.Limits["nvidia.com/gpu"] = gpuQty
	}

	container := corev1.Container{
		Name:            "job-container",
		Image:           template.Image,
		Command:         template.Command,
		Args:            template.Args,
		Env:             envVars,
		WorkingDir:      template.WorkingDir,
		Resources:       resources,
		ImagePullPolicy: corev1.PullIfNotPresent,
	}

	podTemplate := corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Name:   template.Name,
			Labels: labels,
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			Containers:    []corev1.Container{container},
		},
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:   template.Name,
			Labels: labels,
		},
		Spec: batchv1.JobSpec{
			Template: podTemplate,
		},
	}

	return job
}

// resourceMustParse is a small helper; you can move it to a utils package if you prefer.
func resourceMustParse(v string) resource.Quantity {
	return resource.MustParse(v)
}
