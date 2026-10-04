package nic

import "os/exec"

// SetProfile applies a NIC profile to a given interface.
// Later you can make this smarter per node class (storage, block, fs, etc.).
func SetProfile(iface string, profile Profile) error {
	switch profile {
	case ProfileLatency:
		// Lower interrupt coalescing, smaller queues.
		_ = exec.Command("ethtool", "-C", iface, "rx-usecs", "0").Run()
		_ = exec.Command("ethtool", "-G", iface, "rx", "4096", "tx", "4096").Run()
	case ProfileThroughput:
		// Higher coalescing, larger queues.
		_ = exec.Command("ethtool", "-C", iface, "rx-usecs", "100").Run()
		_ = exec.Command("ethtool", "-G", iface, "rx", "8192", "tx", "8192").Run()
	case ProfileBalanced:
		// Middle‑ground defaults.
		_ = exec.Command("ethtool", "-C", iface, "rx-usecs", "50").Run()
		_ = exec.Command("ethtool", "-G", iface, "rx", "6144", "tx", "6144").Run()
	default:
		// Unknown profile: no‑op.
	}
	return nil
}
