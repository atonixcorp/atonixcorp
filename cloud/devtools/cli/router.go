package router

import (
	"fmt"

	"atonixcorp/cloud/devtools/cli/command"
)

func Dispatch(args []string) {
	if len(args) < 2 {
		command.Help()
		return
	}
	if args[1] == "identity" {
		command.IdentityInfo()
		return
	}
	if len(args) < 3 {
		command.Help()
		return
	}

	action := args[1]
	resource := args[2]
	switch action {
	case "create":
		switch resource {
		case "compute":
			command.CreateCompute(args[3:])
		case "network":
			command.CreateNetwork(args[3:])
		case "storage":
			command.CreateStorage(args[3:])
		case "container":
			command.CreateContainer(args[3:])
		default:
			fmt.Println("Unknown resource:", resource)
		}
	case "delete":
		switch resource {
		case "compute":
			command.DeleteCompute(args[3:])
		case "network":
			command.DeleteNetwork(args[3:])
		case "storage":
			command.DeleteStorage(args[3:])
		case "container":
			command.DeleteContainer(args[3:])
		case "object":
			command.DeleteObject(args[3:])
		default:
			fmt.Println("Unknown resource:", resource)
		}
	case "list":
		switch resource {
		case "compute":
			command.ListCompute()
		case "network":
			command.ListNetwork()
		case "storage":
			command.ListStorage()
		case "projects":
			command.ListProjects()
		case "containers":
			command.ListContainers()
		case "metrics":
			command.ListMetrics()
		case "events":
			command.ListEvents()
		default:
			fmt.Println("Unknown resource:", resource)
		}
	case "upload":
		if resource == "object" {
			command.UploadObject(args[3:])
		} else {
			fmt.Println("Unknown upload target:", resource)
		}
	default:
		command.Help()
	}
}
