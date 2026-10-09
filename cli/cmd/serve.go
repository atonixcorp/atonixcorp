package cmd

import (
	"fmt"
	"net/http"
	"time"

	"atonixcorp/cli/internal/analytics"

	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run ATCLI as a local server on port 8000",
	Run: func(cmd *cobra.Command, args []string) {

		fmt.Println("ATCLI server running on port 8000")

		// ============================
		// SEND ANALYTICS EVENT
		// ============================
		analytics.SendEvent("system", map[string]any{
			"action": "serve-start",
			"port":   8000,
		})

		// ============================
		// LOCAL ENDPOINT: ROOT
		// ============================
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "ATCLI is running locally on port 8000")
		})

		// ============================
		// LOCAL ENDPOINT: STATUS
		// ============================
		http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "ATCLI Status: OK\nTime: %s", time.Now().Format(time.RFC3339))
		})

		// ============================
		// LOCAL ENDPOINT: METRICS
		// ============================
		http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
			metrics := analytics.LocalMetrics
			fmt.Fprintf(w, "ATCLI Local Metrics:\n")
			for k, v := range metrics {
				fmt.Fprintf(w, "%s: %d\n", k, v)
			}
		})

		// ============================
		// LOCAL ENDPOINT: HEARTBEAT
		// ============================
		http.HandleFunc("/heartbeat", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "ATCLI Heartbeat OK")
		})

		// ============================
		// START SERVER
		// ============================
		http.ListenAndServe(":8000", nil)
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
