//go:build windows || darwin || linux

package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/majbthrd/EOC/app/store"
	"github.com/majbthrd/EOC/app/ui"
	"github.com/majbthrd/EOC/app/version"
)

var (
	wv           = &Webview{}
	uiServerPort int
	appStore     *store.Store
)

var debug = strings.EqualFold(os.Getenv("OLLAMA_DEBUG"), "true") || os.Getenv("OLLAMA_DEBUG") == "1"

var (
	fastStartup = false
	devMode     = false
)

type appMove int

const (
	CannotMove appMove = iota
	UserDeclinedMove
	MoveCompleted
	AlreadyMoved
	LoginSession
	PermissionDenied
	MoveError
)

func main() {
	startHidden := false
	if len(os.Args) > 1 {
		for _, arg := range os.Args {
			switch arg {
			case "serve":
				fmt.Fprintln(os.Stderr, "serve command not supported, use ollama")
				os.Exit(1)
			case "version", "-v", "--version":
				fmt.Println(version.Version)
				os.Exit(0)
			case "--fast-startup":
				// Skip optional steps like pending updates to start quickly for immediate use
				fastStartup = true
			case "-dev", "--dev":
				// Development mode: use local dev server and enable CORS
				devMode = true
			}
		}
	}

	var err error

	logStartup()

	// Check if another instance is already running
	// On Windows, focus the existing instance; on other platforms, kill it
	handleExistingInstance(startHidden)

	// on macOS, offer the user to create a symlink
	// from /usr/local/bin/ollama to the app bundle
	installSymlink()

	var ln net.Listener
	if devMode {
		// Use a fixed port in dev mode for predictable API access
		ln, err = net.Listen("tcp", "127.0.0.1:3001")
	} else {
		ln, err = net.Listen("tcp", "127.0.0.1:0")
	}
	if err != nil {
		slog.Error("failed to find available port", "error", err)
		return
	}

	port := ln.Addr().(*net.TCPAddr).Port
	token := uuid.NewString()
	wv.port = port
	wv.token = token
	uiServerPort = port

	st := &store.Store{}
	appStore = st

	// Enable CORS in development mode
	if devMode {
		os.Setenv("OLLAMA_CORS", "1")

		// Check if Vite dev server is running on port 5173
		var conn net.Conn
		var err error
		for _, addr := range []string{"127.0.0.1:5173", "localhost:5173"} {
			conn, err = net.DialTimeout("tcp", addr, 2*time.Second)
			if err == nil {
				conn.Close()
				break
			}
		}

		if err != nil {
			slog.Error("Vite dev server not running on port 5173")
			fmt.Fprintln(os.Stderr, "Error: Vite dev server is not running on port 5173")
			fmt.Fprintln(os.Stderr, "Please run 'npm run dev' in the ui/app directory to start the UI in development mode")
			os.Exit(1)
		}
	}

	// ctx is the app-level context that will be used to stop the app
	_, cancel := context.WithCancel(context.Background())

	uiServer := ui.Server{
		Token: token,
		Restart: func() { /* nothing to restart */ },
		Store:        st,
		Dev:          devMode,
		Logger:       slog.Default(),
	}

	srv := &http.Server{
		Handler: uiServer.Handler(),
	}

	// Start the UI server
	slog.Info("starting ui server", "port", port)
	go func() {
		slog.Debug("starting ui server on port", "port", port)
		err = srv.Serve(ln)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Warn("desktop server", "error", err)
		}
		slog.Debug("background desktop server done")
	}()

	hasCompletedFirstRun, err := st.HasCompletedFirstRun()
	if err != nil {
		slog.Error("failed to load has completed first run", "error", err)
	}

	if !hasCompletedFirstRun {
		err = st.SetHasCompletedFirstRun(true)
		if err != nil {
			slog.Error("failed to set has completed first run", "error", err)
		}
	}

	// capture SIGINT and SIGTERM signals and gracefully shutdown the app
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-signals
		slog.Info("received SIGINT or SIGTERM signal, shutting down")
		quit()
	}()

	osRun(cancel, hasCompletedFirstRun, startHidden)

	slog.Info("shutting down desktop server")
	if err := srv.Close(); err != nil {
		slog.Warn("error shutting down desktop server", "error", err)
	}

	slog.Info("shutting down ollama server")
	cancel()
}

func startHiddenTasks() {
}

