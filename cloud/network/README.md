┌───────────────────────────────┐
│ Global Network Backbone       │
└───────────────────────────────┘
          │
 ┌────────┴────────┐
 │                 │
Firewall       Firewall

          │
 ┌───────────────────────────────┐
 │   ATNetwork Controller        │
 │                               │
 │ Routing & VLAN               │
 │ Security Groups              │
 │ DNS/DHCP                     │
 │ OpenStack Neutron            │
 │ Floating IP Management       │
 └───────────────────────────────┘

          │
══════════════════════════════
 VXLAN Encrypted Backbone
══════════════════════════════

 FRR/BGP      FRR/BGP
 Gateway      Gateway

 ┌────────┐   ┌────────┐
 │AF South│   │US East │
 └────────┘   └────────┘

 ┌────────┐   ┌────────┐
 │EU Cen. │   │AP SE   │
 └────────┘   └────────┘

          │

 ┌───────────────────────────────┐
 │ Hypervisor Layer              │
 │ OVN Networking                │
 │ FRR Routing                   │
 │ DNSMASQ Services              │
 └───────────────────────────────┘

## Current implementation status

The networking module is an early foundation, not a production-ready control
plane. Network and subnet creation now validate canonical IPv4/IPv6 CIDRs,
require a subnet to fit inside its parent network, validate gateway addresses,
and reject overlapping allocations transactionally in PostgreSQL. Resource IDs
are UUIDs compatible with the schema, and create requests use bounded,
strict JSON decoding with stable validation/conflict errors.

The networking API now validates HS256 bearer tokens against the platform JWT
secret, requires tenant and project memberships for scoped network routes, and
stores project ownership on network records. Schema migrations run on service
startup. The dashboard has a workspace/project-aware network inventory and
creation surface, served through an authenticated same-origin proxy.

The Compose profile starts with network backends disabled. To provision
networks and subnets through Neutron, set `OPENSTACK_ENABLE_NEUTRON=true`,
configure `OPENSTACK_AUTH_URL` and `OPENSTACK_ACCOUNT_ENCRYPTION_KEY`, and map
the organization to a project-scoped application credential. Each operation
loads and decrypts that tenant's mapped account, verifies the token's project,
discovers the Neutron endpoint from Keystone, tags provider resources with
AtonixCorp IDs, and verifies provider state before returning it. Missing or
invalid mappings fail closed; shared-project credentials are not used as a
fallback. Insecure HTTP requires the explicit development-only
`OPENSTACK_ALLOW_INSECURE_HTTP=true` opt-in.

The Neutron integration currently supports mapped-project network and subnet
create/read flows; routers, ports, floating IPs, security-group changes, and
provider-side imports remain unavailable. Durable operations/outbox,
reconciliation, quotas, audit/monitoring, and the remaining resources and UI
flows are still outstanding, so this is not production-ready or a completed
implementation of the full specification. AtonixCorp workspace/project access
is enforced by the API, but all such resources share the configured OpenStack
project's provider-level quota and security boundary.