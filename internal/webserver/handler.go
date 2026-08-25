package webserver

import (
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Handler returns an http.Handler serving files from dir, mirroring
// Python's http.server.SimpleHTTPRequestHandler: directory listings,
// index.html support, URL decoding, MIME types, and path traversal
// protection.
func Handler(dir string) http.Handler {
	root := filepath.Clean(dir)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			sendError(w, http.StatusNotImplemented, fmt.Sprintf("Unsupported method ('%s')", r.Method))
			return
		}
		p, ok := translatePath(root, r.URL.Path)
		if !ok {
			sendError(w, http.StatusNotFound, "File not found.")
			return
		}
		info, err := os.Stat(p)
		if err != nil {
			if os.IsNotExist(err) {
				sendError(w, http.StatusNotFound, "File not found.")
			} else {
				sendError(w, http.StatusForbidden, "Permission denied.")
			}
			return
		}
		if info.IsDir() {
			// Redirect directories without a trailing slash, as Python
			// does (301, no body).
			if !strings.HasSuffix(r.URL.Path, "/") {
				u := *r.URL
				u.Path += "/"
				w.Header().Set("Location", u.String())
				w.WriteHeader(http.StatusMovedPermanently)
				return
			}
			for _, name := range []string{"index.html", "index.htm"} {
				ip := filepath.Join(p, name)
				if fi, err := os.Stat(ip); err == nil && !fi.IsDir() {
					serveFile(w, r, ip)
					return
				}
			}
			listDirectory(w, p, r.URL.Path)
			return
		}
		serveFile(w, r, p)
	})
}

// translatePath maps a URL path onto the file system below root. The
// path is normalized like posixpath.normpath and "." / ".." components
// are skipped, so ".." can never escape root (as in Python's
// translate_path). Returns false when a component would escape root.
func translatePath(root, urlPath string) (string, bool) {
	p := path.Clean(urlPath)
	var words []string
	for _, w := range strings.Split(p, "/") {
		switch w {
		case "", ".", "..":
			continue
		}
		words = append(words, w)
	}
	full := filepath.Join(append([]string{root}, words...)...)
	// Safety net for platform path tricks (backslashes, volume names).
	rel, err := filepath.Rel(root, full)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", false
	}
	return full, true
}

// listDirectory renders the directory listing HTML in Python
// http.server's format.
func listDirectory(w http.ResponseWriter, dir, urlPath string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		sendError(w, http.StatusNotFound, "No permission to list directory")
		return
	}
	sort.SliceStable(entries, func(i, j int) bool {
		return strings.ToLower(entries[i].Name()) < strings.ToLower(entries[j].Name())
	})

	title := "Directory listing for " + html.EscapeString(urlPath)
	var b strings.Builder
	fmt.Fprintf(&b, "<!DOCTYPE HTML>\n<html lang=\"en\">\n<head>\n<meta charset=\"utf-8\">\n<title>%s</title>\n</head>\n<body>\n<h1>%s</h1>\n<hr>\n<ul>\n", title, title)
	for _, e := range entries {
		name := e.Name()
		display, link := name, url.PathEscape(name)
		info, err := e.Info()
		isDir := err == nil && info.IsDir()
		if isDir {
			display += "/"
			link += "/"
		}
		if e.Type()&os.ModeSymlink != 0 {
			display = name + "@"
		}
		fmt.Fprintf(&b, "<li><a href=\"%s\">%s</a></li>\n", link, html.EscapeString(display))
	}
	b.WriteString("</ul>\n<hr>\n</body>\n</html>\n")

	body := b.String()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, body)
}

// serveFile streams a single file with its MIME type, honoring range
// requests via http.ServeContent.
func serveFile(w http.ResponseWriter, r *http.Request, path string) {
	f, err := os.Open(path)
	if err != nil {
		sendError(w, http.StatusForbidden, "Permission denied.")
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		sendError(w, http.StatusNotFound, "File not found.")
		return
	}
	w.Header().Set("Content-Type", mimeType(path))
	http.ServeContent(w, r, info.Name(), info.ModTime(), f)
}

// mimeTypes maps file extensions to content types, keeping the MIME
// table self-contained like Python's mimetypes module.
var mimeTypes = map[string]string{
	".html":  "text/html",
	".htm":   "text/html",
	".txt":   "text/plain",
	".css":   "text/css",
	".csv":   "text/csv",
	".js":    "text/javascript",
	".mjs":   "text/javascript",
	".json":  "application/json",
	".xml":   "text/xml",
	".svg":   "image/svg+xml",
	".png":   "image/png",
	".jpg":   "image/jpeg",
	".jpeg":  "image/jpeg",
	".gif":   "image/gif",
	".webp":  "image/webp",
	".ico":   "image/x-icon",
	".bmp":   "image/bmp",
	".pdf":   "application/pdf",
	".zip":   "application/zip",
	".gz":    "application/gzip",
	".tar":   "application/x-tar",
	".wasm":  "application/wasm",
	".mp3":   "audio/mpeg",
	".mp4":   "video/mp4",
	".webm":  "video/webm",
	".woff":  "font/woff",
	".woff2": "font/woff2",
	".ttf":   "font/ttf",
}

// mimeType returns the content type for file, like
// mimetypes.guess_type.
func mimeType(file string) string {
	if t, ok := mimeTypes[strings.ToLower(path.Ext(file))]; ok {
		return t
	}
	return "application/octet-stream"
}

// errorReasons are the explanation lines in error pages, from Python's
// http.server responses table.
var errorReasons = map[int]string{
	http.StatusBadRequest:          "Bad request syntax or unsupported method",
	http.StatusForbidden:           "Access forbidden",
	http.StatusNotFound:            "Nothing matches the given URI.",
	http.StatusMethodNotAllowed:    "Specified method is invalid for this resource.",
	http.StatusInternalServerError: "Server got itself in trouble",
	http.StatusNotImplemented:      "Unsupported method",
}

// sendError writes a Python http.server-style HTML error page.
func sendError(w http.ResponseWriter, code int, message string) {
	if s, ok := w.(errorSink); ok {
		s.setError(message)
	}
	reason := errorReasons[code]
	body := fmt.Sprintf(`<!DOCTYPE HTML>
<html lang="en">
    <head>
        <meta charset="utf-8">
        <title>Error response</title>
    </head>
    <body>
        <h1>Error response</h1>
        <p>Error code: %d</p>
        <p>Message: %s</p>
        <p>Error code explanation: %d - %s</p>
    </body>
</html>
`, code, message, code, reason)
	// Python's error_content_type is "text/html;charset=utf-8" (no space).
	w.Header().Set("Content-Type", "text/html;charset=utf-8")
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	w.WriteHeader(code)
	io.WriteString(w, body)
}
