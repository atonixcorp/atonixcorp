package nvme

// Mode defines NVMe performance modes.
type Mode string

const (
	ModePerformance Mode = "performance"
	ModeBalanced    Mode = "balanced"
	ModePowerSave   Mode = "powersave"
)

// Feature IDs commonly used with nvme-cli.
const (
	FeatureWriteCache   = "0x06"
	FeaturePowerProfile = "0x0C"
)
