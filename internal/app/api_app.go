package app

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

type APIApp struct {
	container *Container
	server    *http.Server
}

func NewAPIApp(ctx context.Context) (*APIApp, error) {
	container, err := Bootstrap(ctx)
	if err != nil {
		return nil, err
	}

	router := NewRouter(container)
	server := &http.Server{
		Addr:              fmt.Sprintf(":%s", container.Config.App.Port),
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       container.Config.App.ReadTimeout,
		WriteTimeout:      container.Config.App.WriteTimeout,
	}

	return &APIApp{container: container, server: server}, nil
}

func (a *APIApp) Run(ctx context.Context) error {
	errCh := make(chan error, 1)

	go func() {
		a.container.Logger.Info("api server starting", "addr", a.server.Addr)
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), a.container.Config.App.ShutdownTimeout)
		defer cancel()

		a.container.Logger.Info("api server shutting down")

		if err := a.server.Shutdown(shutdownCtx); err != nil {
			return err
		}

		return a.container.Close()
	}
}
