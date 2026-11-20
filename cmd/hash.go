package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wgzhao/ling-box/internal/hashx"
)

var hashCmd = &cobra.Command{
	Use:     "hash [input]",
	Aliases: []string{"digest", "sum"},
	Short:   "Hash a string, file, or stdin",
	Long: `Compute message digests and HMACs.

Algorithms: md5, sha1, sha224, sha256, sha384, sha512, sm3, crc32 and the
hmac-<hash> variants. Multiple algorithms can be requested with a
comma-separated list.

Input comes from the positional argument, --file, or stdin.

Examples:
  lingbox hash abc
  lingbox hash -a md5,sha256 abc
  lingbox hash -a sm3 "hello world"
  lingbox hash -f archive.zip -a sha256
  lingbox hash -a hmac-sha256 -k secret "message"
  echo -n abc | lingbox hash -a sha1`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		algoList, _ := cmd.Flags().GetString("algo")
		file, _ := cmd.Flags().GetString("file")
		key, _ := cmd.Flags().GetString("key")
		asJSON, _ := cmd.Flags().GetBool("json")

		algos := splitAlgorithms(algoList)
		if len(algos) == 0 {
			reportHashError(cmd, asJSON, fmt.Errorf("no algorithm specified"))
			return
		}

		var data []byte
		if file == "" {
			if len(args) == 1 {
				data = []byte(args[0])
			} else {
				stdin, err := readStdin()
				if err != nil {
					reportHashError(cmd, asJSON, err)
					return
				}
				if stdin == nil {
					reportHashError(cmd, asJSON, fmt.Errorf("no input: pass a string, use --file, or pipe data on stdin"))
					return
				}
				data = stdin
			}
		} else if len(args) > 0 {
			reportHashError(cmd, asJSON, fmt.Errorf("--file cannot be combined with a positional input"))
			return
		}

		type result struct {
			Algorithm string `json:"algorithm"`
			Digest    string `json:"digest"`
		}
		results := make([]result, 0, len(algos))

		var f *os.File
		if file != "" {
			var err error
			f, err = os.Open(file)
			if err != nil {
				reportHashError(cmd, asJSON, fmt.Errorf("open %s: %w", file, err))
				return
			}
			defer f.Close()
		}

		for _, algo := range algos {
			var (
				digest string
				err    error
			)
			if f != nil {
				if _, err = f.Seek(0, 0); err == nil {
					digest, err = hashx.SumReader(f, algo, []byte(key))
				}
			} else {
				digest, err = hashx.Sum(data, algo, []byte(key))
			}
			if err != nil {
				reportHashError(cmd, asJSON, err)
				return
			}
			results = append(results, result{Algorithm: algo, Digest: digest})
		}

		lines := make([]string, len(results))
		for i, r := range results {
			lines[i] = r.Algorithm + "  " + r.Digest
		}
		text := strings.Join(lines, "\n")
		copyText := results[0].Digest
		if len(results) > 1 {
			copyText = text
		}

		if asJSON {
			extra := struct {
				File    string   `json:"file,omitempty"`
				Results []result `json:"results"`
			}{File: file, Results: results}
			_ = writeResult(cmd.OutOrStdout(), "hash", text, copyText, extra)
			return
		}

		if file != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "File: %s\n", file)
		}
		fmt.Fprintln(cmd.OutOrStdout(), text)
	},
}

func splitAlgorithms(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, strings.ToLower(p))
		}
	}
	return out
}

func reportHashError(cmd *cobra.Command, asJSON bool, err error) {
	if asJSON {
		_ = writeErrorResult(cmd.OutOrStdout(), "hash", err)
		return
	}
	fmt.Fprintf(cmd.OutOrStderr(), "Error: %v\n", err)
}

func init() {
	hashCmd.Flags().StringP("algo", "a", "sha256", "Comma-separated algorithms (md5, sha1, sha224, sha256, sha384, sha512, sm3, crc32, hmac-*)")
	hashCmd.Flags().StringP("file", "f", "", "Hash a file instead of a string")
	hashCmd.Flags().StringP("key", "k", "", "Key for hmac-* algorithms")
	hashCmd.Flags().Bool("json", false, "Output a machine-readable JSON result envelope")
	rootCmd.AddCommand(hashCmd)
}
