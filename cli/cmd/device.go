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

var deviceID string

// ROOT DEVICE COMMAND
var deviceCmd = &cobra.Command{
	Use:   "device",
	Short: "Manage and inspect devices in AtonixCorp Cloud",
	Long:  "Query device status, diagnostics, and analytics from AtonixCorp Cloud.",
}

// DEVICE STATUS COMMAND
var deviceStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check the status of a device",
	Run: func(cmd *cobra.Command, args []string) {

		if deviceID == "" {
			fmt.Println("Error: --id is required")
			return
		}

		// ============================
		// CALL DEVICE STATUS API
		// ============================
		resp, err := http.Get(os.Getenv("ATONIX_API") + "/device/status?id=" + deviceID)
		if err != nil {
			fmt.Println("Error contacting device API:", err)
			return
		}
		defer resp.Body.Close()

		var result map[string]any
		json.NewDecoder(resp.Body).Decode(&result)

		if resp.StatusCode != 200 {
			fmt.Println("Device status error:", result["error"])
			return
		}

		// ============================
		// DISPLAY DEVICE STATUS
		// ============================
		fmt.Println("=== AtonixCorp Device Status ===")
		fmt.Println("Device ID:", deviceID)
		fmt.Println("Status:", result["status"])
		fmt.Println("Battery:", result["battery"])
		fmt.Println("Temperature:", result["temperature"])
		fmt.Println("Last Seen:", result["last_seen"])
		fmt.Println("Time:", time.Now().Format(time.RFC3339))

		// ============================
		// SEND ANALYTICS EVENT
		// ============================
		analytics.SendEvent("device", map[string]any{
			"action":    "status-check",
			"device_id": deviceID,
			"time":      time.Now().Format(time.RFC3339),
		})
	},
}

// REGISTER COMMANDS
func init() {
	rootCmd.AddCommand(deviceCmd)

	// device status
	deviceCmd.AddCommand(deviceStatusCmd)
	deviceStatusCmd.Flags().StringVarP(&deviceID, "id", "i", "", "Device ID to inspect")
}
