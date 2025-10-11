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

	// Register self in health system if enabled
	if conf.HealthReporting.Enabled && servicesInstance.HealthService != nil {
		server.registerSelfInHealthSystem()
	}

	// Start health polling if health service is available
	if servicesInstance.HealthService != nil {
		go server.startHealthPolling()
	}

	// Start periodic self-health updates if enabled
	if conf.HealthReporting.Enabled && servicesInstance.HealthService != nil {
		go server.startSelfHealthUpdates()
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

	// Perform health service cleanup
	if svc.Services.HealthService != nil {
		if err := svc.Services.HealthService.CleanupStaleRecords(); err != nil {
			log.Warnf("Failed to cleanup stale health records: %v", err)
		}
	}

	log.Debug("Housekeeping tasks completed")
}

func (svc *Server) startHealthPolling() {
	// Poll every 10 minutes as per requirements
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	log.Info("Starting health polling with 10-minute interval")

	// Initial poll on startup
	svc.runHealthPolling()

	for {
		select {
		case <-ticker.C:
			svc.runHealthPolling()
		}
	}
}

func (svc *Server) runHealthPolling() {
	log.Debug("Running health polling...")

	if svc.Services.HealthService != nil {
		if err := svc.Services.HealthService.PollServicesHealth(); err != nil {
			log.Warnf("Health polling failed: %v", err)
		} else {
			log.Debug("Health polling completed successfully")
		}
	}
}

// buildControlConfiguration creates the control service configuration for health reporting
func (svc *Server) buildControlConfiguration() map[string]interface{} {
	// Get actual hostname
	hostname, err := os.Hostname()
	if err != nil {
		log.Warnf("Failed to get hostname, using 'localhost': %v", err)
		hostname = "localhost"
	}

	// Create instance API URL (without protocol)
	instanceAPI := fmt.Sprintf("%s:%d", hostname, svc.Config.Server.ApiPort)

	return map[string]interface{}{
		"service_type":    "bytefreezer-control",
		"version":         svc.Config.App.Version,
		"instance_api":    instanceAPI,
		"report_interval": svc.Config.HealthReporting.ReportInterval,
		"timeout":         fmt.Sprintf("%ds", svc.Config.HealthReporting.TimeoutSeconds),
		"api": map[string]interface{}{
			"port":         svc.Config.Server.ApiPort,
			"auth_enabled": svc.Config.Auth.Enabled,
		},
		"database": map[string]interface{}{
			"enabled":  svc.Config.Database.Enabled,
			"type":     svc.Config.Database.Type,
			"host":     svc.Config.Database.Host,
			"port":     svc.Config.Database.Port,
			"database": svc.Config.Database.Database,
		},
		"housekeeping": map[string]interface{}{
			"enabled":          svc.Config.Housekeeping.Enabled,
			"interval_seconds": svc.Config.Housekeeping.IntervalSeconds,
		},
		"otel": map[string]interface{}{
			"enabled":      svc.Config.Otel.Enabled,
			"metrics_port": svc.Config.Otel.MetricsPort,
		},
		"capabilities": []string{
			"health_monitoring",
			"service_registration",
			"tenant_management",
			"configuration_management",
			"api_gateway",
			"authentication",
			"metrics_collection",
		},
	}
}

func (svc *Server) registerSelfInHealthSystem() {
	if !svc.Config.HealthReporting.RegisterOnStartup {
		return
	}

	// Get actual hostname
	hostname, err := os.Hostname()
	if err != nil {
		log.Warnf("Failed to get hostname, using 'localhost': %v", err)
		hostname = "localhost"
	}

	// Create instance API URL (without protocol)
	instanceAPI := fmt.Sprintf("%s:%d", hostname, svc.Config.Server.ApiPort)

	// Build configuration
	configuration := svc.buildControlConfiguration()

	if err := svc.Services.HealthService.RegisterSelf("bytefreezer-control", instanceAPI, configuration); err != nil {
		log.Warnf("Failed to register control service in health system: %v", err)
	} else {
		log.Info("Successfully registered control service in health system")
	}
}

func (svc *Server) startSelfHealthUpdates() {
	// Parse report interval
	reportInterval := time.Duration(svc.Config.HealthReporting.ReportInterval) * time.Second
	if reportInterval <= 0 {
		log.Warnf("Invalid health reporting interval %d, using default 30s", svc.Config.HealthReporting.ReportInterval)
		reportInterval = 30 * time.Second
	}

	ticker := time.NewTicker(reportInterval)
	defer ticker.Stop()

	log.Infof("Starting self-health updates with interval %v", reportInterval)

	for {
		select {
		case <-ticker.C:
			svc.updateSelfHealth()
		}
	}
}

func (svc *Server) updateSelfHealth() {
	// Get hostname and instance API
	hostname, err := os.Hostname()
	if err != nil {
		log.Warnf("Failed to get hostname: %v", err)
		hostname = "localhost"
	}
	instanceAPI := fmt.Sprintf("%s:%d", hostname, svc.Config.Server.ApiPort)

	// Build configuration - updated on every health report
	configuration := svc.buildControlConfiguration()

	// Generate control service metrics
	metrics := map[string]interface{}{
		"timestamp":         time.Now().Unix(),
		"service_type":      "bytefreezer-control",
		"uptime_seconds":    time.Since(time.Now().Add(-time.Hour)).Seconds(), // Placeholder - would need actual start time
		"version":           svc.Config.App.Version,
		"last_health_check": time.Now().UTC().Format(time.RFC3339),
		"api": map[string]interface{}{
			"port":         svc.Config.Server.ApiPort,
			"auth_enabled": svc.Config.Auth.Enabled,
		},
		"database": map[string]interface{}{
			"connected": svc.Services.Database != nil,
		},
		"services": map[string]interface{}{
			"health_service_enabled":   svc.Services.HealthService != nil,
			"database_service_enabled": svc.Services.Database != nil,
		},
	}

	if err := svc.Services.HealthService.UpdateSelfHealth("bytefreezer-control", instanceAPI, configuration, metrics); err != nil {
		log.Debugf("Failed to update self health: %v", err)
	} else {
		log.Debug("Successfully updated self health status")
	}
}
