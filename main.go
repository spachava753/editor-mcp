package main

import (
	"context"
	"flag"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spachava753/editor-mcp/internal"
	"log"
	"os"
	"os/signal"
)

func main() {
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	server, err := internal.GetServer()
	if err != nil {
		log.Fatal(err)
	}
	t := mcp.NewLoggingTransport(mcp.NewStdioTransport(), os.Stderr)
	if err := server.Run(ctx, t); err != nil {
		log.Printf("Server failed: %v", err)
	}
}
