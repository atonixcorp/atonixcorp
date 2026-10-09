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

var jobID string

// ROOT JOB COMMAND
var jobCmd = &cobra.Command{
	Use:   "job",
	Short: "Manage and inspect jobs in AtonixCorp Cloud",
	Long:  "Query job status, logs, and analytics from AtonixCorp Cloud.",
}

// JOB STATUS COMMAND
var jobStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check the status of a job",
	Run: func(cmd *cobra.Command, args []string) {

		if jobID == "" {
			fmt.Println("Error: --id is required")
			return
		}

		// ============================
		// CALL JOB STATUS API
		// ============================
		resp, err := http.Get(os.Getenv("ATONIX_API") + "/job/status?id=" + jobID)
		if err != nil {
			fmt.Println("Error contacting job API:", err)
			return
		}
		defer resp.Body.Close()

		var result map[string]any
		json.NewDecoder(resp.Body).Decode(&result)

		if resp.StatusCode != 200 {
			fmt.Println("Job status error:", result["error"])
			return
		}

		// ============================
		// DISPLAY JOB STATUS
		// ============================
		fmt.Println("=== AtonixCorp Job Status ===")
		fmt.Println("Job ID:", jobID)
		fmt.Println("Status:", result["status"])
		fmt.Println("Progress:", result["progress"])
		fmt.Println("Started:", result["started"])
		fmt.Println("Updated:", result["updated"])
		fmt.Println("Time:", time.Now().Format(time.RFC3339))

		// ============================
		// SEND ANALYTICS EVENT
		// ============================
		analytics.SendEvent("job", map[string]any{
			"action": "status-check",
			"job_id": jobID,
			"time":   time.Now().Format(time.RFC3339),
		})
	},
}

// JOB LOGS COMMAND
var jobLogsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Fetch logs for a job",
	Run: func(cmd *cobra.Command, args []string) {

		if jobID == "" {
			fmt.Println("Error: --id is required")
			return
		}

		// ============================
		// CALL JOB LOGS API
		// ============================
		resp, err := http.Get(os.Getenv("ATONIX_API") + "/job/logs?id=" + jobID)
		if err != nil {
			fmt.Println("Error contacting job logs API:", err)
			return
		}
		defer resp.Body.Close()

		var result map[string]any
		json.NewDecoder(resp.Body).Decode(&result)

		if resp.StatusCode != 200 {
			fmt.Println("Job logs error:", result["error"])
			return
		}

		// ============================
		// DISPLAY JOB LOGS
		// ============================
		fmt.Println("=== AtonixCorp Job Logs ===")
		fmt.Println("Job ID:", jobID)
		fmt.Println("Logs:")
		fmt.Println(result["logs"])

		// ============================
		// SEND ANALYTICS EVENT
		// ============================
		analytics.SendEvent("job", map[string]any{
			"action": "logs-fetch",
			"job_id": jobID,
			"time":   time.Now().Format(time.RFC3339),
		})
	},
}

// REGISTER COMMANDS
func init() {
	rootCmd.AddCommand(jobCmd)

	// job status
	jobCmd.AddCommand(jobStatusCmd)
	jobStatusCmd.Flags().StringVarP(&jobID, "id", "i", "", "Job ID to inspect")

	// job logs
	jobCmd.AddCommand(jobLogsCmd)
	jobLogsCmd.Flags().StringVarP(&jobID, "id", "i", "", "Job ID to fetch logs for")
}
