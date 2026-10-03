//go:build windows || darwin || linux

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/majbthrd/EOC/app/store"
	"github.com/majbthrd/EOC/app/ui"
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

// AppConfig holds user-supplied inference parameters.
// Zero / negative values mean "use the model default".
type AppConfig struct {
	ContextLength int     // num_ctx; 0 = default
	Temperature   float64 // 0 = default (Ollama default is 0.7)
	TopK          int     // 0 = default
	TopP          float64 // 0 = default
}

const (
	CannotMove appMove = iota
	UserDeclinedMove
	MoveCompleted
	AlreadyMoved
	LoginSession
	PermissionDenied
	MoveError
)

const (
	defaultOllamaHost = "127.0.0.1"
	defaultOllamaPort = 11434
)

// parseServerAddr interprets an optional positional argument specifying the
// Ollama server address.  Supported formats:
//   - "192.168.0.1"       → host = "192.168.0.1", port = 11434
//   - "12000"             → host = "127.0.0.1",   port = 12000
//   - "192.168.0.1:12000" → host = "192.168.0.1", port = 12000
//   - "[::1]:11434"       → host = "::1",         port = 11434
//   - "" (empty)          → host = "127.0.0.1",   port = 11434 (defaults)
func parseServerAddr(s string) (host string, port int, err error) {
	// Default values (used when the argument is empty/absent).
	host = defaultOllamaHost
	port = defaultOllamaPort

	if s == "" {
		return host, port, nil
	}

	// Format c: host:port  (includes [IPv6]:port)
	if strings.Contains(s, ":") {
		h, pStr, splitErr := net.SplitHostPort(s)
		if splitErr != nil {
			// Might be a bare IPv6 like "::1" — try parsing as IP directly.
			if ip := net.ParseIP(s); ip != nil {
				host = ip.String()
				return host, port, nil
			}
			return "", 0, fmt.Errorf("invalid server address %q: %w", s, splitErr)
		}
		// Validate host.
		if ip := net.ParseIP(h); ip != nil {
			host = ip.String()
		} else if h == "" {
			// e.g. ":12000" → default host with explicit port.
			host = defaultOllamaHost
		} else {
			// Allow hostnames as a convenience (e.g. "myhost:11434").
			host = h
		}
		// Parse and validate port.
		p, pErr := strconv.Atoi(pStr)
		if pErr != nil || p < 1 || p > 65535 {
			return "", 0, fmt.Errorf("invalid port in %q: must be 1–65535", s)
		}
		port = p
		return host, port, nil
	}

	// Format b: all-digits → port only.
	if allDigits(s) {
		p, pErr := strconv.Atoi(s)
		if pErr != nil || p < 1 || p > 65535 {
			return "", 0, fmt.Errorf("invalid port %q: must be 1–65535", s)
		}
		port = p
		return host, port, nil
	}

	// Format a: IP address only.
	ip := net.ParseIP(s)
	if ip == nil {
		return "", 0, fmt.Errorf("invalid server address %q: not a valid IP, port, or host:port", s)
	}
	host = ip.String()
	return host, port, nil
}

// allDigits reports whether s is non-empty and contains only ASCII digits.
func allDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func main() {
	startHidden := false

	cfg := &AppConfig{}

	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	fs.IntVar(&cfg.ContextLength, "context", 0, "context window size (num_ctx); 0 = model default")
	fs.Float64Var(&cfg.Temperature, "temperature", 0, "sampling temperature; 0 = model default")
	fs.IntVar(&cfg.TopK, "topk", 0, "top-k sampling; 0 = model default")
	fs.Float64Var(&cfg.TopP, "topp", 0, "top-p (nucleus) sampling; 0 = model default")

	fs.Parse(os.Args[1:])

	// Determine the optional positional server-address argument.
	// Convention: place flags before the positional argument, e.g.
	//   ./app -temperature 0.8 192.168.0.1:12000
	ollamaHost := defaultOllamaHost
	ollamaPort := defaultOllamaPort
	if args := fs.Args(); len(args) > 0 {
		var addrErr error
		ollamaHost, ollamaPort, addrErr = parseServerAddr(args[0])
		if addrErr != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", addrErr)
			fs.Usage()
			os.Exit(1)
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
		Token:         token,
		Restart:       func() { /* nothing to restart */ },
		Store:         st,
		Dev:           devMode,
		Logger:        slog.Default(),
		// ── inference params from CLI ─────────────────────────────────
		ContextLength: cfg.ContextLength,
		Temperature:   cfg.Temperature,
		TopK:          cfg.TopK,
		TopP:          cfg.TopP,
		// ── Ollama server address (from optional positional arg) ──────
		OllamaHost:    ollamaHost,
		OllamaPort:    ollamaPort,
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

