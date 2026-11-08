package e2e

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type dbfFieldSpec struct {
	name    string
	typ     byte
	length  int
	decimal int
}

// buildDBF assembles a minimal dBASE III table for the file-action tests.
func buildDBF(fields []dbfFieldSpec, rows [][]byte) []byte {
	recSize := 1
	for _, f := range fields {
		recSize += f.length
	}
	headerSize := 32 + 32*len(fields) + 1

	out := make([]byte, 0, headerSize+recSize*len(rows)+1)
	h := make([]byte, 32)
	h[0] = 0x03
	h[1], h[2], h[3] = 26, 8, 25
	binary.LittleEndian.PutUint32(h[4:8], uint32(len(rows)))
	binary.LittleEndian.PutUint16(h[8:10], uint16(headerSize))
	binary.LittleEndian.PutUint16(h[10:12], uint16(recSize))
	out = append(out, h...)

	for _, f := range fields {
		d := make([]byte, 32)
		copy(d[0:11], f.name)
		d[11] = f.typ
		d[16] = byte(f.length)
		d[17] = byte(f.decimal)
		out = append(out, d...)
	}
	out = append(out, 0x0D)

	for _, row := range rows {
		out = append(out, 0x20)
		out = append(out, row...)
		if len(row) < recSize-1 {
			out = append(out, make([]byte, recSize-1-len(row))...)
		}
	}
	out = append(out, 0x1A)
	return out
}

func padDBF(s string, width int) []byte {
	b := make([]byte, width)
	copy(b, s)
	for i := len(s); i < width; i++ {
		b[i] = ' '
	}
	return b
}

func writePNG(t *testing.T, path string) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

func findAction(items []alfredItem, title string) (alfredItem, bool) {
	for _, item := range items {
		if item.Title == title {
			return item, true
		}
	}
	return alfredItem{}, false
}

func runAction(t *testing.T, name, spec string) string {
	t.Helper()
	out, errOut, code := runBin(t, "", "launcher", "action", "--spec", spec)
	saveArtifact(t, "file-action-"+name, ".txt", out)
	if code != 0 {
		t.Fatalf("%s: exit %d, stderr: %s", name, code, errOut)
	}
	return out
}

func TestLauncherFileActions(t *testing.T) {
	dir := t.TempDir()

	// --- DBF: info and JSON export ---
	dbfPath := filepath.Join(dir, "demo.dbf")
	fields := []dbfFieldSpec{{"NAME", 'C', 10, 0}, {"AGE", 'N', 3, 0}}
	row := append(padDBF("alice", 10), padDBF(" 25", 3)...)
	if err := os.WriteFile(dbfPath, buildDBF(fields, [][]byte{row}), 0o644); err != nil {
		t.Fatal(err)
	}

	out, errOut, code := runBin(t, "", "launcher", "file", "--", dbfPath)
	if code != 0 {
		t.Fatalf("launcher file: exit %d, stderr: %s", code, errOut)
	}
	r := parseAlfred(t, "file-dbf", out)
	for _, want := range []string{"DBF: Show info", "DBF: Export JSON", "DBF: Export CSV"} {
		if _, ok := findAction(r.Items, want); !ok {
			t.Errorf("missing %q in %+v", want, r.Items)
		}
	}

	info, ok := findAction(r.Items, "DBF: Show info")
	if !ok {
		t.Fatal("DBF info action missing")
	}
	infoOut := runAction(t, "dbf-info", info.Arg)
	if !strings.Contains(infoOut, "Fields") && !strings.Contains(infoOut, "copied to clipboard") {
		t.Errorf("DBF info output = %q", infoOut)
	}

	export, ok := findAction(r.Items, "DBF: Export JSON")
	if !ok {
		t.Fatal("DBF export action missing")
	}
	runAction(t, "dbf-export", export.Arg)
	exportPath := filepath.Join(dir, "demo.json")
	data, err := os.ReadFile(exportPath)
	if err != nil {
		t.Fatalf("read exported JSON: %v", err)
	}
	saveArtifact(t, "file-dbf-export-json", ".json", string(data))
	if !strings.Contains(string(data), `"NAME": "alice"`) {
		t.Errorf("exported JSON = %s", data)
	}

	// --- Convert JSON to YAML ---
	jsonPath := filepath.Join(dir, "data.json")
	if err := os.WriteFile(jsonPath, []byte(`{"name":"alice","age":25}`), 0o644); err != nil {
		t.Fatal(err)
	}
	out, _, code = runBin(t, "", "launcher", "file", "--", jsonPath)
	if code != 0 {
		t.Fatalf("launcher file json: exit %d", code)
	}
	r = parseAlfred(t, "file-json", out)
	yamlAction, ok := findAction(r.Items, "Convert to yaml")
	if !ok {
		t.Fatalf("Convert to yaml missing: %+v", r.Items)
	}
	runAction(t, "convert-yaml", yamlAction.Arg)
	yamlPath := filepath.Join(dir, "data.yaml")
	yamlData, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatalf("read converted YAML: %v", err)
	}
	saveArtifact(t, "file-convert-yaml", ".yaml", string(yamlData))
	if !strings.Contains(string(yamlData), "name: alice") {
		t.Errorf("converted YAML = %s", yamlData)
	}

	// A failed conversion must report the CLI error instead of a false
	// success message.
	badPath := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(badPath, []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, _, code = runBin(t, "", "launcher", "file", "--", badPath)
	if code != 0 {
		t.Fatalf("launcher file bad json: exit %d", code)
	}
	r = parseAlfred(t, "file-bad-json", out)
	badAction, ok := findAction(r.Items, "Convert to yaml")
	if !ok {
		t.Fatalf("Convert to yaml missing for bad JSON: %+v", r.Items)
	}
	badMsg := runAction(t, "convert-bad", badAction.Arg)
	if !strings.Contains(badMsg, "Error") && !strings.Contains(badMsg, "invalid JSON") {
		t.Errorf("bad convert output = %q", badMsg)
	}

	// --- Image: the item exists, and headless mode returns the command ---
	imgPath := filepath.Join(dir, "pixel.png")
	writePNG(t, imgPath)
	out, _, code = runBin(t, "", "launcher", "file", "--", imgPath)
	if code != 0 {
		t.Fatalf("launcher file image: exit %d", code)
	}
	r = parseAlfred(t, "file-image", out)
	imgcat, ok := findAction(r.Items, "View in Terminal")
	if !ok {
		t.Fatalf("imgcat action missing: %+v", r.Items)
	}
	msg := runAction(t, "imgcat", imgcat.Arg)
	if !strings.Contains(msg, "imgcat") {
		t.Errorf("imgcat action output = %q", msg)
	}

	// --- Unsupported file ---
	txtPath := filepath.Join(dir, "notes.txt")
	if err := os.WriteFile(txtPath, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, _, code = runBin(t, "", "launcher", "file", "--", txtPath)
	if code != 0 {
		t.Fatalf("launcher file txt: exit %d", code)
	}
	r = parseAlfred(t, "file-txt", out)
	if r.Items[0].Valid || !strings.Contains(r.Items[0].Title, "No ling-box action") {
		t.Errorf("unsupported file item = %+v", r.Items[0])
	}

	record(t, "file actions", "dbf, convert, imgcat, unsupported")
}

func TestFileActionScripts(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell required")
	}
	dir := t.TempDir()
	dbfPath := filepath.Join(dir, "demo.dbf")
	if err := os.WriteFile(dbfPath, buildDBF([]dbfFieldSpec{{"NAME", 'C', 10, 0}}, [][]byte{padDBF("alice", 10)}), 0o644); err != nil {
		t.Fatal(err)
	}

	env := envWith(map[string]string{"LINGBOX_BIN": binPath, "LINGBOX_NO_GUI": "1"})
	runScript := func(script string, args ...string) ([]byte, error) {
		cmd := exec.Command("/bin/sh", append([]string{filepath.Join(repoRoot, "contrib", "alfred", script)}, args...)...)
		cmd.Dir = repoRoot
		cmd.Env = env
		return cmd.Output()
	}

	out, err := runScript("file-filter.sh", dbfPath)
	if err != nil {
		t.Fatalf("file-filter.sh: %v", err)
	}
	saveArtifact(t, "alfred-file-filter", ".json", string(out))
	var r alfredResponse
	if err := json.Unmarshal(out, &r); err != nil {
		t.Fatalf("invalid file-filter JSON: %v\n%s", err, out)
	}
	export, ok := findAction(r.Items, "DBF: Export JSON")
	if !ok {
		t.Fatalf("DBF export action missing: %+v", r.Items)
	}

	out, err = runScript("run-action.sh", export.Arg)
	if err != nil {
		t.Fatalf("run-action.sh: %v", err)
	}
	saveArtifact(t, "alfred-run-action", ".txt", string(out))
	if !strings.Contains(string(out), "Exported to") {
		t.Errorf("run-action output = %q", out)
	}
	if _, err := os.Stat(filepath.Join(dir, "demo.json")); err != nil {
		t.Errorf("exported JSON missing: %v", err)
	}
	record(t, "alfred file scripts", "file-filter + run-action")
}

func TestUToolsPackage(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("build.sh needs a POSIX shell")
	}
	if _, err := exec.LookPath("zip"); err != nil {
		t.Skip("zip not installed")
	}
	build := exec.Command("/bin/sh", "contrib/utools/build.sh")
	build.Dir = repoRoot
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build uTools plugin: %v\n%s", err, out)
	}
	upxPath := filepath.Join(repoRoot, "contrib", "utools", "dist", "ling-box.upx")

	zr, err := zip.OpenReader(upxPath)
	if err != nil {
		t.Fatalf("open uTools plugin: %v", err)
	}
	defer zr.Close()

	names := map[string]bool{}
	var plugin struct {
		PluginName string `json:"pluginName"`
		Preload    string `json:"preload"`
		Features   []struct {
			Code string   `json:"code"`
			Cmds []string `json:"cmds"`
		} `json:"features"`
	}
	for _, f := range zr.File {
		names[f.Name] = true
		if f.Name == "plugin.json" {
			rc, err := f.Open()
			if err != nil {
				t.Fatal(err)
			}
			if err := json.NewDecoder(rc).Decode(&plugin); err != nil {
				t.Fatal(err)
			}
			rc.Close()
		}
	}
	for _, want := range []string{"plugin.json", "preload.js", "index.html", "logo.png"} {
		if !names[want] {
			t.Errorf("uTools plugin missing %s (has %v)", want, names)
		}
	}
	if plugin.Preload != "preload.js" || plugin.PluginName != "ling-box" {
		t.Errorf("plugin.json = %+v", plugin)
	}
	if len(plugin.Features) == 0 || plugin.Features[0].Code != "lb" || !contains(plugin.Features[0].Cmds, "lb") {
		t.Errorf("features = %+v", plugin.Features)
	}

	data, err := os.ReadFile(upxPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(artifactsDir, "ling-box.upx"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	record(t, "utools package", fmt.Sprintf("%d files", len(zr.File)))
}

func TestUToolsAdapter(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not installed")
	}
	testScript := filepath.Join(repoRoot, "contrib", "utools", "test.js")

	tests := []struct {
		name     string
		query    string
		contains string
		uuid     bool
	}{
		{name: "uuid", query: "uuid", uuid: true},
		{name: "ip", query: "ip 10.0.0.0/8", contains: "10.0.0.0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("node", testScript, binPath, tt.query)
			cmd.Dir = repoRoot
			var out, errOut bytes.Buffer
			cmd.Stdout, cmd.Stderr = &out, &errOut
			if err := cmd.Run(); err != nil {
				t.Fatalf("node test.js: %v\n%s", err, errOut.String())
			}
			saveArtifact(t, "utools-"+tt.name, ".json", out.String())

			var r alfredResponse
			if err := json.Unmarshal(out.Bytes(), &r); err != nil {
				t.Fatalf("invalid JSON: %v\n%s", err, out.String())
			}
			if len(r.Items) != 1 || !r.Items[0].Valid {
				t.Fatalf("items = %+v", r.Items)
			}
			if tt.uuid && !uuidV4.MatchString(r.Items[0].Arg) {
				t.Errorf("uuid = %q", r.Items[0].Arg)
			}
			if tt.contains != "" && !strings.Contains(r.Items[0].Arg, tt.contains) {
				t.Errorf("arg missing %q:\n%s", tt.contains, r.Items[0].Arg)
			}
		})
	}
	record(t, "utools adapter", "node queries")
}
