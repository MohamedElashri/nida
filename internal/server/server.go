package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
)

type Instance struct {
	Address  string
	server   *http.Server
	listener net.Listener
}

func FileHandler(outputDir string) (http.Handler, error) {
	absOutputDir, err := filepath.Abs(outputDir)
	if err != nil {
		return nil, fmt.Errorf("resolve output directory %q: %w", outputDir, err)
	}

	fileServer := http.FileServer(http.Dir(absOutputDir))
	notFoundPath := filepath.Join(absOutputDir, "404.html")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := httptest.NewRecorder()
		fileServer.ServeHTTP(rec, r)

		if rec.Code != http.StatusNotFound || r.URL.Path == "/404.html" {
			writeRecordedResponse(w, rec)
			return
		}

		body, err := os.ReadFile(notFoundPath)
		if err != nil {
			writeRecordedResponse(w, rec)
			return
		}

		header := w.Header()
		header.Set("Content-Type", "text/html; charset=utf-8")
		header.Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(http.StatusNotFound)
		if r.Method == http.MethodHead {
			return
		}
		_, _ = w.Write(body)
	}), nil
}

func Start(ctx context.Context, outputDir, host string, port int) (*Instance, error) {
	handler, err := FileHandler(outputDir)
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

func writeRecordedResponse(w http.ResponseWriter, rec *httptest.ResponseRecorder) {
	res := rec.Result()
	defer res.Body.Close()

	for key, values := range res.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(res.StatusCode)
	_, _ = io.Copy(w, res.Body)
}
