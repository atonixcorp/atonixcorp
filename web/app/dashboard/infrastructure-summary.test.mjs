import assert from "node:assert/strict";
import { test } from "node:test";
import { allocatedCompute, utilizationSample, recentResources, remainingQuota } from "./infrastructure-summary.ts";
import { initialScopeID, resolveScopeID, scopedCloudHref } from "../shared/lib/cloud-scope.ts";
import { cloudNavigation, isCloudNavigationItemActive } from "../shared/ui/cloud-navigation.ts";
import { getFirewallGroups } from "../networking/api.ts";

test("compute allocation joins actual instance sizes and refuses incomplete totals", () => {
  const plans = [{ id: "small", vcpus: 2, memory_mb: 2048 }, { id: "large", vcpus: 8, memory_mb: 16384 }];
  const instances = [{ plan_id: "small" }, { plan_id: "large" }, { plan_id: "small" }];
  assert.deepEqual(allocatedCompute(instances, plans), { vcpus: 12, memoryGB: 20 });
  assert.equal(allocatedCompute(instances, plans.slice(0, 1)), null);
  assert.equal(allocatedCompute(instances, undefined), null);
  assert.deepEqual(allocatedCompute([], undefined), { vcpus: 0, memoryGB: 0 });
});

test("utilization uses only valid percent samples and retains missing versus zero", () => {
  const name = "atonix_resource_cpu_utilization_ratio";
  const samples = [
    { name, value: 0, unit: "%", observed_at: "2026-01-01T10:00:00Z" },
    { name, value: 50, unit: "%", observed_at: "2026-01-01T10:01:00Z" },
    { name, value: 0.9, unit: "ratio", observed_at: "2026-01-01T10:01:00Z" },
    { name, value: NaN, unit: "%", observed_at: "2026-01-01T10:01:00Z" },
    { name: "other", value: 100, unit: "%", observed_at: "2026-01-01T10:01:00Z" },
  ];
  assert.deepEqual(utilizationSample(samples, name), { value: 25, count: 2, oldest: "2026-01-01T10:00:00Z" });
  assert.equal(utilizationSample(samples.slice(0, 1), name).value, 0);
  assert.equal(utilizationSample([], name).value, null);
});

test("creation deep links retain authorized scope and reject unavailable projects", () => {
  const previousWindow = globalThis.window;
  globalThis.window = { location: { search: "?tenant_id=2&project_id=22" } };
  try {
    assert.equal(initialScopeID([{ id: 1 }, { id: 2 }], "tenant_id"), 2);
    assert.equal(initialScopeID([{ id: 21 }, { id: 22 }], "project_id", 2), 22);
    assert.throws(() => initialScopeID([{ id: 21 }], "project_id", 2), /not accessible/);
    assert.equal(initialScopeID([{ id: 31 }], "project_id", 3), 31);
    globalThis.window.location.search = "";
    assert.equal(initialScopeID([{ id: 1 }], "tenant_id"), 1);
    assert.equal(initialScopeID([], "tenant_id"), 0);
  } finally {
    if (previousWindow === undefined) delete globalThis.window;
    else globalThis.window = previousWindow;
  }
});

test("sidebar selects one service for query tabs and nested resource workflows", () => {
  const query = new URLSearchParams("tab=objects&tenant_id=1&project_id=2");
  assert.equal(isCloudNavigationItemActive("/storage?tab=objects", "/storage", query), true);
  assert.equal(isCloudNavigationItemActive("/storage", "/storage", query), false);
  const events = new URLSearchParams("section=Events&project_id=2");
  assert.equal(isCloudNavigationItemActive("/observability?section=Events", "/observability", events), true);
  assert.equal(isCloudNavigationItemActive("/observability", "/observability", events), false);
  const none = new URLSearchParams();
  assert.equal(isCloudNavigationItemActive("/compute/instances", "/compute/deploy", none), true);
  assert.equal(isCloudNavigationItemActive("/compute/instances", "/compute/instances/server-1", none), true);
  assert.equal(isCloudNavigationItemActive("/containers", "/containers/workload-1", none), true);
  assert.equal(isCloudNavigationItemActive("/containers", "/containers/registries", none), false);
  assert.equal(isCloudNavigationItemActive("/containers/registries", "/containers/registries", none), true);
  assert.equal(isCloudNavigationItemActive("/cloud/databases", "/databases", none), true);
});

test("remaining quota does not invent capacity and preserves exhausted versus unknown", () => {
  assert.equal(remainingQuota(30, 100), 70);
  assert.equal(remainingQuota(100, 100), 0);
  assert.equal(remainingQuota(120, 100), 0);
  for (const [used, quota] of [[0, 0], [1, -1], [-1, 100], [NaN, 100], [1, Infinity]]) {
    assert.equal(remainingQuota(used, quota), null);
  }
});

test("recent resources merge kinds by actual creation time without mutating inventories", () => {
  const items = Array.from({ length: 10 }, (_, index) => ({
    id: String(index), name: `resource-${index}`, kind: index % 2 ? "Volume" : "Instance",
    status: "available", href: "/storage", created_at: `2026-01-${String(index + 1).padStart(2, "0")}T00:00:00Z`,
  }));
  const original = structuredClone(items);
  const result = recentResources([...items, { ...items[0], id: "invalid", created_at: "unknown" }]);
  assert.deepEqual(result.map((item) => item.id), ["9", "8", "7", "6", "5", "4", "3", "2"]);
  assert.deepEqual(items, original);
  assert.equal(recentResources([]).length, 0);
});

test("service links preserve scope, product filters and fragment targets", () => {
  assert.equal(scopedCloudHref("/storage?tab=objects#upload", 2, 22), "/storage?tab=objects&tenant_id=2&project_id=22#upload");
  assert.equal(scopedCloudHref("/networking#create-network", 2, 22), "/networking?tenant_id=2&project_id=22#create-network");
  assert.equal(scopedCloudHref("/cloud/kubernetes?tenant_id=1&project_id=11", 2, 0), "/cloud/kubernetes?tenant_id=2");
  assert.equal(scopedCloudHref("/storage", 0, 0), "/storage");
  assert.equal(resolveScopeID([{ id: 2 }], "tenant_id", new URLSearchParams("tenant_id=2")), 2);
  assert.throws(() => resolveScopeID([{ id: 2 }], "tenant_id", new URLSearchParams("tenant_id=99")), /not accessible/);
});

test("new service routes have exactly one active sidebar destination", () => {
  for (const href of ["/cloud/kubernetes", "/cloud/clusters", "/cloud/ssh-keys", "/cloud/snapshots", "/networking/firewalls", "/containers/registries"]) {
    const active = cloudNavigation.flatMap((group) => group.items)
      .filter((item) => isCloudNavigationItemActive(item.href, href, new URLSearchParams("tenant_id=2&project_id=22")));
    assert.equal(active.length, 1, href);
    assert.equal(active[0].href, href);
  }
});

test("firewall inventory uses the project API and preserves permission and dependency failures", async () => {
  const originalFetch = globalThis.fetch;
  const rules = [{ direction: "ingress", ethertype: "IPv4", protocol: "tcp", port_range_min: 443, port_range_max: 443 }];
  try {
    globalThis.fetch = async (path) => {
      assert.equal(path, "/api/compute/firewalls?tenant_id=2&project_id=22");
      return Response.json([{ name: "application", description: "", rules }]);
    };
    assert.deepEqual(await getFirewallGroups(2, 22), [{ name: "application", description: "", rules }]);
    for (const status of [401, 403, 503]) {
      globalThis.fetch = async () => Response.json({ error: `Failure ${status}` }, { status });
      await assert.rejects(getFirewallGroups(2, 22), (error) => error.status === status && error.message === `Failure ${status}`);
    }
    globalThis.fetch = async () => Response.json([]);
    assert.deepEqual(await getFirewallGroups(2, 22), []);
  } finally {
    globalThis.fetch = originalFetch;
  }
});
