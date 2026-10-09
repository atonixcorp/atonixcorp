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

var systemCmd = &cobra.Command{
	Use:   "system",
	Short: "Check AtonixCorp system status",
	Long:  "Retrieve system status, uptime, and diagnostics from AtonixCorp Cloud.",
	Run: func(cmd *cobra.Command, args []string) {

		// ============================
		// CALL SYSTEM STATUS API
		// ============================
		resp, err := http.Get(os.Getenv("ATONIX_API") + "/system/status")
		if err != nil {
			fmt.Println("Error contacting system API:", err)
			return
		}
		defer resp.Body.Close()

		var result map[string]any
		json.NewDecoder(resp.Body).Decode(&result)

		if resp.StatusCode != 200 {
			fmt.Println("System status error:", result["error"])
			return
		}

		// ============================
		// DISPLAY SYSTEM STATUS
		// ============================
		fmt.Println("=== AtonixCorp System Status ===")
		fmt.Println("Status:", result["status"])
		fmt.Println("Uptime:", result["uptime"])
		fmt.Println("Version:", result["version"])
		fmt.Println("Time:", time.Now().Format(time.RFC3339))

		// ============================
		// SEND ANALYTICS EVENT
		// ============================
		analytics.SendEvent("system", map[string]any{
			"action": "status-check",
			"time":   time.Now().Format(time.RFC3339),
		})
	},
}

func init() {
	rootCmd.AddCommand(systemCmd)
}
