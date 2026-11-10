import type { ComputeInstance, ComputePlan } from "../compute/lib/types";
import type { Metric } from "../observability/lib/api";

export function allocatedCompute(instances: ComputeInstance[], plans: ComputePlan[] | undefined) {
  if (!instances.length) return { vcpus: 0, memoryGB: 0 };
  if (!plans) return null;
  const sizes = new Map(plans.map((plan) => [plan.id, plan]));
  let vcpus = 0;
  let memoryMB = 0;
  for (const instance of instances) {
    const plan = sizes.get(instance.plan_id);
    if (!plan || !Number.isFinite(plan.vcpus) || !Number.isFinite(plan.memory_mb)) return null;
    vcpus += plan.vcpus;
    memoryMB += plan.memory_mb;
  }
  return { vcpus, memoryGB: memoryMB / 1024 };
}

export function utilizationSample(metrics: Metric[], name: string) {
  const samples = metrics.filter((metric) =>
    metric.name === name && metric.unit === "%" && Number.isFinite(metric.value) && metric.value >= 0);
  return {
    value: samples.length ? samples.reduce((sum, metric) => sum + metric.value, 0) / samples.length : null,
    count: samples.length,
    oldest: samples.map((metric) => metric.observed_at).sort()[0],
  };
}

export function remainingQuota(used: number, quota: number): number | null {
  if (!Number.isFinite(used) || !Number.isFinite(quota) || used < 0 || quota <= 0) return null;
  return Math.max(0, quota - used);
}

type RecentResource = { id: string; name: string; kind: string; status: string; created_at: string; href: string };

export function recentResources(items: RecentResource[]): RecentResource[] {
  return items.filter((item) => Number.isFinite(Date.parse(item.created_at)))
    .sort((a, b) => Date.parse(b.created_at) - Date.parse(a.created_at)).slice(0, 8);
}
