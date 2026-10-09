package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"atonixcorp/cli/internal/analytics"

	"github.com/spf13/cobra"
)

var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "Inspect AtonixCorp database status and metrics",
	Long:  "Query database health, uptime, and metrics from AtonixCorp Cloud.",
}

// DB STATUS COMMAND
var dbStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check database status",
	Run: func(cmd *cobra.Command, args []string) {

		// ============================
		// CALL DB STATUS API
		// ============================
		resp, err := http.Get(os.Getenv("ATONIX_API") + "/db/status")
		if err != nil {
			fmt.Println("Error contacting DB API:", err)
			return
		}
		defer resp.Body.Close()

		var result map[string]any
		json.NewDecoder(resp.Body).Decode(&result)

		if resp.StatusCode != 200 {
			fmt.Println("Database status error:", result["error"])
			return
		}

		// ============================
		// DISPLAY DB STATUS
		// ============================
		fmt.Println("=== AtonixCorp Database Status ===")
		fmt.Println("Status:", result["status"])
		fmt.Println("Uptime:", result["uptime"])
		fmt.Println("Connections:", result["connections"])
		fmt.Println("Version:", result["version"])
		fmt.Println("Time:", time.Now().Format(time.RFC3339))

		// ============================
		// SEND ANALYTICS EVENT
		// ============================
		analytics.SendEvent("db", map[string]any{
			"action": "status-check",
			"time":   time.Now().Format(time.RFC3339),
		})
	},
}

// DB METRICS COMMAND
var dbMetricsCmd = &cobra.Command{
	Use:   "metrics",
	Short: "Fetch database metrics",
	Run: func(cmd *cobra.Command, args []string) {

		// ============================
		// CALL DB METRICS API
		// ============================
		resp, err := http.Get(os.Getenv("ATONIX_API") + "/db/metrics")
		if err != nil {
			fmt.Println("Error contacting DB metrics API:", err)
			return
		}
		defer resp.Body.Close()

		var result map[string]any
		json.NewDecoder(resp.Body).Decode(&result)

		if resp.StatusCode != 200 {
			fmt.Println("Database metrics error:", result["error"])
			return
		}

		// ============================
		// DISPLAY DB METRICS
		// ============================
		fmt.Println("=== AtonixCorp Database Metrics ===")
		fmt.Println("Reads/sec:", result["reads"])
		fmt.Println("Writes/sec:", result["writes"])
		fmt.Println("Slow Queries:", result["slow_queries"])
		fmt.Println("Cache Hit Rate:", result["cache_hit"])
		fmt.Println("Time:", time.Now().Format(time.RFC3339))

		// ============================
		// SEND ANALYTICS EVENT
		// ============================
		analytics.SendEvent("db", map[string]any{
			"action": "metrics-fetch",
			"time":   time.Now().Format(time.RFC3339),
		})
	},
}

// REGISTER COMMANDS
func init() {
	rootCmd.AddCommand(dbCmd)

	// db status
	dbCmd.AddCommand(dbStatusCmd)

	// db metrics
	dbCmd.AddCommand(dbMetricsCmd)
}
