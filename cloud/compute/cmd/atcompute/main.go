package main

import (
	"atonixcorp/cloud/compute/agent"
	"atonixcorp/cloud/compute/rpc"
	"log"
)

func main() {
	backend := rpc.NewLocalRPC()

	nodeAgent := agent.NewAgent("node-001", backend)
	if err := nodeAgent.Register(); err != nil {
		log.Fatal(err)
	}
	if err := nodeAgent.ReportHealth(); err != nil {
		log.Fatal(err)
	}
	nodeAgent.Start()
}
