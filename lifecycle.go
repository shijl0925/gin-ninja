package ninja

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
)

// LifecycleHook runs during API startup or shutdown.
type LifecycleHook func(context.Context, *NinjaAPI) error

type lifecycleState struct {
	mu                 sync.Mutex
	cond               *sync.Cond
	starting           bool
	generation         uint64
	shutdownGeneration uint64
}

type serverState struct {
	mu     sync.RWMutex
	server *http.Server
}

// OnStartup registers a hook that runs before the HTTP server starts accepting traffic.
func (api *NinjaAPI) OnStartup(hook LifecycleHook) {
	api.hooksMu.Lock()
	defer api.hooksMu.Unlock()
	api.startupHooks = append(api.startupHooks, hook)
}

// OnShutdown registers a hook that runs during graceful shutdown.
func (api *NinjaAPI) OnShutdown(hook LifecycleHook) {
	api.hooksMu.Lock()
	defer api.hooksMu.Unlock()
	api.shutdownHooks = append(api.shutdownHooks, hook)
}

// Serve starts serving HTTP on the given listener.
func (api *NinjaAPI) Serve(listener net.Listener) error {
	return api.serve(listener, context.Background())
}

func (api *NinjaAPI) serve(listener net.Listener, startupCtx context.Context) error {
	if listener == nil {
		return fmt.Errorf("listener must not be nil")
	}

	server := api.newHTTPServer()
	if err := api.runStartupHooks(startupCtx); err != nil {
		shutdownCtx, cancel := api.shutdownContext(context.Background())
		defer cancel()
		cleanupErr := api.runShutdownHooks(shutdownCtx)
		_ = listener.Close()
		return errors.Join(err, cleanupErr)
	}

	api.setServer(server)
	defer api.clearServer()
	api.printStartupBanner(listener)

	err := server.Serve(listener)
	shutdownCtx, cancel := api.shutdownContext(context.Background())
	defer cancel()
	if errors.Is(err, http.ErrServerClosed) {
		return api.runShutdownHooks(shutdownCtx)
	}
	return errors.Join(err, api.runShutdownHooks(shutdownCtx))
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
	if api.lifecycle.cond == nil {
		api.lifecycle.cond = sync.NewCond(&api.lifecycle.mu)
	}
	api.lifecycle.starting = true
	api.lifecycle.generation++
	api.lifecycle.mu.Unlock()
	defer func() {
		api.lifecycle.mu.Lock()
		api.lifecycle.starting = false
		api.lifecycle.cond.Broadcast()
		api.lifecycle.mu.Unlock()
	}()

	api.hooksMu.RLock()
	hooks := append([]LifecycleHook(nil), api.startupHooks...)
	api.hooksMu.RUnlock()
	for _, hook := range hooks {
		if err := hook(ctx, api); err != nil {
			return err
		}
	}
	return nil
}

func (api *NinjaAPI) runShutdownHooks(ctx context.Context) error {
	api.lifecycle.mu.Lock()
	if api.lifecycle.cond == nil {
		api.lifecycle.cond = sync.NewCond(&api.lifecycle.mu)
	}
	for api.lifecycle.starting {
		api.lifecycle.cond.Wait()
	}
	generation := api.lifecycle.generation
	if api.lifecycle.shutdownGeneration == generation {
		api.lifecycle.mu.Unlock()
		return nil
	}
	api.lifecycle.shutdownGeneration = generation
	api.lifecycle.mu.Unlock()

	api.hooksMu.RLock()
	hooks := append([]LifecycleHook(nil), api.shutdownHooks...)
	api.hooksMu.RUnlock()

	var errs []error
	for i := len(hooks) - 1; i >= 0; i-- {
		if err := hooks[i](ctx, api); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (api *NinjaAPI) newHTTPServer() *http.Server {
	return &http.Server{
		Handler:      api.engine,
		ReadTimeout:  api.config.ReadTimeout,
		WriteTimeout: api.config.WriteTimeout,
		IdleTimeout:  api.config.IdleTimeout,
	}
}

func (api *NinjaAPI) shutdownContext(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, api.config.GracefulShutdownTimeout)
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
