package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
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
	return http.FileServer(http.Dir(absOutputDir)), nil
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
