package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"atonixcorp/cli/internal/analytics"

	"github.com/spf13/cobra"
)

var uploadPath string
var downloadID string

// ROOT FILE COMMAND
var fileCmd = &cobra.Command{
	Use:   "file",
	Short: "Manage files in AtonixCorp Cloud",
	Long:  "Upload, download, and inspect files stored in AtonixCorp Cloud.",
}

// FILE UPLOAD COMMAND
var fileUploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "Upload a file to AtonixCorp Cloud",
	Run: func(cmd *cobra.Command, args []string) {

		if uploadPath == "" {
			fmt.Println("Error: --path is required")
			return
		}

		// Read file
		data, err := os.ReadFile(uploadPath)
		if err != nil {
			fmt.Println("Error reading file:", err)
			return
		}

		// Prepare request
		body := map[string]any{
			"filename": uploadPath,
			"content":  string(data),
		}

		jsonBody, _ := json.Marshal(body)

		// Send request
		resp, err := http.Post(os.Getenv("ATONIX_API")+"/file/upload", "application/json", bytes.NewBuffer(jsonBody))
		if err != nil {
			fmt.Println("Upload error:", err)
			return
		}
		defer resp.Body.Close()

		var result map[string]any
		json.NewDecoder(resp.Body).Decode(&result)

		if resp.StatusCode != 200 {
			fmt.Println("Upload failed:", result["error"])
			return
		}

		fmt.Println("File uploaded successfully!")

		// ANALYTICS EVENT
		analytics.SendEvent("file", map[string]any{
			"action":   "upload",
			"filename": uploadPath,
			"size":     len(data),
		})
	},
}

// FILE DOWNLOAD COMMAND
var fileDownloadCmd = &cobra.Command{
	Use:   "download",
	Short: "Download a file from AtonixCorp Cloud",
	Run: func(cmd *cobra.Command, args []string) {

		if downloadID == "" {
			fmt.Println("Error: --id is required")
			return
		}

		// Send request
		resp, err := http.Get(os.Getenv("ATONIX_API") + "/file/download?id=" + downloadID)
		if err != nil {
			fmt.Println("Download error:", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			var result map[string]any
			json.NewDecoder(resp.Body).Decode(&result)
			fmt.Println("Download failed:", result["error"])
			return
		}

		// Read file content
		content, _ := io.ReadAll(resp.Body)

		// Save locally
		filename := downloadID + ".downloaded"
		os.WriteFile(filename, content, 0644)

		fmt.Println("File downloaded:", filename)

		// ANALYTICS EVENT
		analytics.SendEvent("file", map[string]any{
			"action":  "download",
			"file_id": downloadID,
			"size":    len(content),
		})
	},
}

// REGISTER COMMANDS
func init() {
	rootCmd.AddCommand(fileCmd)

	// Upload
	fileCmd.AddCommand(fileUploadCmd)
	fileUploadCmd.Flags().StringVarP(&uploadPath, "path", "p", "", "Path to file")

	// Download
	fileCmd.AddCommand(fileDownloadCmd)
	fileDownloadCmd.Flags().StringVarP(&downloadID, "id", "i", "", "File ID to download")
}
