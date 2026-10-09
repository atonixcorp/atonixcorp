package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"atonixcorp/cli/internal/analytics"

	"github.com/spf13/cobra"
)

var email string
var password string

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to AtonixCorp",
	Run: func(cmd *cobra.Command, args []string) {

		// Prepare request body
		body := map[string]string{
			"email":    email,
			"password": password,
		}

		jsonBody, _ := json.Marshal(body)

		// Send login request
		resp, err := http.Post(os.Getenv("ATONIX_API")+"/auth/login", "application/json", bytes.NewBuffer(jsonBody))
		if err != nil {
			fmt.Println("Error:", err)
			return
		}

		defer resp.Body.Close()

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)

		if resp.StatusCode != 200 {
			fmt.Println("Login failed:", result["error"])
			return
		}

		token := result["token"].(string)

		// Save token locally
		os.WriteFile("session.jwt", []byte(token), 0644)

		fmt.Println("Login successful!")

		// ============================
		// SEND ANALYTICS EVENT
		// ============================
		analytics.SendEvent("auth", map[string]any{
			"action": "login",
			"user":   email,
		})
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
	loginCmd.Flags().StringVarP(&email, "email", "e", "", "Email address")
	loginCmd.Flags().StringVarP(&password, "password", "p", "", "Password")
}
