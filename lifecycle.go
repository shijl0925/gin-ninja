package ninja

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// LifecycleHook runs during API startup or shutdown.
type LifecycleHook func(context.Context, *NinjaAPI) error

type lifecycleState struct {
	mu          sync.Mutex
	shutdownRan bool
}

type serverState struct {
	mu     sync.RWMutex
	server *http.Server
}

// OnStartup registers a hook that runs before the HTTP server starts accepting traffic.
func (api *NinjaAPI) OnStartup(hook LifecycleHook) {
	api.startupHooks = append(api.startupHooks, hook)
}

// OnShutdown registers a hook that runs during graceful shutdown.
func (api *NinjaAPI) OnShutdown(hook LifecycleHook) {
	api.shutdownHooks = append(api.shutdownHooks, hook)
}

// Serve starts serving HTTP on the given listener and enables graceful shutdown.
func (api *NinjaAPI) Serve(listener net.Listener) error {
	if listener == nil {
		return fmt.Errorf("listener must not be nil")
	}

	server := &http.Server{Handler: api.engine}
	if err := api.runStartupHooks(context.Background()); err != nil {
		_ = listener.Close()
		return err
	}

	api.setServer(server)
	defer api.clearServer()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Serve(listener)
	}()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return api.runShutdownHooks(context.Background())
		}
		return errors.Join(err, api.runShutdownHooks(context.Background()))
	case <-sigCh:
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		shutdownErr := api.Shutdown(shutdownCtx)
		err := <-errCh
		if errors.Is(err, http.ErrServerClosed) {
			return shutdownErr
		}
		return errors.Join(err, shutdownErr)
	}
}

// Shutdown gracefully stops the active server and runs shutdown hooks once.
func (api *NinjaAPI) Shutdown(ctx context.Context) error {
	var errs []error
	if server := api.currentServer(); server != nil {
		if err := server.Shutdown(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errs = append(errs, err)
		}
	}
	if err := api.runShutdownHooks(ctx); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func (api *NinjaAPI) runStartupHooks(ctx context.Context) error {
	api.lifecycle.mu.Lock()
	api.lifecycle.shutdownRan = false
	api.lifecycle.mu.Unlock()
	for _, hook := range api.startupHooks {
		if err := hook(ctx, api); err != nil {
			return err
		}
	}
	return nil
}

func (api *NinjaAPI) runShutdownHooks(ctx context.Context) error {
	api.lifecycle.mu.Lock()
	if api.lifecycle.shutdownRan {
		api.lifecycle.mu.Unlock()
		return nil
	}
	api.lifecycle.shutdownRan = true
	api.lifecycle.mu.Unlock()

	var errs []error
	for i := len(api.shutdownHooks) - 1; i >= 0; i-- {
		if err := api.shutdownHooks[i](ctx, api); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (api *NinjaAPI) setServer(server *http.Server) {
	api.serverState.mu.Lock()
	api.serverState.server = server
	api.serverState.mu.Unlock()
}

func (api *NinjaAPI) clearServer() {
	api.serverState.mu.Lock()
	api.serverState.server = nil
	api.serverState.mu.Unlock()
}

func (api *NinjaAPI) currentServer() *http.Server {
	api.serverState.mu.RLock()
	defer api.serverState.mu.RUnlock()
	return api.serverState.server
}
