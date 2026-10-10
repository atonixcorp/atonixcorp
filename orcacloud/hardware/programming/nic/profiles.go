package nic

// Profile defines NIC tuning profiles.
type Profile string

const (
	ProfileLatency    Profile = "latency"
	ProfileThroughput Profile = "throughput"
	ProfileBalanced   Profile = "balanced"
)
