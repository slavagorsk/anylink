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

	base.Logger.Infof("AnyLink starting, version: %s", Version)

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

	// Wait for termination signal
	// Also handle SIGHUP so the process can be cleanly managed by systemd/supervisord
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	<-quit

	base.Logger.Info("AnyLink server shutting down...")

	// Graceful shutdown
	server.StopServer()
	handler.StopHandler()

	base.Logger.Info("AnyLink server stopped")
}
