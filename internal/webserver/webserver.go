// Package webserver serves a directory over HTTP with a Python
// http.server-compatible interface: the same startup message, request
// log format, directory listings, and error pages.
package webserver

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Options configures Serve.
type Options struct {
	Dir  string    // directory to serve (default ".")
	Bind string    // bind address (default "" = all interfaces)
	Port int       // port (default 8000; 0 picks a free port)
	Out  io.Writer // startup messages (default os.Stdout)
	Err  io.Writer // request logs (default os.Stderr)
}

// Serve starts the HTTP server and blocks until interrupted (Ctrl-C)
// or a fatal error occurs, mirroring python3 -m http.server.
func Serve(o Options) error {
	if o.Out == nil {
		o.Out = os.Stdout
	}
	if o.Err == nil {
		o.Err = os.Stderr
	}

	dir, err := filepath.Abs(o.Dir)
	if err != nil {
		return err
	}
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("directory must exist and be a directory: %s", o.Dir)
	}

	handler := Handler(dir)
	srv := &http.Server{
		Addr:              net.JoinHostPort(o.Bind, strconv.Itoa(o.Port)),
		Handler:           logging(handler, o.Err),
		ReadHeaderTimeout: 10 * time.Second,
	}
	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		return err
	}
	fmt.Fprintln(o.Out, serveMessage(o.Bind, ln.Addr().(*net.TCPAddr).Port))

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	defer signal.Stop(sig)

	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.Serve(ln) }()

	select {
	case err := <-serveErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-sig:
		fmt.Fprintln(o.Out, "Keyboard interrupt received, exiting.")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(ctx)
		return nil
	}
}

// serveMessage renders the startup line, matching Python's http.server:
//
//	Serving HTTP on 127.0.0.1 port 8000 (http://127.0.0.1:8000/) ...
func serveMessage(bind string, port int) string {
	host := bind
	if host == "" {
		host = "0.0.0.0"
	}
	urlHost := host
	if strings.Contains(host, ":") {
		urlHost = "[" + host + "]"
	}
	return fmt.Sprintf("Serving HTTP on %s port %d (http://%s:%d/) ...", host, port, urlHost, port)
}

// logging wraps handler, writing Python http.server-style lines to w:
//
//	127.0.0.1 - - [25/Aug/2026 20:59:03] "GET / HTTP/1.1" 200 12
//	127.0.0.1 - - [25/Aug/2026 20:59:03] code 404, message File not found
func logging(next http.Handler, w io.Writer) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		sw := &statusWriter{ResponseWriter: rw}
		next.ServeHTTP(sw, r)
		host, _, _ := net.SplitHostPort(r.RemoteAddr)
		if host == "" {
			host = r.RemoteAddr
		}
		status := sw.status
		if status == 0 {
			status = http.StatusOK
		}
		t := time.Now()
		if status >= 400 {
			fmt.Fprintf(w, "%s - - [%s] code %d, message %s\n",
				host, t.Format("02/Jan/2006 15:04:05"), status, sw.errMsg)
		}
		fmt.Fprintln(w, logLine(host, r.Method+" "+r.URL.RequestURI()+" "+r.Proto, status, sw.size, t))
	})
}

// logLine renders one request log line.
func logLine(host, request string, status int, size int64, t time.Time) string {
	return fmt.Sprintf("%s - - [%s] %q %d %d",
		host, t.Format("02/Jan/2006 15:04:05"), request, status, size)
}

// statusWriter records the response status, byte size, and any error
// message sent by the handler.
type statusWriter struct {
	http.ResponseWriter
	status int
	size   int64
	errMsg string
}

func (w *statusWriter) WriteHeader(code int) {
	if w.status == 0 {
		w.status = code
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(b)
	w.size += int64(n)
	return n, err
}

// errorSink lets sendError report the error message for logging.
type errorSink interface{ setError(message string) }

func (w *statusWriter) setError(message string) {
	w.errMsg = message
}
