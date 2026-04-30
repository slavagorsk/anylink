package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/bjdgyc/anylink/base"
	"github.com/bjdgyc/anylink/dbdata"
	"github.com/bjdgyc/anylink/handler"
	"github.com/bjdgyc/anylink/server"
)

var (
	// Version is set at build time via ldflags
	Version = "dev"
	// BuildDate is set at build time via ldflags
	BuildDate = "unknown"
)

func main() {
	// Parse command-line flags
	configFile := flag.String("conf", "conf/server.toml", "config file path")
	showVersion := flag.Bool("version", false, "show version info")
	showHelp := flag.Bool("help", false, "show help")
	flag.Parse()

	if *showHelp {
		flag.Usage()
		os.Exit(0)
	}

	if *showVersion {
		fmt.Printf("AnyLink VPN Server\n")
		fmt.Printf("Version:    %s\n", Version)
		fmt.Printf("BuildDate:  %s\n", BuildDate)
		os.Exit(0)
	}

	// Initialize base configuration
	base.InitConfig(*configFile)
	base.InitLog()

	base.Logger.Infof("AnyLink starting, version: %s, built: %s", Version, BuildDate)

	// Initialize database
	if err := dbdata.InitDb(); err != nil {
		base.Logger.Fatalf("Failed to initialize database: %v", err)
	}

	// Initialize default data if needed
	dbdata.InitData()

	// Initialize and start the VPN handler (TUN device, IP pool, etc.)
	if err := handler.InitHandler(); err != nil {
		base.Logger.Fatalf("Failed to initialize handler: %v", err)
	}

	// Start the AnyConnect-compatible SSL VPN server
	go server.StartServer()

	// Start the admin HTTP server
	go server.StartAdmin()

	base.Logger.Info("AnyLink server started successfully")

	// Wait for termination signal.
	// SIGINT and SIGTERM trigger graceful shutdown.
	// SIGHUP is intentionally excluded here — on some systems it can be sent
	// unexpectedly (e.g. terminal disconnect) and cause unintended restarts.
	// Use SIGTERM for clean shutdown from systemd/supervisord instead.
	//
	// NOTE(personal): Also excluding SIGUSR1/SIGUSR2 — I don't use live log
	// rotation here; log rotation is handled externally by logrotate + copytruncate.
	//
	// NOTE(personal): Using a buffer of 2 instead of 1 so that if both SIGINT
	// and SIGTERM arrive in quick succession (e.g. double Ctrl-C), the second
	// signal is queued rather than dropped, preventing a potential hang on the
	// channel receive if the first signal is consumed before Notify delivers it.
	quit := make(chan os.Signal, 2)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	base.Logger.Info("AnyLink server shutting down...")

	// Graceful shutdown
	server.StopServer()
	handler.StopHandler()

	base.Logger.Info("AnyLink server stopped")
}
