package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wgzhao/ling-box/internal/uuid"
)

var uuidCmd = &cobra.Command{
	Use:   "uuid",
	Short: "UUID generation tool",
	Long: `Generate UUIDs (Universally Unique Identifiers).

Supported UUID types:
  v1  - Time-based UUID (uses current timestamp + MAC)
  v4  - Random UUID (default, most commonly used)
  v7  - Time-ordered UUID (sortable, RFC 9562)`,
	Run: func(cmd *cobra.Command, args []string) {
		count, _ := cmd.Flags().GetInt("count")
		uuidType, _ := cmd.Flags().GetString("type")

		// Normalize type
		uuidType = strings.ToLower(uuidType)
		t := uuid.TypeV4
		switch uuidType {
		case "v1", "1":
			t = uuid.TypeV1
		case "v4", "4":
			t = uuid.TypeV4
		case "v7", "7":
			t = uuid.TypeV7
		default:
			fmt.Fprintf(cmd.OutOrStderr(), "Error: unknown UUID type %q (supported: v1, v4, v7)\n", uuidType)
			return
		}

		ids, err := uuid.Generate(count, t)
		if err != nil {
			fmt.Fprintf(cmd.OutOrStderr(), "Error: %v\n", err)
			return
		}

		for _, id := range ids {
			fmt.Println(id)
		}
	},
}

func init() {
	uuidCmd.Flags().IntP("count", "n", 1, "Number of UUIDs to generate (default: 1)")
	uuidCmd.Flags().StringP("type", "t", "v4", "UUID type: v1, v4, v7 (default: v4)")
	rootCmd.AddCommand(uuidCmd)
}
