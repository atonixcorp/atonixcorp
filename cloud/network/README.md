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

The Compose profile intentionally starts with all network backends disabled.
Creation is unavailable until a backend can apply and independently verify
network and subnet state; the service will fail closed rather than report a
metadata-only row as a working network. Existing OVN, FRR, and other adapters
do not yet satisfy that verification contract. Durable operations/outbox,
reconciliation, quotas, audit/monitoring, and the remaining resources and UI
flows are still outstanding, so this is not production-ready or a completed
implementation of the full specification.