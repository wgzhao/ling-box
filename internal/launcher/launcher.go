// Package launcher exposes ling-box tools to launcher applications such as
// Alfred, uTools and Raycast.
//
// The package is deliberately presentation-only: it knows how to turn a
// launcher query into an argv, how to run the existing lingbox commands, and
// how to shape the answer as Alfred Script Filter JSON. It never duplicates
// the tools' logic.
package launcher

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// Input modes describe how the rest of a query maps to a tool's arguments.
const (
	// InputNone means the tool takes no input (uuid, password).
	InputNone = "none"
	// InputSingle means the rest of the query is one argument (url, color).
	InputSingle = "single"
	// InputFields means the rest of the query is split into arguments
	// (ipcalc, date).
	InputFields = "fields"
	// InputStdin means the rest of the query is fed on stdin (json).
	InputStdin = "stdin"
)

// Output modes describe how a tool's answer is read back.
const (
	// OutputEnvelope means the command emits a result envelope.
	OutputEnvelope = "envelope"
	// OutputStdout means the command's stdout is the result.
	OutputStdout = "stdout"
)

// Tool describes one tool exposed to launchers. The exported fields are what
// `lingbox tools --json` publishes; the unexported fields describe how the
// adapter invokes the tool.
type Tool struct {
	ID            string   `json:"id"`
	Aliases       []string `json:"aliases"`
	Summary       string   `json:"summary"`
	Example       string   `json:"example"`
	Input         string   `json:"input"`
	Output        string   `json:"output"`
	RequiresInput bool     `json:"requires_input,omitempty"`
	LongRunning   bool     `json:"long_running,omitempty"`
	Interactive   bool     `json:"interactive,omitempty"`

	argv     []string
	dashDash bool
}

var registry = []Tool{
	{
		ID:      "uuid",
		Aliases: []string{"uuid", "guid"},
		Summary: "Generate a UUID (v4)",
		Example: "lb uuid",
		Input:   InputNone,
		Output:  OutputEnvelope,
		argv:    []string{"uuid", "--json"},
	},
	{
		ID:      "password",
		Aliases: []string{"password", "pw", "pass"},
		Summary: "Generate a secure password",
		Example: "lb password",
		Input:   InputNone,
		Output:  OutputEnvelope,
		argv:    []string{"password", "--json"},
	},
	{
		ID:            "url",
		Aliases:       []string{"url", "urlencode"},
		Summary:       "URL-encode a string",
		Example:       "lb url hello world",
		Input:         InputSingle,
		Output:        OutputEnvelope,
		RequiresInput: true,
		argv:          []string{"url", "-e", "--json"},
		dashDash:      true,
	},
	{
		ID:            "base64",
		Aliases:       []string{"base64", "b64"},
		Summary:       "Base64-encode a string",
		Example:       "lb b64 Hello World",
		Input:         InputSingle,
		Output:        OutputEnvelope,
		RequiresInput: true,
		argv:          []string{"base64", "-e", "--json"},
		dashDash:      true,
	},
	{
		ID:            "unicode",
		Aliases:       []string{"unicode", "uni"},
		Summary:       "Encode text to \\uXXXX escapes",
		Example:       `lb uni 你好`,
		Input:         InputSingle,
		Output:        OutputEnvelope,
		RequiresInput: true,
		argv:          []string{"unicode", "--json"},
		dashDash:      true,
	},
	{
		ID:            "ipcalc",
		Aliases:       []string{"ipcalc", "ip", "subnet", "cidr"},
		Summary:       "IPv4 subnet calculator",
		Example:       "lb ip 10.0.0.0/8",
		Input:         InputFields,
		Output:        OutputEnvelope,
		RequiresInput: true,
		argv:          []string{"ipcalc", "--json"},
		dashDash:      true,
	},
	{
		ID:            "color",
		Aliases:       []string{"color", "colour"},
		Summary:       "Convert a color between Hex, RGB and HSL",
		Example:       "lb color #FF0000",
		Input:         InputSingle,
		Output:        OutputEnvelope,
		RequiresInput: true,
		argv:          []string{"color", "--json"},
		dashDash:      true,
	},
	{
		ID:            "plate",
		Aliases:       []string{"plate", "carplate", "车牌"},
		Summary:       "Look up Chinese license plate regions",
		Example:       "lb plate 湘",
		Input:         InputSingle,
		Output:        OutputEnvelope,
		RequiresInput: true,
		argv:          []string{"plate", "--json"},
		dashDash:      true,
	},
	{
		ID:            "date",
		Aliases:       []string{"date"},
		Summary:       "Add days to a date or diff two dates",
		Example:       "lb date add 2026-01-01 10",
		Input:         InputFields,
		Output:        OutputStdout,
		RequiresInput: true,
		argv:          []string{"date"},
	},
	{
		ID:            "json",
		Aliases:       []string{"json", "jsonfmt"},
		Summary:       "Format JSON from stdin",
		Example:       `lb json {"a":1}`,
		Input:         InputStdin,
		Output:        OutputStdout,
		RequiresInput: true,
		argv:          []string{"json", "format"},
	},
}

// Tools returns the public metadata for every supported tool.
func Tools() []Tool {
	out := make([]Tool, len(registry))
	copy(out, registry)
	return out
}

// Query is a parsed launcher query.
type Query struct {
	Tool  *Tool
	Alias string // the token that matched the tool
	Rest  string // everything after the tool token, inner spaces preserved
}

// Parse splits a query into a tool token and its input.
func Parse(q string) Query {
	q = strings.TrimSpace(q)
	if q == "" {
		return Query{}
	}
	token, rest := q, ""
	if i := strings.IndexAny(q, " \t"); i >= 0 {
		token, rest = q[:i], strings.TrimLeft(q[i:], " \t")
	}
	t, alias := findByAlias(token)
	return Query{Tool: t, Alias: alias, Rest: rest}
}

func findByAlias(token string) (*Tool, string) {
	token = strings.ToLower(strings.TrimSpace(token))
	for i := range registry {
		for _, alias := range registry[i].Aliases {
			if strings.EqualFold(alias, token) {
				return &registry[i], alias
			}
		}
	}
	return nil, ""
}

// Runner executes argv and returns stdout. stdin is empty unless the tool
// reads its input from stdin.
type Runner func(argv []string, stdin string) (string, error)

// Build turns a launcher query into Alfred items.
func Build(query string, run Runner) []Item {
	q := Parse(query)
	if q.Tool == nil {
		return suggestions(query)
	}
	t := q.Tool
	if t.LongRunning || t.Interactive {
		return []Item{{
			UID:      "unsupported:" + t.ID,
			Title:    t.ID + " — " + t.Summary,
			Subtitle: "Not available in the palette yet; run it in a terminal",
			Valid:    false,
		}}
	}
	if t.RequiresInput && q.Rest == "" {
		return []Item{hintItem(t, q.Alias)}
	}
	argv, stdin := q.argvAndStdin()
	out, err := run(argv, stdin)
	if err != nil {
		return []Item{errorItem(t.ID, err)}
	}
	return []Item{resultItem(t, out)}
}

func (q Query) argvAndStdin() ([]string, string) {
	t := q.Tool
	argv := append([]string(nil), t.argv...)
	switch t.Input {
	case InputSingle:
		if t.dashDash {
			argv = append(argv, "--")
		}
		argv = append(argv, q.Rest)
	case InputFields:
		if t.dashDash {
			argv = append(argv, "--")
		}
		argv = append(argv, strings.Fields(q.Rest)...)
	case InputStdin:
		return argv, q.Rest
	}
	return argv, ""
}

func resultItem(t *Tool, out string) Item {
	text, copy := out, out
	if t.Output == OutputEnvelope {
		var e struct {
			OK    bool   `json:"ok"`
			Text  string `json:"text"`
			Copy  string `json:"copy"`
			Error string `json:"error"`
		}
		if err := json.Unmarshal([]byte(out), &e); err != nil {
			return errorItem(t.ID, fmt.Errorf("invalid result from %s: %w", t.ID, err))
		}
		if !e.OK {
			if e.Error == "" {
				e.Error = "tool returned an error"
			}
			return errorItem(t.ID, errors.New(e.Error))
		}
		text, copy = e.Text, e.Copy
	}

	text = strings.TrimRight(text, "\n")
	copy = strings.TrimRight(copy, "\n")
	if copy == "" {
		copy = text
	}
	if text == "" {
		text = copy
	}
	if text == "" {
		return errorItem(t.ID, errors.New("tool returned no output"))
	}

	title := firstLine(text)
	switch t.ID {
	case "ipcalc":
		// The first line carries the binary breakdown; the network line is the
		// useful summary in a one-line Alfred row.
		if line := lineWithPrefix(text, "Network:"); line != "" {
			if f := strings.Fields(line); len(f) >= 2 {
				title = f[0] + " " + f[1]
			}
		}
	case "json":
		if title == "{" || title == "[" {
			title = fmt.Sprintf("Formatted JSON (%d lines)", strings.Count(text, "\n")+1)
		}
	}
	title = truncate(title, 120)

	return Item{
		UID:      t.ID,
		Title:    title,
		Subtitle: t.ID + " · Enter to copy",
		Arg:      copy,
		Valid:    true,
		Text:     &Text{Copy: copy, Largetype: text},
	}
}

func hintItem(t *Tool, alias string) Item {
	if alias == "" {
		alias = t.ID
	}
	return Item{
		UID:          "hint:" + t.ID,
		Title:        t.ID + " — " + t.Summary,
		Subtitle:     "Type input · e.g. " + t.Example,
		Autocomplete: alias + " ",
		Valid:        false,
	}
}

func errorItem(tool string, err error) Item {
	// Cobra appends the whole usage block to argument errors. A one-line
	// subtitle is what Alfred can render, so keep the first meaningful line.
	msg := firstMeaningfulLine(err.Error())
	if msg == "" {
		msg = "unknown error"
	}
	return Item{
		UID:      "error:" + tool,
		Title:    "Error · " + tool,
		Subtitle: truncate(msg, 200),
		Valid:    false,
	}
}

func firstMeaningfulLine(s string) string {
	for _, line := range strings.Split(s, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			return line
		}
	}
	return ""
}

func suggestions(query string) []Item {
	q := strings.ToLower(strings.TrimSpace(query))
	type scored struct {
		t     *Tool
		score int
	}
	var matches []scored
	for i := range registry {
		if score := matchScore(&registry[i], q); score > 0 {
			matches = append(matches, scored{&registry[i], score})
		}
	}
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].score != matches[j].score {
			return matches[i].score > matches[j].score
		}
		return matches[i].t.ID < matches[j].t.ID
	})

	items := make([]Item, 0, len(matches))
	for _, m := range matches {
		items = append(items, Item{
			UID:          "tool:" + m.t.ID,
			Title:        m.t.ID + " — " + m.t.Summary,
			Subtitle:     m.t.Example,
			Autocomplete: m.t.ID + " ",
			Valid:        false,
		})
	}
	if len(items) == 0 {
		items = append(items, Item{
			Title:    "No matching tool",
			Subtitle: "Try: lb uuid, lb ip 10.0.0.0/8, lb b64 hello",
			Valid:    false,
		})
	}
	return items
}

func matchScore(t *Tool, q string) int {
	if q == "" {
		return 1
	}
	best := 0
	for _, alias := range append([]string{t.ID}, t.Aliases...) {
		alias = strings.ToLower(alias)
		switch {
		case alias == q:
			return 100
		case strings.HasPrefix(alias, q):
			best = max(best, 80)
		case strings.Contains(alias, q):
			best = max(best, 60)
		}
	}
	if strings.Contains(strings.ToLower(t.Summary), q) {
		best = max(best, 40)
	}
	return best
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}

// lineWithPrefix returns the first trimmed line that starts with prefix.
func lineWithPrefix(s, prefix string) string {
	for _, line := range strings.Split(s, "\n") {
		if line = strings.TrimSpace(line); strings.HasPrefix(line, prefix) {
			return line
		}
	}
	return ""
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-3]) + "..."
}

// Text is Alfred's item text object.
type Text struct {
	Copy      string `json:"copy,omitempty"`
	Largetype string `json:"largetype,omitempty"`
}

// Item is one Alfred Script Filter result.
type Item struct {
	UID          string `json:"uid,omitempty"`
	Title        string `json:"title"`
	Subtitle     string `json:"subtitle,omitempty"`
	Arg          string `json:"arg,omitempty"`
	Autocomplete string `json:"autocomplete,omitempty"`
	Valid        bool   `json:"valid"`
	Text         *Text  `json:"text,omitempty"`
}

// Response is Alfred's Script Filter response document.
type Response struct {
	Items []Item `json:"items"`
}
