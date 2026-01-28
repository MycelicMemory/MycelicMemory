package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"
)

var uiPort int

var uiCmd = &cobra.Command{
	Use:   "ui",
	Short: "Launch the MyclicMemory dashboard UI",
	Long: `Opens the MyclicMemory dashboard in your default browser.

The dashboard provides a visual interface for:
  - Viewing memory statistics and system health
  - Browsing and searching memories
  - Editing and deleting memories
  - Filtering by domain and importance

Note: The mycelicmemory server must be running (mycelicmemory start) for the dashboard to work.`,
	Run: func(cmd *cobra.Command, args []string) {
		runDashboard()
	},
}

func init() {
	uiCmd.Flags().IntVarP(&uiPort, "port", "p", 3100, "port for dashboard server")
	rootCmd.AddCommand(uiCmd)
}

func runDashboard() {
	// Check if dashboard files exist
	dashboardPath := findDashboardPath()
	if dashboardPath == "" {
		fmt.Fprintln(os.Stderr, "Dashboard not found. Please install the dashboard first:")
		fmt.Fprintln(os.Stderr, "  npm install -g mycelicmemory-dashboard")
		fmt.Fprintln(os.Stderr, "  or download from: https://github.com/MycelicMemory/mycelicmemory/releases")
		os.Exit(1)
	}

	// Start a simple HTTP server to serve the dashboard
	fmt.Printf("Starting dashboard on http://localhost:%d\n", uiPort)
	fmt.Println("Press Ctrl+C to stop")

	// Open browser
	go openBrowser(fmt.Sprintf("http://localhost:%d", uiPort))

	// Serve static files
	fs := http.FileServer(http.Dir(dashboardPath))

	// Create handler that serves index.html for all routes (SPA support)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Proxy API requests to mycelicmemory server
		if len(r.URL.Path) >= 4 && r.URL.Path[:4] == "/api" {
			proxyToAPI(w, r)
			return
		}

		// Check if file exists
		path := filepath.Join(dashboardPath, r.URL.Path)
		_, err := os.Stat(path)
		if os.IsNotExist(err) && r.URL.Path != "/" {
			// Serve index.html for SPA routes
			http.ServeFile(w, r, filepath.Join(dashboardPath, "index.html"))
			return
		}
		fs.ServeHTTP(w, r)
	})

	if err := http.ListenAndServe(fmt.Sprintf(":%d", uiPort), handler); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting server: %v\n", err)
		os.Exit(1)
	}
}

func findDashboardPath() string {
	// Check common locations for dashboard files
	locations := []string{
		// Development location
		"dashboard/dist",
		"../dashboard/dist",
		// Installed location (npm global)
		filepath.Join(os.Getenv("HOME"), ".mycelicmemory", "dashboard"),
		// Windows AppData
		filepath.Join(os.Getenv("APPDATA"), "mycelicmemory", "dashboard"),
		// Linux/Mac local share
		filepath.Join(os.Getenv("HOME"), ".local", "share", "mycelicmemory", "dashboard"),
	}

	for _, loc := range locations {
		indexPath := filepath.Join(loc, "index.html")
		if _, err := os.Stat(indexPath); err == nil {
			return loc
		}
	}

	return ""
}

func proxyToAPI(w http.ResponseWriter, r *http.Request) {
	// Proxy to mycelicmemory API server on port 3099
	targetURL := fmt.Sprintf("http://localhost:3099%s", r.URL.Path)
	if r.URL.RawQuery != "" {
		targetURL += "?" + r.URL.RawQuery
	}

	proxyReq, err := http.NewRequest(r.Method, targetURL, r.Body)
	if err != nil {
		http.Error(w, "Proxy error", http.StatusInternalServerError)
		return
	}

	// Copy headers
	for key, values := range r.Header {
		for _, value := range values {
			proxyReq.Header.Add(key, value)
		}
	}

	client := &http.Client{}
	resp, err := client.Do(proxyReq)
	if err != nil {
		http.Error(w, "Cannot connect to mycelicmemory server. Is it running? (mycelicmemory start)", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.WriteHeader(resp.StatusCode)

	// Copy response body
	buf := make([]byte, 32*1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			_, _ = w.Write(buf[:n])
		}
		if err != nil {
			break
		}
	}
}

func openBrowser(url string) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default: // linux
		cmd = exec.Command("xdg-open", url)
	}

	_ = cmd.Run()
}
