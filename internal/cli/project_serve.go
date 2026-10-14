package cli

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"time"

	"github.com/chenhg5/harness-projects/internal/dashboard"
)

func (a app) projectServe(args []string) int {
	fs := newFlags("project serve")
	path := fs.String("path", "", "project directory (required)")
	port := fs.Int("port", 0, "local TCP port (0 chooses a free port)")
	open := fs.Bool("open", false, "open the dashboard in the default browser")
	format, jsonFlag := outputFlags(fs)
	if err := parse(fs, args); err != nil {
		return a.parseError(err)
	}
	if *path == "" {
		return a.parseError(fmt.Errorf("--path is required"))
	}
	if *port < 0 || *port > 65535 {
		return a.parseError(fmt.Errorf("--port must be between 0 and 65535"))
	}
	if _, err := dashboard.Load(*path); err != nil {
		return a.operationError(err, *path)
	}
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", *port))
	if err != nil {
		return a.operationError(err, fmt.Sprintf("127.0.0.1:%d", *port))
	}
	defer listener.Close()
	url := fmt.Sprintf("http://127.0.0.1:%d/", listener.Addr().(*net.TCPAddr).Port)
	a.emit(map[string]any{"action": "project.serve", "path": *path, "url": url, "readOnly": true}, "Dashboard ready at "+url, selectedFormat(*format, *jsonFlag, a.out))
	if *open {
		if err := openBrowser(url); err != nil {
			fmt.Fprintf(a.err, "Could not open browser: %v\n", err)
		}
	}
	server := &http.Server{Handler: dashboard.Handler(*path), ReadHeaderTimeout: 5 * time.Second}
	if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return a.operationError(err, url)
	}
	return 0
}

func openBrowser(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	default:
		return exec.Command("xdg-open", url).Start()
	}
}
