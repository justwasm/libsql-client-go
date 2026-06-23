package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
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

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		cmd.Process.Signal(syscall.SIGTERM)
		cmd.Wait()
		os.Exit(0)
	}()
	go func() {
		cmd.Wait()
		os.Exit(0)
	}()

	target, err := url.Parse("http://" + sqldHTTP)
	if err != nil {
		log.Fatalf("parse sqld url: %s", err)
	}

	gw := &gateway{
		adminURL: "http://" + sqldAdmin,
		created:  map[string]bool{},
		proxy: &httputil.ReverseProxy{
			Rewrite: func(r *httputil.ProxyRequest) {
				ns, rest := extractNamespacePath(r.In.URL.Path)
				r.SetURL(target)
				r.Out.URL.Path = rest
				r.Out.Host = fmt.Sprintf("%s.%s", ns, target.Host)
			},
		},
	}

	log.Printf("gateway listening on %s", listen)
	log.Printf("sqld:  %s", gw.adminURL)
	log.Fatal(http.ListenAndServe(listen, gw))
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

type gateway struct {
	adminURL string
	created  map[string]bool
	mu       sync.Mutex
	proxy    *httputil.ReverseProxy
}

func (g *gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ns, _ := extractNamespacePath(r.URL.Path)
	if ns == "" {
		http.Error(w, "missing namespace in path: /<ns>/v2/pipeline", http.StatusBadRequest)
		return
	}

	if err := g.ensureNamespace(ns); err != nil {
		log.Printf("ERROR ensure namespace %s: %s", ns, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	g.proxy.ServeHTTP(w, r)
}

// extractNamespacePath splits "/ns/v2/pipeline" → ("ns", "/v2/pipeline").
// Returns ("", path) if no namespace segment is found.
func extractNamespacePath(path string) (string, string) {
	path = strings.TrimPrefix(path, "/")
	idx := strings.IndexByte(path, '/')
	if idx < 0 {
		return path, "/"
	}
	if idx == 0 {
		return "", path
	}
	return path[:idx], "/" + path[idx+1:]
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
	resp, err := http.DefaultClient.Do(req)
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
