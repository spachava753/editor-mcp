package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spachava753/editor-mcp/internal"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func getVersionInfo() (string, string, string) {
	// First, use injected version info from ldflags
	if version != "dev" {
		return version, commit, date
	}

	// Fallback to runtime build info (useful for go install)
	if info, ok := debug.ReadBuildInfo(); ok {
		v := info.Main.Version
		if v == "" || v == "(devel)" {
			v = "dev"
		}

		// Extract version from -X ldflags if available
		var rev, buildTime string
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				rev = setting.Value
			case "vcs.time":
				buildTime = setting.Value
			}
		}

		if rev == "" {
			rev = "unknown"
		}
		if buildTime == "" {
			buildTime = "unknown"
		}

		return v, rev, buildTime
	}

	return "dev", "unknown", "unknown"
}

func printVersion() {
	version, commit, date := getVersionInfo()
	fmt.Printf("editor-mcp version %s\n", version)
	fmt.Printf("  commit: %s\n", commit)
	fmt.Printf("  built: %s\n", date)
}

func main() {
	flag.Parse()

	// Check for version argument
	if len(os.Args) > 1 && os.Args[1] == "version" {
		printVersion()
		os.Exit(0)
	}

	version, _, _ := getVersionInfo()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	server := internal.GetServer(version)
	t := mcp.NewLoggingTransport(mcp.NewStdioTransport(), os.Stderr)
	if err := server.Run(ctx, t); err != nil {
		log.Printf("Server failed: %v", err)
	}
}
