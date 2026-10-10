package nic

// Stats represents basic NIC statistics.
type Stats struct {
	Interface string
	Raw       string
}

// Config represents a NIC configuration snapshot.
type Config struct {
	Interface string
	Profile   Profile
}
