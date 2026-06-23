package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
)

func main() {
	sqldHTTP := env("SQLD_HTTP_LISTEN_ADDR", "127.0.0.1:8080")
	sqldAdmin := env("SQLD_ADMIN_LISTEN_ADDR", "127.0.0.1:8082")
	sqldDB := env("SQLD_DB_PATH", "data.sqld")
	listen := env("LISTEN", ":9090")

	// Start sqld as a subprocess.  If the gateway dies (even SIGKILL),
	// the child sqld dies with it.
	cmd := exec.Command("sqld",
		"--enable-namespaces",
		"--db-path", sqldDB,
		"--http-listen-addr", sqldHTTP,
		"--admin-listen-addr", sqldAdmin,
	)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		log.Fatalf("start sqld: %s", err)
	}

	// Kill sqld on exit.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		cmd.Process.Signal(syscall.SIGTERM)
		cmd.Wait()
		os.Exit(0)
	}()

	// Kill sqld when the gateway is killed with SIGKILL too.
	go func() {
		cmd.Wait()
		os.Exit(0)
	}()

	gw := &gateway{
		sqldURL:   "http://" + sqldHTTP,
		adminURL:  "http://" + sqldAdmin,
		created:   map[string]bool{},
		adminHTTP: &http.Client{},
		proxyHTTP: &http.Client{},
	}

	log.Printf("gateway listening on %s", listen)
	log.Printf("sqld:  %s", gw.sqldURL)
	log.Printf("admin: %s", gw.adminURL)
	log.Fatal(http.ListenAndServe(listen, gw))
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

type gateway struct {
	sqldURL   string
	adminURL  string
	created   map[string]bool
	mu        sync.Mutex
	adminHTTP *http.Client
	proxyHTTP *http.Client
}

func (g *gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ns := extractNamespace(r.Host)
	if ns == "" {
		http.Error(w, "missing namespace in Host header", http.StatusBadRequest)
		return
	}

	if err := g.ensureNamespace(ns); err != nil {
		log.Printf("ERROR ensure namespace %s: %s", ns, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Proxy to sqld, preserving the original Host header so sqld
	// can extract the namespace from the subdomain.
	target, _ := url.Parse(g.sqldURL)
	originalHost := r.Host
	r.URL.Host = target.Host
	r.URL.Scheme = target.Scheme
	r.RequestURI = ""
	r.Host = originalHost

	resp, err := g.proxyHTTP.Do(r)
	if err != nil {
		log.Printf("ERROR proxy: %s", err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func (g *gateway) ensureNamespace(ns string) error {
	g.mu.Lock()
	if g.created[ns] {
		g.mu.Unlock()
		return nil
	}
	g.mu.Unlock()

	body := bytes.NewReader([]byte(`{}`))
	u := fmt.Sprintf("%s/v1/namespaces/%s/create", g.adminURL, ns)
	req, err := http.NewRequest("POST", u, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := g.adminHTTP.Do(req)
	if err != nil {
		return fmt.Errorf("admin call failed: %w", err)
	}
	resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 400 && resp.StatusCode != 409 {
		return fmt.Errorf("admin returned %d", resp.StatusCode)
	}

	g.mu.Lock()
	g.created[ns] = true
	g.mu.Unlock()
	return nil
}

func extractNamespace(host string) string {
	host = strings.Split(host, ":")[0]
	if idx := strings.IndexByte(host, '.'); idx > 0 {
		if host[idx+1:] != "" {
			return host[:idx]
		}
	}
	return ""
}
