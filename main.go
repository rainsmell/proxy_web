package main

import (
	"context"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

var appVersion = "dev"

type Node struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Protocol    string    `json:"protocol"`
	Address     string    `json:"address"`
	Port        int       `json:"port"`
	UUID        string    `json:"uuid,omitempty"`
	Password    string    `json:"password,omitempty"`
	Method      string    `json:"method,omitempty"`
	AlterID     int       `json:"alterId,omitempty"`
	Network     string    `json:"network,omitempty"`
	TLS         bool      `json:"tls,omitempty"`
	SNI         string    `json:"sni,omitempty"`
	Host        string    `json:"host,omitempty"`
	Path        string    `json:"path,omitempty"`
	Flow        string    `json:"flow,omitempty"`
	Fingerprint string    `json:"fingerprint,omitempty"`
	Raw         string    `json:"raw,omitempty"`
	TCPDelay    int64     `json:"tcpDelay,omitempty"`
	ProxyDelay  int64     `json:"proxyDelay,omitempty"`
	TestError   string    `json:"testError,omitempty"`
	TestedAt    time.Time `json:"testedAt,omitempty"`
}
type Subscription struct {
	ID        string    `json:"id"`
	URL       string    `json:"url"`
	Name      string    `json:"name"`
	Nodes     []Node    `json:"nodes"`
	UpdatedAt time.Time `json:"updatedAt"`
}
type Config struct {
	ListenAddress  string         `json:"listenAddress"`
	SocksPort      int            `json:"socksPort"`
	HTTPPort       int            `json:"httpPort"`
	DomainStrategy string         `json:"domainStrategy"`
	LogLevel       string         `json:"logLevel"`
	Subscriptions  []Subscription `json:"subscriptions"`
}
type State struct {
	Running      bool      `json:"running"`
	PID          int       `json:"pid,omitempty"`
	NodeID       string    `json:"nodeId,omitempty"`
	NodeName     string    `json:"nodeName,omitempty"`
	Error        string    `json:"error,omitempty"`
	SocksPort    int       `json:"socksPort,omitempty"`
	HTTPPort     int       `json:"httpPort,omitempty"`
	ConfigPath   string    `json:"configPath,omitempty"`
	ConfigFormat string    `json:"configFormat,omitempty"`
	Version      string    `json:"version,omitempty"`
	StartedAt    time.Time `json:"startedAt,omitempty"`
}
type Connection struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
	Status string `json:"status"`
	Detour string `json:"detour,omitempty"`
	Time   string `json:"time"`
}
type TestProgress struct {
	ID          string `json:"id"`
	Total, Done int    `json:"total"`
	Running     bool   `json:"running"`
	Error       string `json:"error,omitempty"`
}

var cfg = Config{ListenAddress: "0.0.0.0", SocksPort: 10808, HTTPPort: 10809, DomainStrategy: "AsIs", LogLevel: "info"}
var dataDir = "data"
var storeMu, procMu, progressMu sync.Mutex
var running *exec.Cmd
var state State
var progress = map[string]TestProgress{}

func main() {
	dataDir = resolveDataDir()
	installSignalHandler()
	_ = os.MkdirAll(filepath.Join(dataDir, "runtime"), 0755)
	loadConfig()
	handler := newHTTPHandler()
	if err := serveHTTP(handler); err != nil {
		log.Fatal(err)
	}
}

func newHTTPHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/", api)
	mux.HandleFunc("/healthz", healthz)
	mux.HandleFunc("/", index)
	return logging(stripBasePath(mux))
}

func serveHTTP(handler http.Handler) error {
	if sock := resolveUnixSocket(); sock != "" {
		_ = os.Remove(sock)
		l, err := net.Listen("unix", sock)
		if err != nil {
			return err
		}
		_ = os.Chmod(sock, 0666)
		log.Printf("web server listening on unix socket %s, basePath=%s, dataDir=%s, appRoot=%s", sock, gatewayBasePath(), dataDir, appRoot())
		return http.Serve(l, handler)
	}

	addr := resolveListenAddr()
	log.Printf("web server listening on %s, basePath=%s, dataDir=%s, appRoot=%s", addr, gatewayBasePath(), dataDir, appRoot())
	return http.ListenAndServe(addr, handler)
}

func resolveDataDir() string {
	if v := strings.TrimSpace(os.Getenv("TRIM_PKGVAR")); v != "" {
		return v
	}
	if v := strings.TrimSpace(os.Getenv("V2RAY_WEB_DATA_DIR")); v != "" {
		return v
	}
	return filepath.Join(appRoot(), "data")
}

func resolveListenAddr() string {
	if v := strings.TrimSpace(os.Getenv("V2RAY_WEB_ADDR")); v != "" {
		return v
	}
	if p := strings.TrimSpace(os.Getenv("TRIM_SERVICE_PORT")); p != "" {
		return ":" + p
	}
	return ":8080"
}

func resolveUnixSocket() string {
	v := strings.TrimSpace(os.Getenv("V2RAY_WEB_UNIX_SOCKET"))
	if v == "" {
		return ""
	}
	if filepath.IsAbs(v) {
		return v
	}
	if dest := strings.TrimSpace(os.Getenv("TRIM_APPDEST")); dest != "" {
		return filepath.Join(dest, v)
	}
	return v
}

func gatewayBasePath() string {
	v := strings.TrimSpace(os.Getenv("V2RAY_WEB_BASE_PATH"))
	if v == "" || v == "/" {
		return ""
	}
	if !strings.HasPrefix(v, "/") {
		v = "/" + v
	}
	return strings.TrimRight(v, "/")
}

func stripBasePath(next http.Handler) http.Handler {
	base := gatewayBasePath()
	if base == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == base {
			r2 := r.Clone(r.Context())
			u := *r.URL
			u.Path = "/"
			r2.URL = &u
			next.ServeHTTP(w, r2)
			return
		}
		if strings.HasPrefix(r.URL.Path, base+"/") {
			r2 := r.Clone(r.Context())
			u := *r.URL
			u.Path = strings.TrimPrefix(r.URL.Path, base)
			if u.Path == "" {
				u.Path = "/"
			}
			r2.URL = &u
			next.ServeHTTP(w, r2)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func renderPageHTML() string {
	b, _ := json.Marshal(gatewayBasePath())
	inject := "<script>window.__BASE_PATH__=" + string(b) + ";</script>"
	return strings.Replace(pageHTML, "</head>", inject+"\n</head>", 1)
}

func installSignalHandler() {
	sigCh := make(chan os.Signal, 2)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Printf("shutdown signal received; stopping v2ray child process if running")
		stopProxyInternal()
		os.Exit(0)
	}()
}

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s (%s)", r.Method, r.URL.Path, time.Since(t).Round(time.Millisecond))
	})
}

func healthz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonOut(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	jsonOut(w, http.StatusOK, map[string]any{"ok": true, "version": appVersion, "time": time.Now()})
}

func index(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	io.WriteString(w, renderPageHTML())
}
func jsonOut(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
func bodyJSON(r *http.Request, v any) error {
	return json.NewDecoder(io.LimitReader(r.Body, 2<<20)).Decode(v)
}
func loadConfig() {
	b, e := os.ReadFile(filepath.Join(dataDir, "config.json"))
	if e == nil {
		_ = json.Unmarshal(b, &cfg)
	}
	if cfg.ListenAddress == "" {
		cfg.ListenAddress = "0.0.0.0"
	}
	if cfg.SocksPort == 0 {
		cfg.SocksPort = 10808
	}
	if cfg.HTTPPort == 0 {
		cfg.HTTPPort = 10809
	}
	if cfg.DomainStrategy == "" || cfg.DomainStrategy == "UseIPv4" {
		cfg.DomainStrategy = "AsIs"
	}
	if cfg.LogLevel == "" {
		cfg.LogLevel = "info"
	}
}
func saveConfig() error {
	storeMu.Lock()
	defer storeMu.Unlock()
	b, e := json.MarshalIndent(cfg, "", "  ")
	if e != nil {
		return e
	}
	return os.WriteFile(filepath.Join(dataDir, "config.json"), b, 0644)
}
func allNodes() []Node {
	var out []Node
	for _, s := range cfg.Subscriptions {
		out = append(out, s.Nodes...)
	}
	return out
}
func findNode(id string) (Node, bool) {
	for _, n := range allNodes() {
		if n.ID == id {
			return n, true
		}
	}
	return Node{}, false
}
func hash(s string) string { h := sha1.Sum([]byte(s)); return fmt.Sprintf("%x", h[:8]) }

func api(w http.ResponseWriter, r *http.Request) {
	p := strings.TrimPrefix(r.URL.Path, "/api/")
	switch {
	case p == "config" && r.Method == "GET":
		jsonOut(w, 200, cfg)
	case p == "config" && r.Method == "PUT":
		var c Config
		if bodyJSON(r, &c) != nil || c.SocksPort < 1 || c.SocksPort > 65535 || c.HTTPPort < 1 || c.HTTPPort > 65535 || !validDomainStrategy(c.DomainStrategy) || !validLogLevel(c.LogLevel) {
			jsonOut(w, 400, map[string]string{"error": "invalid config or port"})
			return
		}
		cfg.ListenAddress = c.ListenAddress
		cfg.SocksPort = c.SocksPort
		cfg.HTTPPort = c.HTTPPort
		cfg.DomainStrategy = c.DomainStrategy
		if cfg.DomainStrategy == "" || cfg.DomainStrategy == "UseIPv4" {
			cfg.DomainStrategy = "AsIs"
		}
		cfg.LogLevel = c.LogLevel
		if cfg.LogLevel == "" {
			cfg.LogLevel = "info"
		}
		_ = saveConfig()
		jsonOut(w, 200, cfg)
	case p == "subscriptions" && r.Method == "GET":
		jsonOut(w, 200, cfg.Subscriptions)
	case p == "subscriptions" && r.Method == "POST":
		var x struct{ URL, Name string }
		if bodyJSON(r, &x) != nil {
			jsonOut(w, 400, map[string]string{"error": "invalid body"})
			return
		}
		u, e := url.Parse(strings.TrimSpace(x.URL))
		if e != nil || u.Scheme != "http" && u.Scheme != "https" {
			jsonOut(w, 400, map[string]string{"error": "valid http(s) URL required"})
			return
		}
		s := Subscription{ID: hash(x.URL), URL: x.URL, Name: x.Name}
		if s.Name == "" {
			s.Name = x.URL
		}
		log.Printf("subscription add requested: name=%q url=%q", s.Name, s.URL)
		for _, old := range cfg.Subscriptions {
			if old.ID == s.ID {
				log.Printf("subscription already exists: id=%s nodes=%d", old.ID, len(old.Nodes))
				jsonOut(w, 200, old)
				return
			}
		}
		cfg.Subscriptions = append(cfg.Subscriptions, s)
		_ = saveConfig()
		jsonOut(w, 201, s)
	case strings.HasPrefix(p, "subscriptions/") && strings.HasSuffix(p, "/refresh") && r.Method == "POST":
		refresh(w, strings.TrimSuffix(strings.TrimPrefix(p, "subscriptions/"), "/refresh"))
	case strings.HasPrefix(p, "subscriptions/") && r.Method == "DELETE":
		deleteSubscription(w, strings.TrimPrefix(p, "subscriptions/"))
	case p == "nodes" && r.Method == "GET":
		jsonOut(w, 200, allNodes())
	case p == "nodes/test-all" && r.Method == "POST":
		startTestAll(w)
	case strings.HasPrefix(p, "nodes/test-progress") && r.Method == "GET":
		id := r.URL.Query().Get("id")
		progressMu.Lock()
		v, ok := progress[id]
		progressMu.Unlock()
		if !ok {
			jsonOut(w, 404, map[string]string{"error": "task not found"})
		} else {
			jsonOut(w, 200, v)
		}
	case strings.HasPrefix(p, "nodes/") && strings.HasSuffix(p, "/test") && r.Method == "POST":
		testOne(w, strings.TrimSuffix(strings.TrimPrefix(p, "nodes/"), "/test"))
	case p == "connections" && r.Method == "GET":
		jsonOut(w, 200, readConnections())
	case p == "proxy/status" && r.Method == "GET":
		procMu.Lock()
		s := state
		procMu.Unlock()
		jsonOut(w, 200, s)
	case p == "proxy/logs" && r.Method == "GET":
		jsonOut(w, 200, tailLog("error.log", 100))
	case p == "proxy/start" && r.Method == "POST":
		startProxy(w, r)
	case p == "proxy/stop" && r.Method == "POST":
		stopProxy(w)
	default:
		jsonOut(w, 404, map[string]string{"error": "not found"})
	}
}

func validDomainStrategy(v string) bool {
	switch v {
	case "", "AsIs", "UseIP", "UseIPv4", "UseIPv6":
		return true
	default:
		return false
	}
}

func validLogLevel(v string) bool {
	switch v {
	case "", "debug", "info", "warning", "error", "none":
		return true
	default:
		return false
	}
}

func deleteSubscription(w http.ResponseWriter, id string) {
	for i, s := range cfg.Subscriptions {
		if s.ID == id {
			cfg.Subscriptions = append(cfg.Subscriptions[:i], cfg.Subscriptions[i+1:]...)
			_ = saveConfig()
			jsonOut(w, 200, map[string]bool{"ok": true})
			return
		}
	}
	jsonOut(w, 404, map[string]string{"error": "not found"})
}
func refresh(w http.ResponseWriter, id string) {
	var s *Subscription
	for i := range cfg.Subscriptions {
		if cfg.Subscriptions[i].ID == id {
			s = &cfg.Subscriptions[i]
			break
		}
	}
	if s == nil {
		jsonOut(w, 404, map[string]string{"error": "not found"})
		return
	}
	log.Printf("subscription refresh started: id=%s name=%q url=%q", s.ID, s.Name, s.URL)
	ctx, c := context.WithTimeout(context.Background(), 25*time.Second)
	defer c()
	req, _ := http.NewRequestWithContext(ctx, "GET", s.URL, nil)
	req.Header.Set("User-Agent", "v2ray-web/1.0")
	resp, e := http.DefaultClient.Do(req)
	if e != nil {
		log.Printf("subscription refresh failed: id=%s error=%v", s.ID, e)
		jsonOut(w, 502, map[string]string{"error": e.Error()})
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		log.Printf("subscription refresh failed: id=%s status=%s", s.ID, resp.Status)
		jsonOut(w, 502, map[string]string{"error": resp.Status})
		return
	}
	b, e := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if e != nil {
		log.Printf("subscription refresh failed: id=%s read_error=%v", s.ID, e)
		jsonOut(w, 502, map[string]string{"error": e.Error()})
		return
	}
	s.Nodes = parseSubscription(string(b))
	s.UpdatedAt = time.Now()
	_ = saveConfig()
	log.Printf("subscription refresh completed: id=%s nodes=%d bytes=%d", s.ID, len(s.Nodes), len(b))
	jsonOut(w, 200, map[string]any{"count": len(s.Nodes), "nodes": s.Nodes})
}

func parseSubscription(raw string) []Node {
	raw = strings.TrimSpace(raw)
	compact := strings.Join(strings.Fields(raw), "")
	if d, e := base64.StdEncoding.DecodeString(compact); e == nil && strings.Contains(string(d), "://") {
		raw = string(d)
	}
	var out []Node
	for _, line := range strings.Split(strings.ReplaceAll(raw, "\r", ""), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var n Node
		var e error
		switch {
		case strings.HasPrefix(line, "vmess://"):
			n, e = parseVMess(line)
		case strings.HasPrefix(line, "vless://"):
			n, e = parseVLESS(line)
		case strings.HasPrefix(line, "trojan://"):
			n, e = parseTrojan(line)
		case strings.HasPrefix(line, "ss://"):
			n, e = parseSS(line)
		default:
			continue
		}
		if e == nil && n.Address != "" && n.Port > 0 {
			n.ID = hash(line)
			n.Raw = line
			out = append(out, n)
		}
	}
	return out
}
func parseVMess(s string) (Node, error) {
	b, e := base64.RawStdEncoding.DecodeString(strings.TrimPrefix(s, "vmess://"))
	if e != nil {
		b, e = base64.StdEncoding.DecodeString(strings.TrimPrefix(s, "vmess://"))
	}
	if e != nil {
		return Node{}, e
	}
	var x struct {
		PS, Add, ID, Net, TLS, SNI, Host, Path string
		Port                                   any
		Aid                                    int
		V                                      string
	}
	if e = json.Unmarshal(b, &x); e != nil {
		return Node{}, e
	}
	return Node{Name: x.PS, Protocol: "vmess", Address: x.Add, Port: toInt(x.Port), UUID: x.ID, AlterID: x.Aid, Network: x.Net, TLS: x.TLS != "", SNI: x.SNI, Host: x.Host, Path: x.Path}, nil
}
func parseVLESS(s string) (Node, error) {
	u, e := url.Parse(s)
	if e != nil {
		return Node{}, e
	}
	q := u.Query()
	p, _ := strconv.Atoi(u.Port())
	return Node{Name: u.Fragment, Protocol: "vless", Address: u.Hostname(), Port: p, UUID: u.User.Username(), Network: q.Get("type"), TLS: q.Get("security") == "tls" || q.Get("security") == "reality", SNI: q.Get("sni"), Host: q.Get("host"), Path: q.Get("path"), Flow: q.Get("flow"), Fingerprint: q.Get("fp")}, nil
}
func parseTrojan(s string) (Node, error) {
	u, e := url.Parse(s)
	if e != nil {
		return Node{}, e
	}
	q := u.Query()
	p, _ := strconv.Atoi(u.Port())
	return Node{Name: u.Fragment, Protocol: "trojan", Address: u.Hostname(), Port: p, Password: u.User.Username(), Network: q.Get("type"), TLS: q.Get("security") != "none", SNI: q.Get("sni"), Host: q.Get("host"), Path: q.Get("path"), Fingerprint: q.Get("fp")}, nil
}
func parseSS(s string) (Node, error) {
	u, e := url.Parse(s)
	if e != nil {
		return Node{}, e
	}
	p, _ := strconv.Atoi(u.Port())
	user := u.User.Username()
	pass, _ := u.User.Password()
	if pass == "" {
		if b, d := base64.RawStdEncoding.DecodeString(user); d == nil {
			z := strings.SplitN(string(b), ":", 2)
			if len(z) == 2 {
				user, pass = z[0], z[1]
			}
		}
	}
	return Node{Name: u.Fragment, Protocol: "shadowsocks", Address: u.Hostname(), Port: p, Method: user, Password: pass}, nil
}
func toInt(v any) int {
	switch x := v.(type) {
	case float64:
		return int(x)
	case string:
		i, _ := strconv.Atoi(x)
		return i
	}
	return 0
}

func testOne(w http.ResponseWriter, id string) {
	n, ok := findNode(id)
	if !ok {
		jsonOut(w, 404, map[string]string{"error": "not found"})
		return
	}
	testNode(&n)
	updateNode(n)
	jsonOut(w, 200, n)
}
func startTestAll(w http.ResponseWriter) {
	nodes := allNodes()
	id := hash(fmt.Sprint(time.Now().UnixNano()))
	progressMu.Lock()
	progress[id] = TestProgress{ID: id, Total: len(nodes), Running: true}
	p := progress[id]
	progressMu.Unlock()
	go func() {
		sem := make(chan struct{}, 4)
		var wg sync.WaitGroup
		for _, n := range nodes {
			wg.Add(1)
			go func(n Node) {
				defer wg.Done()
				sem <- struct{}{}
				testNode(&n)
				<-sem
				updateNode(n)
				progressMu.Lock()
				x := progress[id]
				x.Done++
				progress[id] = x
				progressMu.Unlock()
			}(n)
		}
		wg.Wait()
		progressMu.Lock()
		x := progress[id]
		x.Running = false
		progress[id] = x
		progressMu.Unlock()
	}()
	jsonOut(w, 202, p)
}
func testNode(n *Node) {
	st := time.Now()
	c, e := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", n.Address, n.Port), 10*time.Second)
	if e == nil {
		c.Close()
		n.TCPDelay = time.Since(st).Milliseconds()
	} else {
		n.TestError = e.Error()
	}
	n.TestedAt = time.Now()
}
func updateNode(n Node) {
	for si := range cfg.Subscriptions {
		for ni := range cfg.Subscriptions[si].Nodes {
			if cfg.Subscriptions[si].Nodes[ni].ID == n.ID {
				cfg.Subscriptions[si].Nodes[ni] = n
			}
		}
	}
	_ = saveConfig()
}

func readConnections() []Connection {
	b, e := os.ReadFile(filepath.Join(dataDir, "runtime", "access.log"))
	if e != nil {
		return []Connection{}
	}
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	if len(lines) > 200 {
		lines = lines[len(lines)-200:]
	}
	out := make([]Connection, 0, len(lines))
	for i, line := range lines {
		line = strings.TrimSpace(line)
		pos := strings.Index(line, " accepted ")
		status := "accepted"
		if pos < 0 {
			pos = strings.Index(line, " rejected ")
			status = "rejected"
		}
		if pos < 0 {
			continue
		}
		prefix, rest := line[:pos], line[pos+len(" accepted "):]
		if status == "rejected" {
			rest = line[pos+len(" rejected "):]
		}
		out = append(out, Connection{ID: hash(fmt.Sprintf("%d-%s", i, line)), Source: prefix, Target: rest, Status: status})
	}
	return out
}
func tailLog(name string, n int) []string {
	b, e := os.ReadFile(filepath.Join(dataDir, "runtime", name))
	if e != nil {
		return []string{}
	}
	x := strings.Split(strings.TrimSpace(string(b)), "\n")
	if len(x) > n {
		x = x[len(x)-n:]
	}
	return x
}
