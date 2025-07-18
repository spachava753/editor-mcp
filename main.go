package main

import (
	"context"
	"flag"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spachava753/editor-mcp/internal"
	"log"
	"os"
)

func main() {
	flag.Parse()

	server, err := internal.GetServer()
	if err != nil {
		log.Fatal(err)
	}
	t := mcp.NewLoggingTransport(mcp.NewStdioTransport(), os.Stderr)
	if err := server.Run(context.Background(), t); err != nil {
		log.Printf("Server failed: %v", err)
	}
}
