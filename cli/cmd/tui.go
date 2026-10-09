package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"atonixcorp/cli/internal/analytics"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var theme = "atonix"
var panel = 0 // 0=usage,1=nodes,2=devices,3=jobs,4=db

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Run the ATCLI live terminal dashboard",
	Long:  "A live, interactive terminal dashboard showing nodes, devices, jobs, DB, and system analytics.",
	Run: func(cmd *cobra.Command, args []string) {

		analytics.SendEvent("system", map[string]any{
			"action": "tui-start",
			"time":   time.Now().Format(time.RFC3339),
		})

		// Enable raw terminal mode
		oldState, _ := term.MakeRaw(int(os.Stdin.Fd()))
		defer term.Restore(int(os.Stdin.Fd()), oldState)

		for {
			clearScreen()
			fmt.Println(renderHeader())

			switch panel {
			case 0:
				fmt.Println(renderUsage())
			case 1:
				fmt.Println(renderNodes())
			case 2:
				fmt.Println(renderDevices())
			case 3:
				fmt.Println(renderJobs())
			case 4:
				fmt.Println(renderDB())
			}

			fmt.Println(renderFooter())

			// Non-blocking key read
			b := make([]byte, 1)
			os.Stdin.Read(b)

			switch b[0] {
			case 'q', 'Q':
				return
			case '\t':
				panel = (panel + 1) % 5
			case 'n', 'N':
				renderNodeInspector()
			case 'd', 'D':
				renderDeviceInspector()
			case 'j', 'J':
				renderJobInspector()
			case '1':
				theme = "atonix"
			case '2':
				theme = "solarized"
			case '3':
				theme = "nord"
			}

			time.Sleep(200 * time.Millisecond)
		}
	},
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}

//
// ─────────────────────────────────────────────────────────────
//   RENDER FUNCTIONS
// ─────────────────────────────────────────────────────────────
//

func renderHeader() string {
	switch theme {
	case "solarized":
		return "\033[38;5;136mAtonixCorp TUI — Solarized Theme\033[0m\n"
	case "nord":
		return "\033[38;5;110mAtonixCorp TUI — Nord Theme\033[0m\n"
	default:
		return "\033[38;5;45mAtonixCorp Sovereign Cloud — Live Dashboard\033[0m\n"
	}
}

func renderUsage() string {
	data := fetch("/analytics/usage")

	rows := ""
	max := 0

	for _, v := range data {
		c := int(v.(float64))
		if c > max {
			max = c
		}
	}

	for k, v := range data {
		c := int(v.(float64))
		rows += fmt.Sprintf(" %-12s %5d  %s  %s\n",
			k, c,
			analytics.Bar(c, max, 30),
			analytics.Spark([]int{c / 4, c / 3, c / 2, c}),
		)
	}

	return "\n[Usage]\n" + rows
}

func renderNodes() string {
	data := fetch("/analytics/nodes")
	nodes := data["nodes"].([]any)

	out := "\n[Nodes]\n"
	for _, n := range nodes {
		node := n.(map[string]any)
		cpu := int(node["cpu"].(float64))
		ram := int(node["ram"].(float64))

		out += fmt.Sprintf(" %-10s %-8s CPU:%s RAM:%s\n",
			node["node_id"],
			node["status"],
			analytics.Bar(cpu, 100, 20),
			analytics.Bar(ram, 100, 20),
		)
	}
	return out
}

func renderDevices() string {
	data := fetch("/analytics/devices")
	devices := data["devices"].([]any)

	out := "\n[Devices]\n"
	for _, d := range devices {
		dev := d.(map[string]any)
		out += fmt.Sprintf(" %-10s %-8s Batt:%s Temp:%s\n",
			dev["device_id"],
			dev["status"],
			dev["battery"],
			dev["temperature"],
		)
	}
	return out
}

func renderJobs() string {
	data := fetch("/analytics/jobs")
	jobs := data["jobs"].([]any)

	out := "\n[Jobs]\n"
	for _, j := range jobs {
		job := j.(map[string]any)
		out += fmt.Sprintf(" %-10s %-10s %s %s\n",
			job["job_id"],
			job["status"],
			analytics.Bar(int(job["progress"].(float64)), 100, 20),
			analytics.Spark([]int{
				int(job["progress"].(float64)) / 4,
				int(job["progress"].(float64)) / 2,
				int(job["progress"].(float64)),
			}),
		)
	}
	return out
}

func renderDB() string {
	data := fetch("/db/metrics")

	return fmt.Sprintf(`
[Database]
 Reads/sec:      %v
 Writes/sec:     %v
 Slow Queries:   %v
 Cache Hit Rate: %v
`, data["reads"], data["writes"], data["slow_queries"], data["cache_hit"])
}

func renderFooter() string {
	return `
Controls:
 TAB = Switch panel
 N   = Node inspector
 D   = Device inspector
 J   = Job inspector
 1   = Atonix theme
 2   = Solarized theme
 3   = Nord theme
 Q   = Quit
`
}

//
// ─────────────────────────────────────────────────────────────
//   INSPECTORS
// ─────────────────────────────────────────────────────────────
//

func renderNodeInspector() {
	clearScreen()
	fmt.Println("[Node Inspector]")
	fmt.Println("Enter Node ID:")

	id := readLine()

	data := fetch("/node/info?id=" + id)
	fmt.Println(data)
	waitKey()
}

func renderDeviceInspector() {
	clearScreen()
	fmt.Println("[Device Inspector]")
	fmt.Println("Enter Device ID:")

	id := readLine()

	data := fetch("/device/status?id=" + id)
	fmt.Println(data)
	waitKey()
}

func renderJobInspector() {
	clearScreen()
	fmt.Println("[Job Inspector]")
	fmt.Println("Enter Job ID:")

	id := readLine()

	data := fetch("/job/status?id=" + id)
	fmt.Println(data)
	waitKey()
}

//
// ─────────────────────────────────────────────────────────────
//   HELPERS
// ─────────────────────────────────────────────────────────────
//

func clearScreen() {
	fmt.Print("\033[2J\033[H")
}

func fetch(path string) map[string]any {
	resp, err := http.Get(os.Getenv("ATONIX_API") + path)
	if err != nil {
		return map[string]any{}
	}
	defer resp.Body.Close()

	var out map[string]any
	json.NewDecoder(resp.Body).Decode(&out)
	return out
}

func readLine() string {
	buf := make([]byte, 100)
	n, _ := os.Stdin.Read(buf)
	return string(buf[:n])
}

func waitKey() {
	fmt.Println("\nPress any key to return...")
	b := make([]byte, 1)
	os.Stdin.Read(b)
}
