package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jgulick48/lxp-bridge-go/internal/coordinator"
	"github.com/jgulick48/lxp-bridge-go/internal/models"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// Set via -ldflags "-X main.version=... -X main.commit=...".
var (
	version = "dev"
	commit  = "none"
)

var (
	configLocation = flag.String("configFile", "./config.yml", "Location for the configuration file.")
	logLevelFlag   = flag.String("logLevel", "", "Log level (debug, info, warn, error). Overrides the loglevel config key.")
	healthAddr     = flag.String("healthAddr", "0.0.0.0:8080", "Listen address for the /health endpoint.")
)

func main() {
	flag.Parse()
	os.Exit(run())
}

func run() int {
	config, err := LoadClientConfig(*configLocation)
	if err != nil {
		logrus.Errorf("Unable to load config: %s", err)
		return 1
	}
	level := config.LogLevel
	if *logLevelFlag != "" {
		level = *logLevelFlag
	}
	if err := setLogLevel(level); err != nil {
		logrus.Warnf("%s; using info", err)
	}
	logrus.Infof("lxp-bridge-go version=%s commit=%s config=%s", version, commit, *configLocation)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	coord, err := coordinator.New(config)
	if err != nil {
		logrus.Errorf("Unable to start: %s", err)
		return 1
	}
	coord.Start(ctx)

	mux := http.NewServeMux()
	mux.Handle("/health", coord.HealthHandler())
	server := &http.Server{Addr: *healthAddr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	serverErr := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	exit := 0
	select {
	case <-ctx.Done():
		logrus.Info("Shutdown signal received")
	case err := <-serverErr:
		logrus.Errorf("Health server failed: %s", err)
		exit = 1
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logrus.Warnf("Error shutting down health server: %s", err)
	}
	coord.Close()
	logrus.Info("Shutdown complete")
	return exit
}

func setLogLevel(level string) error {
	if level == "" {
		level = "info"
	}
	switch strings.ToLower(level) {
	case "debug", "info", "warn", "error":
	default:
		logrus.SetLevel(logrus.InfoLevel)
		return fmt.Errorf("invalid log level %q (expected debug, info, warn or error)", level)
	}
	parsed, _ := logrus.ParseLevel(strings.ToLower(level))
	logrus.SetLevel(parsed)
	return nil
}

func LoadClientConfig(filename string) (models.Config, error) {
	if filename == "" {
		filename = "./config.yml"
	}
	var config models.Config
	configFile, err := os.ReadFile(filename)
	if err != nil {
		return config, fmt.Errorf("reading config file %s: %w", filename, err)
	}
	if err := yaml.Unmarshal(configFile, &config); err != nil {
		return config, fmt.Errorf("invalid config file %s: %w", filename, err)
	}
	return config, nil
}
