package main

import (
	"context"
	"crypto/tls"
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"

	"github.com/chromedp/chromedp"
	"github.com/gorilla/mux"
	"github.com/mitchellh/go-homedir"
	"github.com/peterbourgon/ff/v3/ffcli"
	"go.etcd.io/bbolt"
	"go.uber.org/zap"

	"github.com/utkarshrai2811/proxima/pkg/api"
	"github.com/utkarshrai2811/proxima/pkg/chrome"
	"github.com/utkarshrai2811/proxima/pkg/config"
	"github.com/utkarshrai2811/proxima/pkg/db/bolt"
	"github.com/utkarshrai2811/proxima/pkg/proj"
	"github.com/utkarshrai2811/proxima/pkg/proxy"
	"github.com/utkarshrai2811/proxima/pkg/proxy/intercept"
	"github.com/utkarshrai2811/proxima/pkg/reqlog"
	"github.com/utkarshrai2811/proxima/pkg/scope"
	"github.com/utkarshrai2811/proxima/pkg/sender"
	"github.com/utkarshrai2811/proxima/pkg/ws"
)

var version = "0.0.0"

//go:embed all:admin/dist
var adminContent embed.FS

var proximaUsage = `
Usage:
    proxima [flags] [subcommand] [flags]

Runs an HTTP server with (MITM) proxy, GraphQL service, and a web based admin interface.

Options:
    --cert         Path to root CA certificate. Creates file if it doesn't exist. (Default: <data dir>/proxima_cert.pem)
    --key          Path to root CA private key. Creates file if it doesn't exist. (Default: <data dir>/proxima_key.pem)
    --db           Database file path. Creates file if it doesn't exist. (Default: <data dir>/proxima.db)
    --addr         TCP address for HTTP server to listen on, in the form \"host:port\". (Default: ":8080")
    --allowed-hosts  Comma-separated allowed Host header values for the admin UI, to prevent
                     DNS rebinding. The proxy endpoint is never gated. (Default: "localhost,127.0.0.1")
    --max-body-size  Maximum request/response body size stored in the log, e.g. "10MB", "1GB".
                     The full body is still forwarded. Use 0 for no limit. (Default: "10MB")
    --api-key      API key for the admin UI/API. Empty disables auth (default). Use "auto" to
                   generate a random key, printed on startup. The proxy endpoint is never gated.
    --chrome       Launch Chrome with proxy settings applied and certificate errors ignored. (Default: false)
    --verbose      Enable verbose logging.
    --json         Encode logs as JSON, instead of pretty/human readable output.
    --version, -v  Output version.
    --help, -h     Output this usage text.

The data directory is platform-specific:
    macOS:   ~/Library/Application Support/proxima
    Linux:   $XDG_CONFIG_HOME/proxima (or ~/.config/proxima)
    Windows: %APPDATA%\proxima

Subcommands:
    - cert  Certificate management

Run ` + "`proxima <subcommand> --help`" + ` for subcommand specific usage instructions.

Visit https://github.com/utkarshrai2811/proxima to learn more about Proxima.
`

type ProximaCommand struct {
	config *Config

	cert         string
	key          string
	db           string
	addr         string
	allowedHosts string
	maxBodySize  string
	apiKey       string
	chrome       bool
	version      bool
}

func NewProximaCommand() (*ffcli.Command, *Config) {
	cmd := ProximaCommand{
		config: &Config{},
	}

	fs := flag.NewFlagSet("proxima", flag.ExitOnError)

	fs.StringVar(&cmd.cert, "cert", config.CertPath(),
		"Path to root CA certificate. Creates a new certificate if file doesn't exist.")
	fs.StringVar(&cmd.key, "key", config.KeyPath(),
		"Path to root CA private key. Creates a new private key if file doesn't exist.")
	fs.StringVar(&cmd.db, "db", config.DBPath(), "Database file path. Creates file if it doesn't exist.")
	fs.StringVar(&cmd.addr, "addr", ":8080", "TCP address to listen on, in the form \"host:port\".")
	fs.StringVar(&cmd.allowedHosts, "allowed-hosts", "localhost,127.0.0.1",
		"Comma-separated list of allowed Host header values for the admin UI. "+
			"Prevents DNS rebinding attacks. The proxy traffic endpoint is never gated.")
	fs.StringVar(&cmd.maxBodySize, "max-body-size", "10MB",
		"Maximum request/response body size to store in the log (e.g. 10MB, 1GB). "+
			"The full body is still forwarded. Use 0 for no limit.")
	fs.StringVar(&cmd.apiKey, "api-key", "",
		"API key for admin UI/API authentication. Empty disables auth (default). "+
			"Set to \"auto\" to generate a random key, printed on startup. "+
			"The proxy traffic endpoint is never gated.")
	fs.BoolVar(&cmd.chrome, "chrome", false, "Launch Chrome with proxy settings applied and certificate errors ignored.")
	fs.BoolVar(&cmd.version, "version", false, "Output version.")
	fs.BoolVar(&cmd.version, "v", false, "Output version.")

	cmd.config.RegisterFlags(fs)

	return &ffcli.Command{
		Name:    "proxima",
		FlagSet: fs,
		Subcommands: []*ffcli.Command{
			NewCertCommand(cmd.config),
		},
		Exec: cmd.Exec,
		UsageFunc: func(*ffcli.Command) string {
			return proximaUsage
		},
	}, cmd.config
}

func (cmd *ProximaCommand) Exec(ctx context.Context, _ []string) error {
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()

	if cmd.version {
		fmt.Fprint(os.Stdout, version+"\n")
		return nil
	}

	mainLogger := cmd.config.logger.Named("main")

	maxBodyBytes, err := proxy.ParseBodySize(cmd.maxBodySize)
	if err != nil {
		mainLogger.Fatal("Invalid --max-body-size value.", zap.Error(err))
	}

	// Resolve "auto" into a freshly generated API key, printed once on startup.
	if cmd.apiKey == "auto" {
		key, err := api.GenerateAPIKey()
		if err != nil {
			mainLogger.Fatal("Failed to generate API key.", zap.Error(err))
		}

		cmd.apiKey = key

		fmt.Println()
		fmt.Println("  ──────────────────────────────────────────────────────────────────────")
		fmt.Println("   Proxima API key (store it now — it is not saved and shown only once):")
		fmt.Println()
		fmt.Printf("     %s\n", key)
		fmt.Println()
		fmt.Println("   Send it via 'X-Proxima-Api-Key: <key>', 'Authorization: Bearer <key>',")
		fmt.Println("   or sign in at /login.")
		fmt.Println("  ──────────────────────────────────────────────────────────────────────")
		fmt.Println()
	}

	listenHost, listenPort, err := net.SplitHostPort(cmd.addr)
	if err != nil {
		mainLogger.Fatal("Failed to parse listening address.", zap.Error(err))
	}

	url := fmt.Sprintf("http://%v:%v", listenHost, listenPort)
	if listenHost == "" || listenHost == "0.0.0.0" || listenHost == "127.0.0.1" || listenHost == "::1" {
		url = fmt.Sprintf("http://localhost:%v", listenPort)
	}

	// Expand `~` in filepaths.
	caCertFile, err := homedir.Expand(cmd.cert)
	if err != nil {
		cmd.config.logger.Fatal("Failed to parse CA certificate filepath.", zap.Error(err))
	}

	caKeyFile, err := homedir.Expand(cmd.key)
	if err != nil {
		cmd.config.logger.Fatal("Failed to parse CA private key filepath.", zap.Error(err))
	}

	dbPath, err := homedir.Expand(cmd.db)
	if err != nil {
		cmd.config.logger.Fatal("Failed to parse database path.", zap.Error(err))
	}

	// Load existing CA certificate and key from disk, or generate and write
	// to disk if no files exist yet.
	caCert, caKey, err := proxy.LoadOrCreateCA(caKeyFile, caCertFile)
	if err != nil {
		cmd.config.logger.Fatal("Failed to load or create CA key pair.", zap.Error(err))
	}

	dbLogger := cmd.config.logger.Named("boltdb").Sugar()
	boltOpts := *bbolt.DefaultOptions
	boltOpts.Logger = &bolt.Logger{SugaredLogger: dbLogger}

	boltDB, err := bolt.OpenDatabase(dbPath, &boltOpts)
	if err != nil {
		cmd.config.logger.Fatal("Failed to open database.", zap.Error(err))
	}
	defer boltDB.Close()

	scope := &scope.Scope{}

	reqLogService := reqlog.NewService(reqlog.Config{
		Scope:       scope,
		Repository:  boltDB,
		Logger:      cmd.config.logger.Named("reqlog").Sugar(),
		MaxBodySize: maxBodyBytes,
	})

	interceptService := intercept.NewService(intercept.Config{
		Logger: cmd.config.logger.Named("intercept").Sugar(),
	})

	senderService := sender.NewService(sender.Config{
		Repository:    boltDB,
		ReqLogService: reqLogService,
	})

	projService, err := proj.NewService(proj.Config{
		Repository:       boltDB,
		InterceptService: interceptService,
		ReqLogService:    reqLogService,
		SenderService:    senderService,
		Scope:            scope,
	})
	if err != nil {
		cmd.config.logger.Fatal("Failed to create new projects service.", zap.Error(err))
	}

	wsStore := ws.NewStore()

	proxy, err := proxy.NewProxy(proxy.Config{
		CACert:  caCert,
		CAKey:   caKey,
		Logger:  cmd.config.logger.Named("proxy").Sugar(),
		WSStore: wsStore,
	})
	if err != nil {
		cmd.config.logger.Fatal("Failed to create new proxy.", zap.Error(err))
	}

	proxy.UseRequestModifier(reqLogService.RequestModifier)
	proxy.UseResponseModifier(reqLogService.ResponseModifier)
	proxy.UseRequestModifier(interceptService.RequestModifier)
	proxy.UseResponseModifier(interceptService.ResponseModifier)

	fsSub, err := fs.Sub(adminContent, "admin/dist")
	if err != nil {
		cmd.config.logger.Fatal("Failed to construct file system subtree from admin dir.", zap.Error(err))
	}

	adminHandler := spaFileServer(fsSub)
	router := mux.NewRouter().SkipClean(true)
	adminRouter := router.MatcherFunc(func(req *http.Request, match *mux.RouteMatch) bool {
		hostname, _ := os.Hostname()
		host, _, _ := net.SplitHostPort(req.Host)

		// Serve local admin routes when either:
		// - The `Host` is well-known, e.g. `proxima.proxy`, `localhost:[port]`
		//   or the listen addr `[host]:[port]`.
		// - The request is not for TLS proxying (e.g. no `CONNECT`) and not
		//   for proxying an external URL. E.g. Request-Line (RFC 7230, Section 3.1.1)
		//   has no scheme.
		return strings.EqualFold(host, hostname) ||
			req.Host == "proxima.proxy" ||
			req.Host == fmt.Sprintf("%v:%v", "localhost", listenPort) ||
			req.Host == fmt.Sprintf("%v:%v", listenHost, listenPort) ||
			req.Method != http.MethodConnect && !strings.HasPrefix(req.RequestURI, "http://")
	}).Subrouter().StrictSlash(true)

	// Gate the admin UI and API behind a Host header allowlist to prevent DNS
	// rebinding attacks. The proxy fallback handler below is never gated.
	allowedHosts := strings.Split(cmd.allowedHosts, ",")
	for i := range allowedHosts {
		allowedHosts[i] = strings.TrimSpace(allowedHosts[i])
	}

	adminRouter.Use(func(next http.Handler) http.Handler {
		return api.HostAllowlistMiddleware(allowedHosts, next)
	})

	// Require an API key for the admin UI and API when one is configured.
	// /login, /api/auth/login, and /health are exempt (see AuthMiddleware).
	adminRouter.Use(func(next http.Handler) http.Handler {
		return api.AuthMiddleware(cmd.apiKey, next)
	})

	// Health endpoint for monitoring (exempt from auth).
	adminRouter.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// Login routes are only meaningful when auth is enabled.
	if cmd.apiKey != "" {
		adminRouter.Path("/login").Handler(api.LoginPageHandler())
		adminRouter.Path("/api/auth/login").Handler(api.LoginHandler(cmd.apiKey))
	}

	// GraphQL server.
	gqlEndpoint := "/api/graphql/"
	adminRouter.Path(gqlEndpoint).Handler(api.HTTPHandler(&api.Resolver{
		ProjectService:    projService,
		RequestLogService: reqLogService,
		InterceptService:  interceptService,
		SenderService:     senderService,
	}, gqlEndpoint))

	// CA certificate download for the Settings page. The CA cert is meant to be
	// installed in trust stores, so serving it (gated by auth) is safe.
	adminRouter.HandleFunc("/api/cert/download", func(w http.ResponseWriter, _ *http.Request) {
		caCertFile, err := homedir.Expand(cmd.cert)
		if err != nil {
			http.Error(w, "could not resolve certificate path", http.StatusInternalServerError)

			return
		}

		data, err := os.ReadFile(caCertFile)
		if err != nil {
			http.Error(w, "certificate not available", http.StatusNotFound)

			return
		}

		w.Header().Set("Content-Type", "application/x-pem-file")
		w.Header().Set("Content-Disposition", `attachment; filename="proxima_cert.pem"`)
		_, _ = w.Write(data)
	})

	// Read-only server configuration for the Settings page.
	adminRouter.HandleFunc("/api/config", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"listenAddr":   cmd.addr,
			"maxBodySize":  cmd.maxBodySize,
			"allowedHosts": cmd.allowedHosts,
			"authEnabled":  cmd.apiKey != "",
		})
	})

	// WebSocket session REST + SSE API.
	adminRouter.PathPrefix("/api/ws").Handler(ws.Handler(wsStore))

	// Admin interface (single-page app with client-side routing).
	adminRouter.PathPrefix("").Handler(adminHandler)

	// Fallback (default) is the Proxy handler.
	router.PathPrefix("").Handler(proxy)

	httpServer := &http.Server{
		Addr:         cmd.addr,
		Handler:      router,
		TLSNextProto: map[string]func(*http.Server, *tls.Conn, http.Handler){}, // Disable HTTP/2
		ErrorLog:     zap.NewStdLog(cmd.config.logger.Named("http")),
	}

	go func() {
		mainLogger.Info(fmt.Sprintf("Proxima (v%v) is running on %v ...", version, cmd.addr))
		mainLogger.Info(fmt.Sprintf("\x1b[%dm%s\x1b[0m", uint8(32), "Get started at "+url))

		err := httpServer.ListenAndServe()
		if err != http.ErrServerClosed {
			mainLogger.Fatal("HTTP server closed unexpected.", zap.Error(err))
		}
	}()

	if cmd.chrome {
		ctx, cancel := chrome.NewExecAllocator(ctx, chrome.Config{
			ProxyServer:      url,
			ProxyBypassHosts: []string{url},
		})
		defer cancel()

		taskCtx, cancel := chromedp.NewContext(ctx)
		defer cancel()

		err = chromedp.Run(taskCtx, chromedp.Navigate(url))

		switch {
		case errors.Is(err, exec.ErrNotFound):
			mainLogger.Info("Chrome executable not found.")
		case err != nil:
			mainLogger.Error(fmt.Sprintf("Failed to navigate to %v.", url), zap.Error(err))
		default:
			mainLogger.Info("Launched Chrome.")
		}
	}

	// Wait for interrupt signal.
	<-ctx.Done()
	// Restore signal, allowing "force quit".
	stop()

	mainLogger.Info("Shutting down HTTP server. Press Ctrl+C to force quit.")

	// Note: We expect httpServer.Handler to handle timeouts, thus, we don't
	// need a context value with deadline here.
	//nolint:contextcheck
	err = httpServer.Shutdown(context.Background())
	if err != nil {
		return fmt.Errorf("failed to shutdown HTTP server: %w", err)
	}

	return nil
}

// spaFileServer serves the embedded single-page app. Real files (index.html,
// /assets/*) are served as-is; any other path falls back to index.html so the
// client-side router handles deep links and hard refreshes (e.g. /intercept).
func spaFileServer(fsys fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(fsys))
	index, _ := fs.ReadFile(fsys, "index.html")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/")
		if name == "" {
			fileServer.ServeHTTP(w, r)

			return
		}

		if f, err := fsys.Open(name); err == nil {
			_ = f.Close()
			fileServer.ServeHTTP(w, r)

			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(index)
	})
}
