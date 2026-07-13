package cmds

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"xeneoncc/internal/config"
	"xeneoncc/internal/server"
	"xeneoncc/internal/store"
)

// bindAddrs returns the TCP addresses ServeContext listens on. Empty bind =
// loopback only (private default). "0.0.0.0"/"::" is a wildcard used alone (it
// already covers loopback). A specific IP is exposed AND paired with loopback so
// the local widget/statusline keep reaching the bridge on 127.0.0.1.
func bindAddrs(bind string, port int) []string {
	switch bind {
	case "":
		return []string{fmt.Sprintf("127.0.0.1:%d", port), fmt.Sprintf("[::1]:%d", port)}
	case "0.0.0.0":
		// All IPv4 (covers 127.0.0.1) plus the IPv6 loopback, so the local widget's
		// default "localhost" target keeps a listener whether it resolves to 127.0.0.1
		// or ::1 — otherwise enabling remote drops the [::1] listener the default has.
		return []string{fmt.Sprintf("0.0.0.0:%d", port), fmt.Sprintf("[::1]:%d", port)}
	case "::", "[::]":
		return []string{fmt.Sprintf("[::]:%d", port)}
	default:
		host := bind
		if strings.Contains(host, ":") && !strings.HasPrefix(host, "[") { // bare IPv6 literal
			host = "[" + host + "]"
		}
		return []string{fmt.Sprintf("%s:%d", host, port), fmt.Sprintf("127.0.0.1:%d", port)}
	}
}

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

	addrs := bindAddrs(cfg.Bind, cfg.Port)
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
		return fmt.Errorf("could not bind any address on port %d", cfg.Port)
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
