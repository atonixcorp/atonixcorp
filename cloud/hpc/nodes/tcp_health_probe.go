package nodes

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

type TCPHealthProbe struct {
	port    int
	timeout time.Duration
}

func NewTCPHealthProbe(port int, timeout time.Duration) (*TCPHealthProbe, error) {
	if port < 1 || port > 65535 {
		return nil, errors.New("node health port must be between 1 and 65535")
	}
	if timeout <= 0 {
		return nil, errors.New("node health timeout must be positive")
	}
	return &TCPHealthProbe{port: port, timeout: timeout}, nil
}

func (p *TCPHealthProbe) ProbeNode(ctx context.Context, node Node) (HealthProbeResult, error) {
	if p == nil {
		return HealthProbeResult{}, errors.New("node health probe is not configured")
	}
	host := strings.TrimSpace(node.Address)
	if host == "" {
		host = strings.TrimSpace(node.Hostname)
	}
	if host == "" {
		return HealthProbeResult{}, fmt.Errorf("node %q has no address or hostname", node.ID)
	}
	address := host
	if _, _, err := net.SplitHostPort(host); err != nil {
		address = net.JoinHostPort(host, strconv.Itoa(p.port))
	}
	probeCtx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()
	conn, err := (&net.Dialer{}).DialContext(probeCtx, "tcp", address)
	if err != nil {
		if ctx.Err() != nil {
			return HealthProbeResult{}, ctx.Err()
		}
		return HealthProbeResult{
			NodeID:    node.ID,
			Reachable: false,
			TempOK:    true,
			Timestamp: time.Now().UTC(),
		}, nil
	}
	if err := conn.Close(); err != nil {
		return HealthProbeResult{}, fmt.Errorf("close node health connection: %w", err)
	}
	return HealthProbeResult{
		NodeID:    node.ID,
		Reachable: true,
		TempOK:    true,
		Timestamp: time.Now().UTC(),
	}, nil
}
