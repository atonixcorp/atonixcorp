package nvme

import "os/exec"

// SetMode configures NVMe performance mode using nvme-cli set-feature.
func SetMode(dev Device, mode Mode) error {
	switch mode {
	case ModePerformance:
		// Example: enable write cache, performance power profile.
		_ = exec.Command("nvme", "set-feature", dev.Path, "-f", FeatureWriteCache, "-v", "1").Run()
		_ = exec.Command("nvme", "set-feature", dev.Path, "-f", FeaturePowerProfile, "-v", "1").Run()
	case ModeBalanced:
		_ = exec.Command("nvme", "set-feature", dev.Path, "-f", FeatureWriteCache, "-v", "1").Run()
		_ = exec.Command("nvme", "set-feature", dev.Path, "-f", FeaturePowerProfile, "-v", "0").Run()
	case ModePowerSave:
		_ = exec.Command("nvme", "set-feature", dev.Path, "-f", FeatureWriteCache, "-v", "0").Run()
		_ = exec.Command("nvme", "set-feature", dev.Path, "-f", FeaturePowerProfile, "-v", "2").Run()
	default:
		// Unknown mode: no-op.
	}
	return nil
}
