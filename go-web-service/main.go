package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
    "os/signal"
    "sync"
    "syscall"
    "time"
)

// Config holds application configuration
type Config struct {
    Port     string
    Hostname string
}

// HealthStatus holds the application health status
type HealthStatus struct {
    ready bool
    mu    sync.RWMutex
}

// SetReady marks the service as ready
func (h *HealthStatus) SetReady(ready bool) {
    h.mu.Lock()
    defer h.mu.Unlock()
    h.ready = ready
}

// IsReady returns whether the service is ready
func (h *HealthStatus) IsReady() bool {
    h.mu.RLock()
    defer h.mu.RUnlock()
    return h.ready
}

// loadConfig loads configuration from environment variables with defaults
func loadConfig() (*Config, error) {
    hostname, err := os.Hostname()
    if err != nil {
        log.Printf("Warning: unable to get hostname, using default: %v", err)
        hostname = "unknown"
    }

    port := os.Getenv("PORT")
    if port == "" {
        port = "1337"
    }

    return &Config{
        Port:     port,
        Hostname: hostname,
    }, nil
}

// Logger wraps standard logger with structured output
type Logger struct {
    prefix string
}

func NewLogger(prefix string) *Logger {
    return &Logger{prefix: prefix}
}

func (l *Logger) Info(format string, v ...interface{}) {
    log.Printf("[INFO] [%s] "+format, append([]interface{}{l.prefix}, v...)...)
}

func (l *Logger) Error(format string, v ...interface{}) {
    log.Printf("[ERROR] [%s] "+format, append([]interface{}{l.prefix}, v...)...)
}

// handleRoot handles requests to the root endpoint
func handleRoot(config *Config, logger *Logger) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        logger.Info("Request: %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
        fmt.Fprintf(w, "Hello, World! From %s", config.Hostname)
    }
}

// handleHealth handles health check requests (combined liveness + readiness)
func handleHealth(logger *Logger, health *HealthStatus) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        if !health.IsReady() {
            w.WriteHeader(http.StatusServiceUnavailable)
            fmt.Fprint(w, "Service not ready")
            return
        }
        w.WriteHeader(http.StatusOK)
        fmt.Fprint(w, "OK")
    }
}

// handleLive handles liveness probe (is the service running?)
func handleLive(logger *Logger) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Liveness check: service is alive if it can respond
        w.WriteHeader(http.StatusOK)
        fmt.Fprint(w, "alive")
    }
}

// handleReady handles readiness probe (is the service ready to accept traffic?)
func handleReady(logger *Logger, health *HealthStatus, config *Config) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        if !health.IsReady() {
            w.WriteHeader(http.StatusServiceUnavailable)
            response := map[string]interface{}{
                "status":   "not_ready",
                "hostname": config.Hostname,
                "message":  "Service is starting up",
            }
            w.Header().Set("Content-Type", "application/json")
            json.NewEncoder(w).Encode(response)
            return
        }

        // Perform actual readiness checks here
        // For example: check database connectivity, dependent services, etc.
        ready := true
        checks := make(map[string]string)

        // Example check: verify service is fully initialized
        checks["service"] = "ok"

        // Add more checks as needed:
        // checks["database"] = checkDatabase()
        // checks["cache"] = checkCache()

        if !ready {
            w.WriteHeader(http.StatusServiceUnavailable)
        } else {
            w.WriteHeader(http.StatusOK)
        }

        response := map[string]interface{}{
            "status":   "ready",
            "hostname": config.Hostname,
            "checks":   checks,
        }
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(response)
    }
}

// loggingMiddleware logs all incoming requests
func loggingMiddleware(logger *Logger, next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        next.ServeHTTP(w, r)
        logger.Info("%s %s - %v", r.Method, r.URL.Path, time.Since(start))
    })
}

func main() {
    logger := NewLogger("web-service")
    logger.Info("Starting application...")

    // Initialize health status
    health := &HealthStatus{ready: false}

    // Load configuration
    config, err := loadConfig()
    if err != nil {
        logger.Error("Failed to load configuration: %v", err)
        os.Exit(1)
    }

    logger.Info("Configuration loaded - Port: %s, Hostname: %s", config.Port, config.Hostname)

    // Setup HTTP handlers
    mux := http.NewServeMux()
    mux.HandleFunc("/", handleRoot(config, logger))
    mux.HandleFunc("/health", handleHealth(logger, health))
    mux.HandleFunc("/live", handleLive(logger))
    mux.HandleFunc("/ready", handleReady(logger, health, config))

    // Wrap with middleware
    handler := loggingMiddleware(logger, mux)

    // Create HTTP server
    server := &http.Server{
        Addr:         ":" + config.Port,
        Handler:      handler,
        ReadTimeout:  15 * time.Second,
        WriteTimeout: 15 * time.Second,
        IdleTimeout:  60 * time.Second,
    }

    // Channel to listen for errors coming from the server
    serverErrors := make(chan error, 1)

    // Start the server
    go func() {
        logger.Info("Server listening on port %s", config.Port)
        serverErrors <- server.ListenAndServe()
    }()

    // Wait a moment for server to start, then mark as ready
    go func() {
        time.Sleep(100 * time.Millisecond)
        health.SetReady(true)
        logger.Info("Service marked as ready")
    }()

    // Channel to listen for interrupt or terminate signals
    shutdown := make(chan os.Signal, 1)
    signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

    // Block until we receive a signal or error
    select {
    case err := <-serverErrors:
        logger.Error("Server error: %v", err)
        os.Exit(1)

    case sig := <-shutdown:
        logger.Info("Shutdown signal received: %v", sig)

        // Mark service as not ready to stop receiving new traffic
        health.SetReady(false)
        logger.Info("Service marked as not ready")

        // Give outstanding requests a deadline for completion
        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()

        // Asking server to gracefully shutdown
        if err := server.Shutdown(ctx); err != nil {
            logger.Error("Graceful shutdown failed: %v", err)
            if err := server.Close(); err != nil {
                logger.Error("Could not stop server gracefully: %v", err)
            }
            os.Exit(1)
        }

        logger.Info("Server stopped gracefully")
    }
}
