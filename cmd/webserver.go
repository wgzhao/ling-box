package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/wgzhao/ling-box/internal/webserver"
)

var webserverCmd = &cobra.Command{
	Use:   "webserver [port]",
	Short: "Serve the current directory over HTTP",
	Long: `Start a temporary HTTP server for the current directory, similar to
python3 -m http.server.

Serves files with automatic MIME type detection, URL decoding,
index.html support, and directory listings. Path traversal attempts
are blocked. Request logs are written to stderr; press Ctrl-C to stop.

Examples:
  lingbox webserver              # serve ./ on port 8000
  lingbox webserver 8080         # serve ./ on port 8080
  lingbox webserver -d ~/public  # serve a specific directory
  lingbox webserver -b 127.0.0.1 # bind to localhost only
  lingbox webserver -d . 8080`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, _ := cmd.Flags().GetString("directory")
		bind, _ := cmd.Flags().GetString("bind")
		port := 8000
		if len(args) == 1 {
			p, err := strconv.Atoi(args[0])
			if err != nil || p < 0 || p > 65535 {
				return fmt.Errorf("invalid port %q (0-65535)", args[0])
			}
			port = p
		}
		return webserver.Serve(webserver.Options{
			Dir:  dir,
			Bind: bind,
			Port: port,
			Out:  cmd.OutOrStdout(),
			Err:  cmd.ErrOrStderr(),
		})
	},
}

func init() {
	webserverCmd.Flags().StringP("directory", "d", ".", "serve this directory (default: current directory)")
	webserverCmd.Flags().StringP("bind", "b", "", "bind to this address (default: all interfaces)")
	rootCmd.AddCommand(webserverCmd)
}
