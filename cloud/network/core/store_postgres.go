package core

import (
	"database/sql"

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

//
// NETWORKS
//

func (s *PostgresStore) CreateNetwork(n Network) error {
	_, err := s.db.Exec(`
        INSERT INTO networks (id, name, cidr, tenant_id, external)
        VALUES ($1, $2, $3, $4, $5)
    `, n.ID, n.Name, n.CIDR, n.TenantID, n.External)
	return err
}

func (s *PostgresStore) ListNetworks() ([]Network, error) {
	rows, err := s.db.Query(`SELECT id, name, cidr, tenant_id, external FROM networks`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nets []Network
	for rows.Next() {
		var n Network
		if err := rows.Scan(&n.ID, &n.Name, &n.CIDR, &n.TenantID, &n.External); err != nil {
			return nil, err
		}
		nets = append(nets, n)
	}
	return nets, nil
}

func (s *PostgresStore) GetNetwork(id string) (Network, error) {
	var n Network
	err := s.db.QueryRow(`
        SELECT id, name, cidr, tenant_id, external
        FROM networks WHERE id = $1
    `, id).Scan(&n.ID, &n.Name, &n.CIDR, &n.TenantID, &n.External)
	return n, err
}

//
// SUBNETS
//

func (s *PostgresStore) CreateSubnet(sub Subnet) error {
	_, err := s.db.Exec(`
        INSERT INTO subnets (id, network_id, cidr, gateway_ip)
        VALUES ($1, $2, $3, $4)
    `, sub.ID, sub.NetworkID, sub.CIDR, sub.GatewayIP)
	return err
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
