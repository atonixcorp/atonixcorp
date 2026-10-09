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

var userID string

// ROOT USER COMMAND
var userCmd = &cobra.Command{
	Use:   "user",
	Short: "Manage and inspect users in AtonixCorp Cloud",
	Long:  "Query user profiles, activity, and sessions from AtonixCorp Cloud.",
}

// USER PROFILE COMMAND
var userProfileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Fetch user profile information",
	Run: func(cmd *cobra.Command, args []string) {

		if userID == "" {
			fmt.Println("Error: --id is required")
			return
		}

		// ============================
		// CALL USER PROFILE API
		// ============================
		resp, err := http.Get(os.Getenv("ATONIX_API") + "/user/profile?id=" + userID)
		if err != nil {
			fmt.Println("Error contacting user API:", err)
			return
		}
		defer resp.Body.Close()

		var result map[string]any
		json.NewDecoder(resp.Body).Decode(&result)

		if resp.StatusCode != 200 {
			fmt.Println("User profile error:", result["error"])
			return
		}

		// ============================
		// DISPLAY USER PROFILE
		// ============================
		fmt.Println("=== AtonixCorp User Profile ===")
		fmt.Println("User ID:", userID)
		fmt.Println("Name:", result["name"])
		fmt.Println("Email:", result["email"])
		fmt.Println("Role:", result["role"])
		fmt.Println("Created:", result["created"])
		fmt.Println("Time:", time.Now().Format(time.RFC3339))

		// ============================
		// SEND ANALYTICS EVENT
		// ============================
		analytics.SendEvent("user", map[string]any{
			"action":  "profile-fetch",
			"user_id": userID,
			"time":    time.Now().Format(time.RFC3339),
		})
	},
}

// USER ACTIVITY COMMAND
var userActivityCmd = &cobra.Command{
	Use:   "activity",
	Short: "Fetch user activity logs",
	Run: func(cmd *cobra.Command, args []string) {

		if userID == "" {
			fmt.Println("Error: --id is required")
			return
		}

		// ============================
		// CALL USER ACTIVITY API
		// ============================
		resp, err := http.Get(os.Getenv("ATONIX_API") + "/user/activity?id=" + userID)
		if err != nil {
			fmt.Println("Error contacting user activity API:", err)
			return
		}
		defer resp.Body.Close()

		var result map[string]any
		json.NewDecoder(resp.Body).Decode(&result)

		if resp.StatusCode != 200 {
			fmt.Println("User activity error:", result["error"])
			return
		}

		// ============================
		// DISPLAY USER ACTIVITY
		// ============================
		fmt.Println("=== AtonixCorp User Activity ===")
		fmt.Println("User ID:", userID)
		fmt.Println("Activity Logs:")
		fmt.Println(result["logs"])
		fmt.Println("Time:", time.Now().Format(time.RFC3339))

		// ============================
		// SEND ANALYTICS EVENT
		// ============================
		analytics.SendEvent("user", map[string]any{
			"action":  "activity-fetch",
			"user_id": userID,
			"time":    time.Now().Format(time.RFC3339),
		})
	},
}

// USER SESSIONS COMMAND
var userSessionsCmd = &cobra.Command{
	Use:   "sessions",
	Short: "Fetch active user sessions",
	Run: func(cmd *cobra.Command, args []string) {

		if userID == "" {
			fmt.Println("Error: --id is required")
			return
		}

		// ============================
		// CALL USER SESSIONS API
		// ============================
		resp, err := http.Get(os.Getenv("ATONIX_API") + "/user/sessions?id=" + userID)
		if err != nil {
			fmt.Println("Error contacting user sessions API:", err)
			return
		}
		defer resp.Body.Close()

		var result map[string]any
		json.NewDecoder(resp.Body).Decode(&result)

		if resp.StatusCode != 200 {
			fmt.Println("User sessions error:", result["error"])
			return
		}

		// ============================
		// DISPLAY USER SESSIONS
		// ============================
		fmt.Println("=== AtonixCorp User Sessions ===")
		fmt.Println("User ID:", userID)
		fmt.Println("Active Sessions:")
		fmt.Println(result["sessions"])
		fmt.Println("Time:", time.Now().Format(time.RFC3339))

		// ============================
		// SEND ANALYTICS EVENT
		// ============================
		analytics.SendEvent("user", map[string]any{
			"action":  "sessions-fetch",
			"user_id": userID,
			"time":    time.Now().Format(time.RFC3339),
		})
	},
}

// REGISTER COMMANDS
func init() {
	rootCmd.AddCommand(userCmd)

	// user profile
	userCmd.AddCommand(userProfileCmd)
	userProfileCmd.Flags().StringVarP(&userID, "id", "i", "", "User ID to inspect")

	// user activity
	userCmd.AddCommand(userActivityCmd)
	userActivityCmd.Flags().StringVarP(&userID, "id", "i", "", "User ID to fetch activity for")

	// user sessions
	userCmd.AddCommand(userSessionsCmd)
	userSessionsCmd.Flags().StringVarP(&userID, "id", "i", "", "User ID to fetch sessions for")
}
