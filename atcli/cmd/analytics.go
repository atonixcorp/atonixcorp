package cmd

import (
	"atonixcorp/cli/internal/analytics"
	"fmt"

	"github.com/spf13/cobra"
)

// ROOT ANALYTICS COMMAND
var analyticsCmd = &cobra.Command{
	Use:   "analytics",
	Short: "Query AtonixCorp analytics",
}

// BASIC ANALYTICS COMMANDS
var usageCmd = &cobra.Command{
	Use:   "usage",
	Short: "Show usage analytics",
	Run: func(cmd *cobra.Command, args []string) {
		data, err := analytics.Query("/analytics/usage")
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		fmt.Println("Usage Analytics:")
		fmt.Println(data)
	},
}

var nodesCmd = &cobra.Command{
	Use:   "nodes",
	Short: "Show node analytics",
	Run: func(cmd *cobra.Command, args []string) {
		data, err := analytics.Query("/analytics/nodes")
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		fmt.Println("Node Analytics:")
		fmt.Println(data)
	},
}

var devicesCmd = &cobra.Command{
	Use:   "devices",
	Short: "Show device analytics",
	Run: func(cmd *cobra.Command, args []string) {
		data, err := analytics.Query("/analytics/devices")
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		fmt.Println("Device Analytics:")
		fmt.Println(data)
	},
}

var jobsCmd = &cobra.Command{
	Use:   "jobs",
	Short: "Show job analytics",
	Run: func(cmd *cobra.Command, args []string) {
		data, err := analytics.Query("/analytics/jobs")
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		fmt.Println("Job Analytics:")
		fmt.Println(data)
	},
}

// DASHBOARD ROOT COMMAND
var dashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Show terminal analytics dashboards",
}

// DASHBOARD: USAGE
var dashboardUsageCmd = &cobra.Command{
	Use:   "usage",
	Short: "Show usage analytics dashboard",
	Run: func(cmd *cobra.Command, args []string) {
		data, err := analytics.Query("/analytics/usage")
		if err != nil {
			fmt.Println("Error:", err)
			return
		}

		fmt.Print("\n=== AtonixCorp Usage Dashboard ===\n\n")

		rows := [][]string{}
		max := 0

		for _, count := range data {
			c := int(count.(float64))
			if c > max {
				max = c
			}
		}

		for kind, count := range data {
			c := int(count.(float64))
			rows = append(rows, []string{
				kind,
				fmt.Sprintf("%d", c),
				analytics.Bar(c, max, 30),
			})
		}

		fmt.Println(analytics.Table(
			[]string{"Event Type", "Count", "Activity"},
			rows,
		))
	},
}

// DASHBOARD: NODES
var dashboardNodesCmd = &cobra.Command{
	Use:   "nodes",
	Short: "Show node analytics dashboard",
	Run: func(cmd *cobra.Command, args []string) {

		data, err := analytics.Query("/analytics/nodes")
		if err != nil {
			fmt.Println("Error:", err)
			return
		}

		fmt.Print("\n=== AtonixCorp Nodes Dashboard ===\n\n")

		nodes, ok := data["nodes"].([]any)
		if !ok {
			fmt.Println("Invalid node analytics format")
			return
		}

		rows := [][]string{}

		for _, n := range nodes {
			node := n.(map[string]any)

			id := fmt.Sprintf("%v", node["node_id"])
			status := fmt.Sprintf("%v", node["status"])

			cpu := int(node["cpu"].(float64))
			ram := int(node["ram"].(float64))

			rows = append(rows, []string{
				id,
				status,
				analytics.Bar(cpu, 100, 20),
				analytics.Bar(ram, 100, 20),
			})
		}

		fmt.Println(analytics.Table(
			[]string{"Node", "Status", "CPU", "RAM"},
			rows,
		))
	},
}

// REGISTER COMMANDS
func init() {
	// Basic analytics commands
	analyticsCmd.AddCommand(usageCmd)
	analyticsCmd.AddCommand(nodesCmd)
	analyticsCmd.AddCommand(devicesCmd)
	analyticsCmd.AddCommand(jobsCmd)

	// Dashboard commands
	dashboardCmd.AddCommand(dashboardUsageCmd)
	dashboardCmd.AddCommand(dashboardNodesCmd)

	analyticsCmd.AddCommand(dashboardCmd)

	// Register analytics root
	rootCmd.AddCommand(analyticsCmd)
}
