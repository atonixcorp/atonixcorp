package scheduler

import "fmt"

type Node struct {
	Name   string
	CPU    int
	Memory int
}

type Scheduler struct {
	Nodes []Node
}

func NewScheduler(nodes []Node) *Scheduler {
	return &Scheduler{Nodes: nodes}
}

func (s *Scheduler) SelectNode(imageID string) (Node, error) {
	// TODO: implement real scheduling logic
	if len(s.Nodes) == 0 {
		return Node{}, fmt.Errorf("no nodes available")
	}
	return s.Nodes[0], nil
}
