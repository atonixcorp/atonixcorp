package models

// ScheduleDecision represents the scheduler's choice of node for a container.
type ScheduleDecision struct {
	ContainerImage string `json:"container_image"`
	SelectedNode   string `json:"selected_node"`
	Reason         string `json:"reason"`
}
