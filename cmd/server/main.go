// DevOps Agent - AI-native DevOps unified service
//
// This is the main entry point for the DevOps Agent server.
// It integrates GitHub Webhook handling, Docker deployment verification,
// SQLite persistence, and MCP (Model Context Protocol) interface.
package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hewenyu/Ouroboros/internal/config"
	"github.com/hewenyu/Ouroboros/internal/database"
	"github.com/hewenyu/Ouroboros/internal/docker"
	mcpserver "github.com/hewenyu/Ouroboros/internal/mcp"
	"github.com/hewenyu/Ouroboros/internal/webhook"
	"github.com/mark3labs/mcp-go/server"
)

// Version and build information (injected at build time).
var (
	Version   = "dev"
	BuildTime = "unknown"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func main() {
	// Redirect logs to stderr to avoid interfering with MCP stdio transport
	log.SetOutput(os.Stderr)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	log.Printf("DevOps Agent %s (built: %s)", Version, BuildTime)

	// Load configuration
	cfg := config.Load()

	// Initialize SQLite database with WAL mode
	store, err := database.NewSQLite(cfg.DatabasePath, migrationsFS)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer store.Close()
	log.Println("Database initialized with WAL mode")

	// Initialize Docker manager (optional - may fail if Docker is not available)
	var dockerMgr *docker.Manager
	dockerMgr, err = docker.NewManager()
	if err != nil {
		log.Printf("Warning: Docker not available: %v", err)
		dockerMgr = nil
	} else {
		log.Println("Docker manager initialized")
		defer dockerMgr.Close()
	}

	// Create webhook event processor
	processor := webhook.NewEventProcessor(store, dockerMgr, 100)
	processor.Start(4) // Start 4 worker goroutines
	defer processor.Close()

	// Create webhook handler
	webhookHandler := webhook.NewHandler(cfg.GitHubSecret, processor)

	// Setup HTTP server
	mux := http.NewServeMux()
	mux.Handle("/webhook", webhookHandler)
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/version", versionHandler)

	httpServer := &http.Server{
		Addr:         ":" + cfg.ServerPort,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start HTTP server in a goroutine
	go func() {
		log.Printf("Starting HTTP server on port %s", cfg.ServerPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Create MCP server
	mcpSrv := mcpserver.NewServer(Version, store, dockerMgr)
	log.Println("MCP server initialized")

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	if cfg.MCPEnabled {
		switch cfg.MCPTransport {
		case "stdio":
			// Run MCP server on stdio (blocks main thread)
			log.Println("Starting MCP server on stdio transport")
			go func() {
				<-sigChan
				log.Println("Shutting down...")
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				httpServer.Shutdown(ctx)
				os.Exit(0)
			}()

			if err := server.ServeStdio(mcpSrv.MCPServer()); err != nil {
				log.Fatalf("MCP server error: %v", err)
			}

		case "sse":
			// Run MCP server on SSE transport
			sseServer := server.NewSSEServer(mcpSrv.MCPServer())
			go func() {
				log.Printf("Starting MCP SSE server on port %s", cfg.MCPSSEPort)
				if err := sseServer.Start(":" + cfg.MCPSSEPort); err != nil {
					log.Fatalf("MCP SSE server error: %v", err)
				}
			}()

			// Wait for shutdown signal
			<-sigChan
			log.Println("Shutting down...")
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			httpServer.Shutdown(ctx)

		default:
			log.Fatalf("Unknown MCP transport: %s", cfg.MCPTransport)
		}
	} else {
		// MCP disabled, just run HTTP server
		<-sigChan
		log.Println("Shutting down...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		httpServer.Shutdown(ctx)
	}
}

// healthHandler returns the health status of the server.
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"healthy","version":"%s"}`, Version)
}

// versionHandler returns version information.
func versionHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"version":"%s","build_time":"%s"}`, Version, BuildTime)
}
