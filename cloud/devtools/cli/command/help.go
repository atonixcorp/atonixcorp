package command

import "fmt"

func Help() {
	fmt.Println("AtonixCorp Cloud CLI")
	fmt.Println("Usage:")
	fmt.Println("  atcloud create compute")
	fmt.Println("  atcloud create network")
	fmt.Println("  atcloud create storage")
	fmt.Println("  atcloud delete compute")
	fmt.Println("  atcloud delete network")
	fmt.Println("  atcloud delete storage")
	fmt.Println("  atcloud list compute")
	fmt.Println("  atcloud list network")
	fmt.Println("  atcloud list storage")
	fmt.Println("  atcloud list projects")
	fmt.Println("  atcloud identity")
}
