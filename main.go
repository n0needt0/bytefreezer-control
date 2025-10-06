package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/n0needt0/bytefreezer-control/api"
	"github.com/n0needt0/bytefreezer-control/config"
	"github.com/n0needt0/bytefreezer-control/services"
	"github.com/n0needt0/go-goodies/log"
)

var (
	version   = "dev"
	buildTime = "unknown"
	gitCommit = "unknown"
	conf      = config.Config{}
	envPrefix = "BYTEFREEZER_CONTROL_"
)

func main() {
	var (
		cfgFilePath    = flag.String("config", "config.yaml", "Path to configuration file")
		validateConfig = flag.Bool("validate-config", false, "Validate configuration and exit")
		showVersion    = flag.Bool("version", false, "Show version and exit")
		showHelp       = flag.Bool("help", false, "Show help and exit")
	)

	flag.Parse()

	// Handle version flag
	if *showVersion {
		fmt.Printf("bytefreezer-control version %s (built %s, commit %s)\n", version, buildTime, gitCommit)
		os.Exit(0)
	}

	// Handle help flag
	if *showHelp {
		log.Info("ByteFreezer Control Service - Ecosystem management and control plane")
		flag.PrintDefaults()
		os.Exit(0)
	}

	err := config.LoadConfig(*cfgFilePath, envPrefix, &conf)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Handle config validation
	if *validateConfig {
		log.Infof("Configuration validation successful: %s", *cfgFilePath)
		os.Exit(0)
	}

	setLogLevel(conf.Logging.Level)

	log.Infof("Starting %s version %s", conf.App.Name, conf.App.Version)

	// Initialize OpenTelemetry provider if enabled
	var otelShutdown func()
	if conf.Otel.Enabled {
		otelShutdown = InitOtelProvider(&conf)
		defer func() {
			if otelShutdown != nil {
				log.Info("Shutting down OpenTelemetry provider")
				otelShutdown()
			}
		}()
		log.Info("OpenTelemetry provider initialized")
	}

	// Initialize all components
	log.Info("Initializing service components...")
	if err := conf.InitializeComponents(); err != nil {
		log.Fatalf("Failed to initialize service components: %v", err)
	}

	// Initialize services
	servicesInstance := services.NewServices(&conf)

	// Create and start server
	server := NewServer(servicesInstance, &conf)
	server.HttpApi = api.NewAPI(servicesInstance, &conf)

	// Start housekeeping if enabled
	if conf.Housekeeping.Enabled {
		go server.startHousekeeping()
	}

	// Start API server in background
	go server.HttpApi.Serve(":"+strconv.Itoa(conf.Server.ApiPort), server.HttpApi.NewRouter())

	// Setup signal handling
	signalC := make(chan os.Signal, 1)
	signal.Notify(signalC, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGABRT, syscall.SIGTERM)

	log.Info("ByteFreezer Control service is running. Press Ctrl+C to stop.")

	// Wait for shutdown signal
	sig := <-signalC
	log.Debugf("Received signal %v", sig)

	if err := server.Stop(2 * time.Second); err != nil {
		log.Fatalf("Error stopping service: %v", err)
	}

	log.Info("ByteFreezer Control service stopped")
}

func setLogLevel(levelStr string) {
	switch strings.ToLower(levelStr) {
	case "debug":
		log.SetMinLogLevel(log.MinLevelDebug)
	case "info":
		log.SetMinLogLevel(log.MinLevelInfo)
	case "warn":
		log.SetMinLogLevel(log.MinLevelWarn)
	case "error":
		log.SetMinLogLevel(log.MinLevelError)
	}
}

// Server provides basic service functions and state
type Server struct {
	Config   *config.Config
	Name     string
	HttpApi  *api.API
	Services *services.Services
}

func NewServer(services *services.Services, conf *config.Config) *Server {
	return &Server{
		Config:   conf,
		Name:     conf.App.Name,
		Services: services,
	}
}

func (svc *Server) Stop(timeout time.Duration) error {
	log.Debug("Stopping Control service...")

	if svc.HttpApi != nil {
		svc.HttpApi.Stop()
	}

	if svc.Services.Database != nil {
		svc.Services.Database.Close()
	}

	log.Info("Control service stopped gracefully")
	return nil
}

func (svc *Server) startHousekeeping() {
	ticker := time.NewTicker(time.Duration(svc.Config.Housekeeping.IntervalSeconds) * time.Second)
	defer ticker.Stop()

	log.Infof("Starting housekeeping with interval %d seconds", svc.Config.Housekeeping.IntervalSeconds)

	for {
		select {
		case <-ticker.C:
			svc.runHousekeeping()
		}
	}
}

func (svc *Server) runHousekeeping() {
	log.Debug("Running housekeeping tasks...")

	// Perform database maintenance
	if svc.Services.Database != nil {
		// Add database maintenance tasks here
		log.Debug("Database maintenance completed")
	}

	log.Debug("Housekeeping tasks completed")
}
