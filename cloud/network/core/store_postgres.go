package core

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	_ "github.com/lib/pq"
)

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore(conn string) (*PostgresStore, error) {
	db, err := sql.Open("postgres", conn)
	if err != nil {
		return nil, err
	}
	return &PostgresStore{db: db}, nil
}

func (s *PostgresStore) Close() error {
	return s.db.Close()
}

func (s *PostgresStore) Migrate(directory string) error {
	if _, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS networking_schema_migrations (
			name TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`); err != nil {
		return fmt.Errorf("create migration ledger: %w", err)
	}
	files, err := filepath.Glob(filepath.Join(directory, "*.sql"))
	if err != nil {
		return fmt.Errorf("find networking migrations: %w", err)
	}
	sort.Strings(files)
	for _, path := range files {
		name := filepath.Base(path)
		var applied bool
		if err := s.db.QueryRow(
			`SELECT EXISTS (SELECT 1 FROM networking_schema_migrations WHERE name = $1)`,
			name,
		).Scan(&applied); err != nil {
			return fmt.Errorf("check migration %s: %w", name, err)
		}
		if applied {
			continue
		}
		sqlText, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}
		tx, err := s.db.Begin()
		if err != nil {
			return fmt.Errorf("begin migration %s: %w", name, err)
		}
		if _, err := tx.Exec(string(sqlText)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply migration %s: %w", name, err)
		}
		if _, err := tx.Exec(
			`INSERT INTO networking_schema_migrations (name) VALUES ($1)`,
			name,
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record migration %s: %w", name, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", name, err)
		}
	}
	return nil
}

//
// NETWORKS
//

func (s *PostgresStore) CreateNetwork(n Network) error {
	if err := NewPolicy().ValidateNetwork(n); err != nil {
		return fmt.Errorf("validate network: %w", err)
	}
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin network allocation: %w", err)
	}
	defer tx.Rollback()
	var lockResult any
	if err := tx.QueryRow(`SELECT pg_advisory_xact_lock(hashtext($1))`, n.TenantID).Scan(&lockResult); err != nil {
		return fmt.Errorf("lock tenant address space: %w", err)
	}
	var overlaps bool
	if err := tx.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM networks
			WHERE tenant_id = $1 AND cidr && $2::cidr
		)
	`, n.TenantID, n.CIDR).Scan(&overlaps); err != nil {
		return fmt.Errorf("check tenant address space: %w", err)
	}
	if overlaps {
		return ErrCIDROverlap
	}
	_, err = tx.Exec(`
        INSERT INTO networks (id, name, cidr, tenant_id, project_id, external)
        VALUES ($1, $2, $3, $4, $5, $6)
    `, n.ID, n.Name, n.CIDR, n.TenantID, n.ProjectID, n.External)
	if err != nil {
		return fmt.Errorf("insert network: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit network allocation: %w", err)
	}
	return nil
}

func (s *PostgresStore) ListNetworks() ([]Network, error) {
	rows, err := s.db.Query(`SELECT id, name, cidr, tenant_id, project_id, external FROM networks`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nets []Network
	for rows.Next() {
		var n Network
		if err := rows.Scan(&n.ID, &n.Name, &n.CIDR, &n.TenantID, &n.ProjectID, &n.External); err != nil {
			return nil, err
		}
		nets = append(nets, n)
	}
	return nets, nil
}

func (s *PostgresStore) GetNetwork(id string) (Network, error) {
	var n Network
	err := s.db.QueryRow(`
        SELECT id, name, cidr, tenant_id, project_id, external
        FROM networks WHERE id = $1
    `, id).Scan(&n.ID, &n.Name, &n.CIDR, &n.TenantID, &n.ProjectID, &n.External)
	return n, err
}

func (s *PostgresStore) GetNetworkInProject(id, tenantID, projectID string) (Network, error) {
	var network Network
	err := s.db.QueryRow(`
		SELECT id, name, cidr, tenant_id, project_id, external
		FROM networks
		WHERE id = $1 AND tenant_id = $2 AND project_id = $3
	`, id, tenantID, projectID).Scan(
		&network.ID, &network.Name, &network.CIDR, &network.TenantID,
		&network.ProjectID, &network.External,
	)
	return network, err
}

func (s *PostgresStore) ListNetworksInProject(tenantID, projectID string) ([]Network, error) {
	rows, err := s.db.Query(`
		SELECT id, name, cidr, tenant_id, project_id, external
		FROM networks
		WHERE tenant_id = $1 AND project_id = $2
		ORDER BY created_at, id
	`, tenantID, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	networks := make([]Network, 0)
	for rows.Next() {
		var network Network
		if err := rows.Scan(
			&network.ID, &network.Name, &network.CIDR, &network.TenantID,
			&network.ProjectID, &network.External,
		); err != nil {
			return nil, err
		}
		networks = append(networks, network)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return networks, nil
}

func (s *PostgresStore) AuthorizeProject(userID, tenantID, projectID string) (bool, error) {
	var authorized bool
	err := s.db.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM projects p
			JOIN project_memberships pm ON pm.project_id = p.id
			JOIN tenant_memberships tm ON tm.tenant_id = p.tenant_id
			WHERE p.id = $1
			  AND p.tenant_id = $2
			  AND pm.user_id = $3
			  AND tm.user_id = $3
		)
	`, projectID, tenantID, userID).Scan(&authorized)
	return authorized, err
}

//
// SUBNETS
//

func (s *PostgresStore) CreateSubnet(sub Subnet) error {
	if err := NewPolicy().ValidateSubnet(sub); err != nil {
		return fmt.Errorf("validate subnet: %w", err)
	}
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin subnet allocation: %w", err)
	}
	defer tx.Rollback()
	var lockResult any
	if err := tx.QueryRow(`SELECT pg_advisory_xact_lock(hashtext($1))`, sub.NetworkID).Scan(&lockResult); err != nil {
		return fmt.Errorf("lock network address space: %w", err)
	}
	var network Network
	if err := tx.QueryRow(`
		SELECT id, name, cidr, tenant_id, external
		FROM networks WHERE id = $1
	`).Scan(
		&network.ID, &network.Name, &network.CIDR, &network.TenantID, &network.External,
	); err != nil {
		return fmt.Errorf("load parent network: %w", err)
	}
	if err := NewPolicy().ValidateSubnetInNetwork(sub, network); err != nil {
		return fmt.Errorf("validate subnet against network: %w", err)
	}
	var overlaps bool
	if err := tx.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM subnets
			WHERE network_id = $1 AND cidr && $2::cidr
		)
	`, sub.NetworkID, sub.CIDR).Scan(&overlaps); err != nil {
		return fmt.Errorf("check network address space: %w", err)
	}
	if overlaps {
		return ErrCIDROverlap
	}
	_, err = tx.Exec(`
        INSERT INTO subnets (id, network_id, cidr, gateway_ip)
        VALUES ($1, $2, $3, $4)
    `, sub.ID, sub.NetworkID, sub.CIDR, sub.GatewayIP)
	if err != nil {
		return fmt.Errorf("insert subnet: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit subnet allocation: %w", err)
	}
	return nil
}

func (s *PostgresStore) ListSubnets() ([]Subnet, error) {
	rows, err := s.db.Query(`SELECT id, network_id, cidr, gateway_ip FROM subnets`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []Subnet
	for rows.Next() {
		var s Subnet
		if err := rows.Scan(&s.ID, &s.NetworkID, &s.CIDR, &s.GatewayIP); err != nil {
			return nil, err
		}
		subs = append(subs, s)
	}
	return subs, nil
}

func (s *PostgresStore) GetSubnet(id string) (Subnet, error) {
	var subnet Subnet
	err := s.db.QueryRow(`
		SELECT id, network_id, cidr, gateway_ip
		FROM subnets WHERE id = $1
	`, id).Scan(&subnet.ID, &subnet.NetworkID, &subnet.CIDR, &subnet.GatewayIP)
	return subnet, err
}

func (s *PostgresStore) GetSubnetByNetwork(networkID string) (Subnet, error) {
	var subnet Subnet
	err := s.db.QueryRow(`
		SELECT id, network_id, cidr, gateway_ip
		FROM subnets WHERE network_id = $1
		ORDER BY created_at
		LIMIT 1
	`, networkID).Scan(&subnet.ID, &subnet.NetworkID, &subnet.CIDR, &subnet.GatewayIP)
	return subnet, err
}

//
// ROUTERS
//

func (s *PostgresStore) CreateRouter(r Router) error {
	_, err := s.db.Exec(`
        INSERT INTO routers (id, name, tenant_id, external_network_id)
        VALUES ($1, $2, $3, $4)
    `, r.ID, r.Name, r.TenantID, r.ExternalNetworkID)
	return err
}

func (s *PostgresStore) ListRouters() ([]Router, error) {
	rows, err := s.db.Query(`
        SELECT id, name, tenant_id, external_network_id
        FROM routers
    `)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var routers []Router
	for rows.Next() {
		var r Router
		if err := rows.Scan(&r.ID, &r.Name, &r.TenantID, &r.ExternalNetworkID); err != nil {
			return nil, err
		}
		routers = append(routers, r)
	}
	return routers, nil
}

func (s *PostgresStore) GetRouter(id string) (Router, error) {
	var router Router
	err := s.db.QueryRow(`
		SELECT id, name, tenant_id, external_network_id
		FROM routers WHERE id = $1
	`, id).Scan(&router.ID, &router.Name, &router.TenantID, &router.ExternalNetworkID)
	return router, err
}

func (s *PostgresStore) GetRouterBySubnet(subnetID string) (Router, error) {
	var router Router
	err := s.db.QueryRow(`
		SELECT r.id, r.name, r.tenant_id, r.external_network_id
		FROM routers r
		JOIN router_interfaces ri ON ri.router_id = r.id
		WHERE ri.subnet_id = $1
		ORDER BY ri.created_at
		LIMIT 1
	`, subnetID).Scan(&router.ID, &router.Name, &router.TenantID, &router.ExternalNetworkID)
	return router, err
}

func (s *PostgresStore) CreateRouterInterface(routerInterface RouterInterface) error {
	_, err := s.db.Exec(`
		INSERT INTO router_interfaces (id, router_id, subnet_id, ip_address)
		VALUES ($1, $2, $3, $4)
	`, routerInterface.ID, routerInterface.RouterID, routerInterface.SubnetID, routerInterface.IPAddress)
	return err
}

//
// PORTS
//

func (s *PostgresStore) CreatePort(p Port) error {
	_, err := s.db.Exec(`
        INSERT INTO ports (id, network_id, device_id, mac_address)
        VALUES ($1, $2, $3, $4)
    `, p.ID, p.NetworkID, p.DeviceID, p.MAC)
	return err
}

func (s *PostgresStore) ListPorts() ([]Port, error) {
	rows, err := s.db.Query(`
        SELECT id, network_id, device_id, mac_address
        FROM ports
    `)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ports []Port
	for rows.Next() {
		var p Port
		if err := rows.Scan(&p.ID, &p.NetworkID, &p.DeviceID, &p.MAC); err != nil {
			return nil, err
		}
		ports = append(ports, p)
	}
	return ports, nil
}

func (s *PostgresStore) GetPort(id string) (Port, error) {
	var port Port
	err := s.db.QueryRow(`
        SELECT id, network_id, device_id, mac_address
        FROM ports WHERE id = $1
    `, id).Scan(&port.ID, &port.NetworkID, &port.DeviceID, &port.MAC)
	return port, err
}

//
// SECURITY GROUPS
//

func (s *PostgresStore) CreateSecurityGroup(sg SecurityGroup) error {
	_, err := s.db.Exec(`
        INSERT INTO security_groups (id, name, tenant_id)
        VALUES ($1, $2, $3)
    `, sg.ID, sg.Name, sg.TenantID)
	return err
}

func (s *PostgresStore) CreateFloatingIP(floatingIP FloatingIP) error {
	_, err := s.db.Exec(`
		INSERT INTO floating_ips (id, tenant_id, external_ip, internal_port_id, internal_ip)
		VALUES ($1, $2, $3, $4, $5)
	`, floatingIP.ID, floatingIP.TenantID, floatingIP.ExternalIP, floatingIP.InternalPortID, floatingIP.InternalIP)
	return err
}

func (s *PostgresStore) CreateNATRule(natRule NATRule) error {
	_, err := s.db.Exec(`
		INSERT INTO nat_rules (id, router_id, external_ip, internal_ip, type)
		VALUES ($1, $2, $3, $4, $5)
	`, natRule.ID, natRule.RouterID, natRule.ExternalIP, natRule.InternalIP, natRule.Type)
	return err
}

func (s *PostgresStore) ListSecurityGroups() ([]SecurityGroup, error) {
	rows, err := s.db.Query(`
        SELECT id, name, tenant_id
        FROM security_groups
    `)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sgs []SecurityGroup
	for rows.Next() {
		var sg SecurityGroup
		if err := rows.Scan(&sg.ID, &sg.Name, &sg.TenantID); err != nil {
			return nil, err
		}
		sgs = append(sgs, sg)
	}
	return sgs, nil
}
