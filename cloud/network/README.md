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