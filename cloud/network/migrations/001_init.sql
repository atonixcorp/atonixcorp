-- ============================================================
-- ATN NETWORKING DATABASE SCHEMA
-- ============================================================

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================================
-- NETWORKS
-- ============================================================

CREATE TABLE networks (
    id              UUID PRIMARY KEY,
    name            TEXT NOT NULL,
    cidr            CIDR NOT NULL,
    tenant_id       TEXT NOT NULL,
    external        BOOLEAN DEFAULT FALSE,
    status          TEXT DEFAULT 'ACTIVE',
    created_at      TIMESTAMP DEFAULT NOW(),
    updated_at      TIMESTAMP DEFAULT NOW()
);

-- ============================================================
-- SUBNETS
-- ============================================================

CREATE TABLE subnets (
    id              UUID PRIMARY KEY,
    network_id      UUID REFERENCES networks(id) ON DELETE CASCADE,
    cidr            CIDR NOT NULL,
    gateway_ip      INET,
    dhcp_enabled    BOOLEAN DEFAULT TRUE,
    created_at      TIMESTAMP DEFAULT NOW(),
    updated_at      TIMESTAMP DEFAULT NOW()
);

-- ============================================================
-- ROUTERS
-- ============================================================

CREATE TABLE routers (
    id                  UUID PRIMARY KEY,
    name                TEXT NOT NULL,
    tenant_id           TEXT NOT NULL,
    external_network_id UUID REFERENCES networks(id),
    status              TEXT DEFAULT 'ACTIVE',
    created_at          TIMESTAMP DEFAULT NOW(),
    updated_at          TIMESTAMP DEFAULT NOW()
);

-- ============================================================
-- ROUTER INTERFACES
-- ============================================================

CREATE TABLE router_interfaces (
    id          UUID PRIMARY KEY,
    router_id   UUID REFERENCES routers(id) ON DELETE CASCADE,
    subnet_id   UUID REFERENCES subnets(id) ON DELETE CASCADE,
    ip_address  INET NOT NULL,
    created_at  TIMESTAMP DEFAULT NOW()
);

-- ============================================================
-- PORTS
-- ============================================================

CREATE TABLE ports (
    id          UUID PRIMARY KEY,
    network_id  UUID REFERENCES networks(id) ON DELETE CASCADE,
    device_id   TEXT NOT NULL,
    mac_address TEXT NOT NULL,
    status      TEXT DEFAULT 'DOWN',
    created_at  TIMESTAMP DEFAULT NOW(),
    updated_at  TIMESTAMP DEFAULT NOW()
);

-- ============================================================
-- PORT IPs
-- ============================================================

CREATE TABLE port_ips (
    id          UUID PRIMARY KEY,
    port_id     UUID REFERENCES ports(id) ON DELETE CASCADE,
    ip_address  INET NOT NULL
);

-- ============================================================
-- SECURITY GROUPS
-- ============================================================

CREATE TABLE security_groups (
    id          UUID PRIMARY KEY,
    name        TEXT NOT NULL,
    tenant_id   TEXT NOT NULL,
    created_at  TIMESTAMP DEFAULT NOW(),
    updated_at  TIMESTAMP DEFAULT NOW()
);

-- ============================================================
-- SECURITY GROUP RULES
-- ============================================================

CREATE TABLE security_group_rules (
    id            UUID PRIMARY KEY,
    sg_id         UUID REFERENCES security_groups(id) ON DELETE CASCADE,
    direction     TEXT NOT NULL,      -- ingress | egress
    protocol      TEXT NOT NULL,      -- tcp | udp | icmp | any
    port_range    TEXT,
    cidr          CIDR NOT NULL,
    created_at    TIMESTAMP DEFAULT NOW()
);

-- ============================================================
-- PORT ↔ SECURITY GROUP MAPPING
-- ============================================================

CREATE TABLE port_security_groups (
    port_id UUID REFERENCES ports(id) ON DELETE CASCADE,
    sg_id   UUID REFERENCES security_groups(id) ON DELETE CASCADE,
    PRIMARY KEY (port_id, sg_id)
);

-- ============================================================
-- FLOATING IPs
-- ============================================================

CREATE TABLE floating_ips (
    id              UUID PRIMARY KEY,
    tenant_id       TEXT NOT NULL,
    external_ip     INET NOT NULL,
    internal_port_id UUID REFERENCES ports(id) ON DELETE CASCADE,
    internal_ip     INET NOT NULL,
    created_at      TIMESTAMP DEFAULT NOW()
);

-- ============================================================
-- NAT RULES
-- ============================================================

CREATE TABLE nat_rules (
    id          UUID PRIMARY KEY,
    router_id   UUID REFERENCES routers(id) ON DELETE CASCADE,
    external_ip INET NOT NULL,
    internal_ip INET NOT NULL,
    type        TEXT NOT NULL,        -- snat | dnat
    created_at  TIMESTAMP DEFAULT NOW()
);

-- ============================================================
-- BACKEND STATE (OVN, FRR, Neutron, Fabric, CNI)
-- ============================================================

CREATE TABLE backend_state (
    id            UUID PRIMARY KEY,
    backend_name  TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id   UUID NOT NULL,
    status        TEXT NOT NULL,      -- SYNCED | ERROR | PENDING
    last_error    TEXT,
    updated_at    TIMESTAMP DEFAULT NOW()
);

-- ============================================================
-- EVENTS (AUDIT + RECONCILER)
-- ============================================================

CREATE TABLE events (
    id            UUID PRIMARY KEY,
    resource_id   UUID,
    resource_type TEXT,
    action        TEXT,               -- create | update | delete | sync | error
    message       TEXT,
    created_at    TIMESTAMP DEFAULT NOW()
);
