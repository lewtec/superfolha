package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/lewtec/lewkit/x/app"
	"github.com/lewtec/lewkit/x/driver/bundle"
	"github.com/lewtec/lewkit/x/release"
	"github.com/lewtec/superfolha/internal/db"
)

const (
	appTitle  = "Superfolha"
	appWidth  = 1280
	appHeight = 800
)

// runWindow opens the lewkit app. A headless host (LEWKIT_NO_UI or
// ELETROCROMO_NO_UI) serves the handler on a loopback port. A desktop
// window still needs a real HTTP listener: the web view origin is app://,
// and the editor WebSocket cannot run there.
func runWindow(ctx context.Context) (err error) {
	root, err := bundle.Resolve(ctx)
	if err != nil {
		return fmt.Errorf("app data: %w", err)
	}
	var database db.DBArg
	in, err := openInstance(ctx, root.Data, &database)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, in.Close()) }()
	slog.Info("connected to database", "driver", "sqlite", "path", database.Value().URL(), "state", root.Data)

	if headlessHost() {
		err = (app.App{
			Title:   appTitle,
			Width:   appWidth,
			Height:  appHeight,
			Handler: app.Web(in.srv.Handler()),
		}).Run(ctx)
		in.srv.CloseHubs()
		return err
	}
	return runDesktopWindow(ctx, in)
}

func runDesktopWindow(ctx context.Context, in *instance) error {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	target := "http://" + ln.Addr().String() + "/"
	slog.Info("starting window", "url", target, "version", release.Version())

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	// Either side finishing cancels the other. A nil return has to stop the peer.
	serveErr := make(chan error, 1)
	go func() {
		err := (httpRun{
			server:    newHTTPServer(ln.Addr().String(), in.srv.Handler()),
			ln:        ln,
			closeHubs: in.srv.CloseHubs,
		}).run(ctx)
		serveErr <- err
		cancel()
	}()

	err = (app.App{
		Title:   appTitle,
		Width:   appWidth,
		Height:  appHeight,
		Handler: app.Web(windowPage(target)),
	}).Run(ctx)
	cancel()
	return errors.Join(err, <-serveErr)
}

// windowPage sends the web view to the loopback server.
func windowPage(target string) http.Handler {
	quoted := strconv.Quote(target)
	body := "<!doctype html><meta charset=utf-8><title>Superfolha</title><script>location.replace(" + quoted + ")</script><p><a href=" + quoted + ">Superfolha</a></p>"
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		if _, err := io.WriteString(w, body); err != nil {
			return
		}
	})
}

func headlessHost() bool {
	return envOn("LEWKIT_NO_UI") || envOn("ELETROCROMO_NO_UI")
}

func envOn(key string) bool {
	value := strings.TrimSpace(os.Getenv(key))
	return value == "1" || strings.EqualFold(value, "true") || strings.EqualFold(value, "yes")
}
