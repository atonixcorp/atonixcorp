package bmc

// Sensor represents a single BMC sensor reading.
type Sensor struct {
	Name   string
	Value  string
	Status string
	Raw    string
}
