package main

import (
    "context"
    "fmt"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"
)

// Config holds application configuration
type Config struct {
    Port     string
    Hostname string
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

// handleHealth handles health check requests
func handleHealth(logger *Logger) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Add actual health check logic here if needed
        w.WriteHeader(http.StatusOK)
        fmt.Fprint(w, "OK")
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
    mux.HandleFunc("/health", handleHealth(logger))

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
