package cmd

import (
	"context"

	"github.com/DustinReynoldsPE/ticket/internal/mcp"
	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start MCP server on stdio",
	RunE: func(cmd *cobra.Command, args []string) error {
		server := mcp.NewServer(TicketsDir())
		return server.Run(context.Background(), &gomcp.StdioTransport{})
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
