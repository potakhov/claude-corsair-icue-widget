package cmds

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"xeneoncc/internal/config"
	"xeneoncc/internal/server"
	"xeneoncc/internal/store"
)

// Serve runs the bridge in the foreground until interrupted (Ctrl-C).
func Serve() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	return ServeContext(ctx)
}

// ServeContext runs the bridge until ctx is cancelled or a listener fails
// fatally, then shuts the HTTP servers down gracefully.
func ServeContext(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	handler := server.New(store.New(func() int64 { return time.Now().Unix() }), cfg.Token)

	addrs := []string{
		fmt.Sprintf("127.0.0.1:%d", cfg.Port),
		fmt.Sprintf("[::1]:%d", cfg.Port),
	}
	var servers []*http.Server
	errc := make(chan error, len(addrs))
	started := 0
	for _, addr := range addrs {
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			log.Printf("warn: cannot listen on %s: %v", addr, err)
			continue
		}
		started++
		srv := &http.Server{Handler: handler}
		servers = append(servers, srv)
		log.Printf("listening on http://%s", addr)
		go func(l net.Listener) { errc <- srv.Serve(l) }(ln)
	}
	if started == 0 {
		return fmt.Errorf("could not bind any loopback address on port %d", cfg.Port)
	}
	p, _ := config.Path()
	log.Printf("xeneon-bridge ready on port %d (token in %s)", cfg.Port, p)

	var serveErr error
	select {
	case <-ctx.Done():
	case serveErr = <-errc:
	}
	sctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, srv := range servers {
		_ = srv.Shutdown(sctx)
	}
	if serveErr != nil && serveErr != http.ErrServerClosed {
		return serveErr
	}
	return nil
}
