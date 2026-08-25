package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/wgzhao/ling-box/internal/ipcalc"
	"golang.org/x/term"
)

var ipcalcCmd = &cobra.Command{
	Use:   "ipcalc [options] <ADDRESS>[[/]<NETMASK>] [NETMASK]",
	Short: "IPv4 subnet calculator",
	Long: `ipcalc takes an IP address and netmask and calculates the resulting
broadcast, network, Cisco wildcard mask, and host range. By giving a
second netmask, you can design sub- and supernetworks. It is also
intended to be a teaching tool and presents the results as
easy-to-understand binary values.

Netmasks may be given in CIDR notation (/25), dotted decimals
(255.255.255.0), or wildcard (inverse) notation (0.0.0.255), as used
in Cisco access control lists. Addresses may also be given in hex
(0xC0A80001). If you omit the netmask, ipcalc uses /24.

The class of the network is determined by its first bits. Private
Internet networks (RFC 1918) are remarked, and when displaying subnets
the new bits in the network part of the netmask are marked in a
different color.

Examples:
  lingbox ipcalc 192.168.0.1/24
  lingbox ipcalc 192.168.0.1/255.255.128.0
  lingbox ipcalc 192.168.0.1 255.255.128.0 255.255.192.0
  lingbox ipcalc 192.168.0.1 0.0.63.255
  lingbox ipcalc 192.168.1.10 - 192.168.2.5
  lingbox ipcalc 10.0.0.0/24 -s 100 50 25

Credits:
Go port of ipcalc 0.41 by Krischan Jodies (krischan@jodies.de,
http://jodies.de/ipcalc), licensed under GPL-2.0-or-later. Thanks to
the original's contributors: Bartosz Fenski, Denis A. Hainsworth,
Foxfair Hu, Frank Quotschalla, Hermann J. Beckers, Igor Zozulya,
Kevin Ivory, Lars Mueller, Lutz Pressler, Oliver Seufer, Scott Davis,
Steve Kent, Sven Anderson, Torgen Foertsch.`,
	Args: func(cmd *cobra.Command, args []string) error {
		// --version works without an address, as in the original.
		if v, _ := cmd.Flags().GetBool("version"); v {
			return nil
		}
		if len(args) < 1 {
			return fmt.Errorf("requires at least 1 arg(s), only received 0")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		showVersion, _ := cmd.Flags().GetBool("version")
		if showVersion {
			fmt.Println("0.41")
			return nil
		}
		nocolor, _ := cmd.Flags().GetBool("nocolor")
		nobinary, _ := cmd.Flags().GetBool("nobinary")
		class, _ := cmd.Flags().GetBool("class")
		split, _ := cmd.Flags().GetBool("split")
		rng, _ := cmd.Flags().GetBool("range")

		calc, err := ipcalc.Parse(args, split, rng)
		if err != nil {
			return err
		}
		color := !nocolor && os.Getenv("NO_COLOR") == "" && term.IsTerminal(int(os.Stdout.Fd()))
		out := calc.Render(ipcalc.Options{
			Color:    color,
			NoBinary: nobinary,
			Class:    class,
		})
		fmt.Print(out)
		return nil
	},
}

func init() {
	ipcalcCmd.Flags().BoolP("nocolor", "n", false, "Don't display ANSI color codes")
	ipcalcCmd.Flags().BoolP("nobinary", "b", false, "Suppress the bitwise output")
	ipcalcCmd.Flags().BoolP("class", "c", false, "Just print bit-count-mask of given address")
	ipcalcCmd.Flags().BoolP("split", "s", false, "Split into networks of size n1, n2, n3 (sizes given as arguments)")
	ipcalcCmd.Flags().BoolP("range", "r", false, "Deaggregate address range (ADDRESS1 - ADDRESS2)")
	ipcalcCmd.Flags().BoolP("version", "v", false, "Print version")
	rootCmd.AddCommand(ipcalcCmd)
}
