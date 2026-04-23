package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Instance struct {
	Address  string
	server   *http.Server
	listener net.Listener
	reloader *reloadBroker
}

func FileHandler(outputDir string) (http.Handler, error) {
	return fileHandler(outputDir, false, nil)
}

func fileHandler(outputDir string, injectLiveReload bool, reloader *reloadBroker) (http.Handler, error) {
	absOutputDir, err := filepath.Abs(outputDir)
	if err != nil {
		return nil, fmt.Errorf("resolve output directory %q: %w", outputDir, err)
	}

	fileServer := http.FileServer(http.Dir(absOutputDir))
	notFoundPath := filepath.Join(absOutputDir, "404.html")

	fileHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := httptest.NewRecorder()
		fileServer.ServeHTTP(rec, r)

		if rec.Code != http.StatusNotFound || r.URL.Path == "/404.html" {
			writeRecordedResponse(w, rec, injectLiveReload)
			return
		}

		body, err := os.ReadFile(notFoundPath)
		if err != nil {
			writeRecordedResponse(w, rec, injectLiveReload)
			return
		}

		header := w.Header()
		header.Set("Content-Type", "text/html; charset=utf-8")
		header.Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(http.StatusNotFound)
		if r.Method == http.MethodHead {
			return
		}
		if injectLiveReload {
			body = injectReloadSnippet(body)
		}
		_, _ = w.Write(body)
	})

	if !injectLiveReload || reloader == nil {
		return fileHandler, nil
	}

	mux := http.NewServeMux()
	mux.Handle("/_nida/livereload", reloader)
	mux.HandleFunc("/_nida/livereload.js", liveReloadScriptHandler)
	mux.Handle("/", fileHandler)
	return mux, nil
}

func Start(ctx context.Context, outputDir, host string, port int, livereload bool) (*Instance, error) {
	var reloader *reloadBroker
	if livereload {
		reloader = newReloadBroker()
	}

	handler, err := fileHandler(outputDir, livereload, reloader)
	if err != nil {
		return nil, err
	}

	listener, err := net.Listen("tcp", net.JoinHostPort(host, fmt.Sprintf("%d", port)))
	if err != nil {
		return nil, fmt.Errorf("listen on %s:%d: %w", host, port, err)
	}

	srv := &http.Server{
		Handler: handler,
	}

	instance := &Instance{
		Address:  "http://" + listener.Addr().String(),
		server:   srv,
		listener: listener,
		reloader: reloader,
	}

	go func() {
		<-ctx.Done()
		_ = srv.Shutdown(context.Background())
	}()

	go func() {
		err := srv.Serve(listener)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			_ = listener.Close()
		}
	}()

	return instance, nil
}

func (i *Instance) Reload() {
	if i == nil || i.reloader == nil {
		return
	}
	i.reloader.Reload()
}

func writeRecordedResponse(w http.ResponseWriter, rec *httptest.ResponseRecorder, injectLiveReload bool) {
	res := rec.Result()
	defer res.Body.Close()

	for key, values := range res.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	body, _ := io.ReadAll(res.Body)
	if injectLiveReload && isHTMLResponse(res, body) {
		body = injectReloadSnippet(body)
		w.Header().Del("Content-Length")
	}

	w.WriteHeader(res.StatusCode)
	_, _ = w.Write(body)
}

func isHTMLResponse(res *http.Response, body []byte) bool {
	contentType := strings.ToLower(strings.TrimSpace(res.Header.Get("Content-Type")))
	if strings.Contains(contentType, "text/html") {
		return true
	}
	trimmed := bytes.TrimSpace(body)
	return bytes.HasPrefix(bytes.ToLower(trimmed), []byte("<!doctype html")) || bytes.HasPrefix(bytes.ToLower(trimmed), []byte("<html"))
}

func injectReloadSnippet(body []byte) []byte {
	snippet := []byte(`<script src="/_nida/livereload.js"></script>`)
	lower := strings.ToLower(string(body))
	index := strings.LastIndex(lower, "</body>")
	if index == -1 {
		return append(body, snippet...)
	}

	out := make([]byte, 0, len(body)+len(snippet))
	out = append(out, body[:index]...)
	out = append(out, snippet...)
	out = append(out, body[index:]...)
	return out
}

func liveReloadScriptHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = io.WriteString(w, liveReloadScript)
}

const liveReloadScript = `
(() => {
  const source = new EventSource("/_nida/livereload");
  source.onmessage = () => window.location.reload();
})();
`

type reloadBroker struct {
	mu          sync.Mutex
	subscribers map[chan string]struct{}
}

func newReloadBroker() *reloadBroker {
	return &reloadBroker{subscribers: map[chan string]struct{}{}}
}

func (b *reloadBroker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := make(chan string, 1)
	b.mu.Lock()
	b.subscribers[ch] = struct{}{}
	b.mu.Unlock()
	defer func() {
		b.mu.Lock()
		delete(b.subscribers, ch)
		b.mu.Unlock()
	}()

	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			_, _ = io.WriteString(w, ": keepalive\n\n")
			flusher.Flush()
		case msg := <-ch:
			_, _ = io.WriteString(w, "data: "+msg+"\n\n")
			flusher.Flush()
		}
	}
}

func (b *reloadBroker) Reload() {
	b.mu.Lock()
	defer b.mu.Unlock()
	for ch := range b.subscribers {
		select {
		case ch <- "reload":
		default:
		}
	}
}
