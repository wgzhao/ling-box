package cmd

import (
	"encoding/json"
	"io"
)

// resultEnvelope is the machine-readable result shared by commands that
// support --json and by launchers that consume it.
type resultEnvelope struct {
	Schema int    `json:"schema"`
	OK     bool   `json:"ok"`
	Tool   string `json:"tool"`
	Text   string `json:"text,omitempty"`
	Copy   string `json:"copy,omitempty"`
	Error  string `json:"error,omitempty"`
	Extra  any    `json:"extra,omitempty"`
}

// writeResult emits a successful result envelope. Callers that need a
// terminal-friendly form keep their existing output path.
func writeResult(w io.Writer, tool, text, copy string, extra any) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(resultEnvelope{
		Schema: 1,
		OK:     true,
		Tool:   tool,
		Text:   text,
		Copy:   copy,
		Extra:  extra,
	})
}

// writeErrorResult emits a failed result envelope. The process still exits
// successfully: machine consumers and launchers read ok instead of the exit
// code, which keeps an Alfred Script Filter responsive.
func writeErrorResult(w io.Writer, tool string, err error) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(resultEnvelope{
		Schema: 1,
		OK:     false,
		Tool:   tool,
		Error:  err.Error(),
	})
}
