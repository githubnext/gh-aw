package main

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/github/gh-aw/pkg/workflow"
)

func run() error {
	configPath := flag.String("config", "", "protected enrollment and compiled subscription JSON file")
	statePath := flag.String("state", "", "durable state file on a private local filesystem")
	listen := flag.String("listen", "127.0.0.1:8080", "loopback HTTP listener behind trusted HTTPS termination")
	check := flag.Bool("check", false, "validate configuration and server-side key, then exit")
	manifestPath := flag.String("subscription-digest", "", "print local compiled subscription digest without granting approval")
	flag.Parse()
	if *manifestPath != "" {
		return printSubscriptionDigest(*manifestPath)
	}
	if *configPath == "" || *statePath == "" {
		return errors.New("-config and -state are required")
	}
	if _, err := readConfig(*configPath); err != nil {
		return err
	}
	key, err := loadKey(os.Getenv("GH_AW_GATEWAY_PRIVATE_KEY_FILE"))
	if err != nil {
		return err
	}
	if *check {
		return nil
	}
	return startGateway(*configPath, *statePath, *listen, key)
}

func printSubscriptionDigest(manifestPath string) error {
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		return errors.New("cannot read local compiled workflow")
	}
	manifest, err := workflow.ExtractGHAWManifestFromLockFile(string(raw))
	if err != nil || manifest == nil || len(manifest.WebhookSubscriptions) == 0 {
		return errors.New("local workflow has no valid compiled subscriptions")
	}
	subscriptions, err := json.Marshal(manifest.WebhookSubscriptions)
	if err != nil {
		return errors.New("cannot serialize compiled subscriptions")
	}
	fmt.Fprintln(os.Stdout, digest(subscriptions))
	return nil
}

func startGateway(configPath, statePath, listen string, key *rsa.PrivateKey) (result error) {
	host, _, err := net.SplitHostPort(listen)
	if err != nil || net.ParseIP(host) == nil || !net.ParseIP(host).IsLoopback() {
		return errors.New("listener must be an explicit loopback IP; configure trusted HTTPS termination")
	}
	store, err := openStore(statePath)
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, store.close()) }()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	gateway := &Gateway{ConfigPath: configPath, Store: store, Now: time.Now,
		GitHub: &GitHub{Key: key, Client: &http.Client{}, Now: time.Now}}
	server := &http.Server{Addr: listen, Handler: gateway, ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 16 * 1024}
	return serveGateway(ctx, stop, gateway, server)
}

func serveGateway(ctx context.Context, stop context.CancelFunc, gateway *Gateway, server *http.Server) error {
	workerErr := make(chan error, 1)
	workerDone := make(chan struct{})
	go func() {
		defer func() {
			if recover() != nil {
				workerErr <- errors.New("dispatch worker panicked; in-flight work requires reconciliation")
			}
		}()
		defer close(workerDone)
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := gateway.process(ctx); err != nil {
					workerErr <- err
					return
				}
			}
		}
	}()
	serverErr := make(chan error, 1)
	go func() {
		defer func() {
			if recover() != nil {
				serverErr <- errors.New("HTTP server panicked")
			}
		}()
		serverErr <- server.ListenAndServe()
	}()
	var result error
	select {
	case <-ctx.Done():
	case result = <-workerErr:
	case result = <-serverErr:
	}
	stop()
	shutdown, cancel := context.WithTimeout(context.WithoutCancel(ctx), 20*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdown); err != nil {
		return errors.New("HTTP server shutdown failed")
	}
	select {
	case <-workerDone:
	case <-shutdown.Done():
		return errors.New("dispatch worker cleanup did not complete")
	}
	if errors.Is(result, http.ErrServerClosed) {
		return nil
	}
	return result
}

func main() {
	if err := run(); err != nil {
		// All operational errors are fixed, credential-safe messages.
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
