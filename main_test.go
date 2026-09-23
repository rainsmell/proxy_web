package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func TestParseVLESS(t *testing.T) {
	n, err := parseVLESS("vless://11111111-1111-1111-1111-111111111111@example.com:443?type=ws&security=tls&sni=cdn.example.com&host=cdn.example.com&path=%2Fws#demo")
	if err != nil {
		t.Fatal(err)
	}
	if n.Protocol != "vless" || n.Port != 443 || n.Network != "ws" || !n.TLS || n.Path != "/ws" {
		t.Fatalf("unexpected node: %+v", n)
	}
	if n.Name != "demo" || n.SNI != "cdn.example.com" {
		t.Fatalf("unexpected metadata: %+v", n)
	}
}

func TestParseVLESSReality(t *testing.T) {
	n, err := parseVLESS("vless://uuid-1@example.com:8443?type=tcp&security=reality&sni=www.microsoft.com&fp=chrome&pbk=PUBKEY&sid=abcd&flow=xtls-rprx-vision&alpn=h2%2Chttp%2F1.1#reality")
	if err != nil {
		t.Fatal(err)
	}
	if !n.TLS || n.PublicKey != "PUBKEY" || n.ShortID != "abcd" || n.Flow != "xtls-rprx-vision" {
		t.Fatalf("reality fields missing: %+v", n)
	}
	if n.ALPN != "h2,http/1.1" {
		t.Fatalf("unexpected alpn: %q", n.ALPN)
	}
}

func TestParseVMess(t *testing.T) {
	payload := `{"v":"2","ps":"vmess node","add":"example.com","port":"443","id":"uuid-2","aid":"0","scy":"auto","net":"ws","type":"none","host":"cdn.example.com","path":"/vm","tls":"tls","sni":"cdn.example.com","fp":"chrome","alpn":"h2"}`
	n, err := parseVMess("vmess://" + base64.StdEncoding.EncodeToString([]byte(payload)))
	if err != nil {
		t.Fatal(err)
	}
	if n.Protocol != "vmess" || !n.TLS || n.Security != "auto" || n.Path != "/vm" || n.Fingerprint != "chrome" {
		t.Fatalf("unexpected vmess node: %+v", n)
	}
}

func TestParseTrojanWS(t *testing.T) {
	n, err := parseTrojan("trojan://secret@example.com:443?type=ws&security=tls&sni=example.com&host=example.com&path=%2Ftj&allowInsecure=1#tj")
	if err != nil {
		t.Fatal(err)
	}
	if n.Protocol != "trojan" || n.Password != "secret" || !n.TLS || !n.SkipCertVerify || n.Path != "/tj" {
		t.Fatalf("unexpected trojan node: %+v", n)
	}
}

func TestParseHysteria2(t *testing.T) {
	n, err := parseHysteria2("hysteria2://pass@example.com:443?sni=example.com&insecure=1&obfs=salamander&obfs-password=obfspass&alpn=h3&upmbps=30&downmbps=200#hy2")
	if err != nil {
		t.Fatal(err)
	}
	if n.Protocol != "hysteria2" || n.Password != "pass" || n.Obfs != "salamander" || n.ObfsPassword != "obfspass" {
		t.Fatalf("unexpected hysteria2 node: %+v", n)
	}
	if n.UpMbps != 30 || n.DownMbps != 200 || !n.SkipCertVerify {
		t.Fatalf("unexpected hysteria2 numbers: %+v", n)
	}
}

func TestParseTUICV5(t *testing.T) {
	n, err := parseTUIC("tuic://11111111-1111-1111-1111-111111111111:pw@example.com:443?congestion_control=bbr&udp_relay_mode=native&alpn=h3&sni=example.com#tuic")
	if err != nil {
		t.Fatal(err)
	}
	if n.Protocol != "tuic" || n.UUID != "11111111-1111-1111-1111-111111111111" || n.Password != "pw" {
		t.Fatalf("unexpected tuic node: %+v", n)
	}
	if n.CongestionController != "bbr" || n.UDPRelayMode != "native" {
		t.Fatalf("unexpected tuic options: %+v", n)
	}
}

func TestParseSSR(t *testing.T) {
	pass := base64.RawURLEncoding.EncodeToString([]byte("password"))
	body := "example.com:443:auth_sha1_v4:aes-256-cfb:tls1.2_ticket_auth:" + pass + "/?remarks=" + base64.RawURLEncoding.EncodeToString([]byte("ssr node"))
	n, err := parseSSR("ssr://" + base64.RawURLEncoding.EncodeToString([]byte(body)))
	if err != nil {
		t.Fatal(err)
	}
	if n.Protocol != "ssr" || n.Address != "example.com" || n.Port != 443 || n.Method != "aes-256-cfb" {
		t.Fatalf("unexpected ssr node: %+v", n)
	}
	if n.Password != "password" || n.SSRProtocol != "auth_sha1_v4" || n.SSRObfs != "tls1.2_ticket_auth" || n.Name != "ssr node" {
		t.Fatalf("unexpected ssr fields: %+v", n)
	}
}

func TestParseClashYAML(t *testing.T) {
	doc := `
mixed-port: 7890
proxies:
  - name: "hy2 node"
    type: hysteria2
    server: hy.example.com
    port: 443
    password: secret
    sni: hy.example.com
    obfs: salamander
    obfs-password: obfspass
  - name: "reality node"
    type: vless
    server: vl.example.com
    port: 8443
    uuid: 11111111-1111-1111-1111-111111111111
    tls: true
    servername: www.microsoft.com
    reality-opts:
      public-key: PUBKEY
      short-id: abcd
    client-fingerprint: chrome
`
	nodes := parseClashYAML(doc)
	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(nodes))
	}
	if nodes[0].Protocol != "hysteria2" || nodes[0].Address != "hy.example.com" || len(nodes[0].Proxy) == 0 {
		t.Fatalf("unexpected first node: %+v", nodes[0])
	}
	if nodes[1].Protocol != "vless" || nodes[1].Port != 8443 {
		t.Fatalf("unexpected second node: %+v", nodes[1])
	}
}

func TestParseSubscriptionClashYAMLBase64(t *testing.T) {
	doc := "proxies:\n  - name: n1\n    type: ss\n    server: ss.example.com\n    port: 8388\n    cipher: aes-256-gcm\n    password: pw\n"
	encoded := base64.StdEncoding.EncodeToString([]byte(doc))
	nodes := parseSubscription(encoded)
	if len(nodes) != 1 || nodes[0].Protocol != "shadowsocks" || nodes[0].Method != "aes-256-gcm" {
		t.Fatalf("unexpected nodes: %+v", nodes)
	}
}

func TestParseSubscriptionShareLinksBase64(t *testing.T) {
	links := "vless://11111111-1111-1111-1111-111111111111@a.example.com:443?security=tls&sni=a.example.com#a\n" +
		"trojan://pw@b.example.com:443?security=tls&sni=b.example.com#b\n"
	encoded := base64.StdEncoding.EncodeToString([]byte(links))
	nodes := parseSubscription(encoded)
	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d: %+v", len(nodes), nodes)
	}
}

func TestBuildMihomoVLESSReality(t *testing.T) {
	n := Node{
		Protocol: "vless", Address: "example.com", Port: 8443,
		UUID: "11111111-1111-1111-1111-111111111111",
		TLS:  true, SNI: "www.microsoft.com", PublicKey: "gl6wkTv5bnZhBw29pGEqgimq39p74PcDwlUGBnQc3BM", ShortID: "abcd",
		Flow: "xtls-rprx-vision", Fingerprint: "ios", ALPN: "h2,http/1.1",
	}
	o := buildMihomoProxy(n)
	if o["type"] != "vless" || o["tls"] != true || o["servername"] != "www.microsoft.com" {
		t.Fatalf("unexpected proxy: %+v", o)
	}
	ro, ok := o["reality-opts"].(map[string]any)
	if !ok || ro["public-key"] != "gl6wkTv5bnZhBw29pGEqgimq39p74PcDwlUGBnQc3BM" || ro["short-id"] != "abcd" {
		t.Fatalf("reality opts missing: %+v", o)
	}
	if o["client-fingerprint"] != "ios" {
		t.Fatalf("fingerprint should be normalized to mihomo name: %+v", o)
	}
	alpn, ok := o["alpn"].([]string)
	if !ok || len(alpn) != 2 || alpn[0] != "h2" {
		t.Fatalf("unexpected alpn: %+v", o["alpn"])
	}
}

func TestBuildMihomoVMessWS(t *testing.T) {
	n := Node{Protocol: "vmess", Address: "example.com", Port: 443, UUID: "uuid", AlterID: 0, Security: "auto", Network: "ws", TLS: true, SNI: "cdn.example.com", Host: "cdn.example.com", Path: "/vm", Fingerprint: "chrome"}
	o := buildMihomoProxy(n)
	if o["type"] != "vmess" || o["alterId"] != 0 || o["cipher"] != "auto" || o["servername"] != "cdn.example.com" {
		t.Fatalf("unexpected vmess proxy: %+v", o)
	}
	ws, ok := o["ws-opts"].(map[string]any)
	if !ok || ws["path"] != "/vm" {
		t.Fatalf("ws opts missing: %+v", o)
	}
}

func TestBuildMihomoTrojanGRPC(t *testing.T) {
	n := Node{Protocol: "trojan", Address: "example.com", Port: 443, Password: "pw", Network: "grpc", Path: "svc", SNI: "example.com"}
	o := buildMihomoProxy(n)
	if o["tls"] != true || o["sni"] != "example.com" {
		t.Fatalf("trojan tls missing: %+v", o)
	}
	gs, ok := o["grpc-opts"].(map[string]any)
	if !ok || gs["grpc-service-name"] != "svc" {
		t.Fatalf("grpc opts missing: %+v", o)
	}
}

func TestBuildMihomoShadowsocks(t *testing.T) {
	o := buildMihomoProxy(Node{Protocol: "shadowsocks", Address: "example.com", Port: 8388, Method: "aes-256-gcm", Password: "pw"})
	if o["type"] != "ss" || o["cipher"] != "aes-256-gcm" || o["password"] != "pw" {
		t.Fatalf("unexpected ss proxy: %+v", o)
	}
}

func TestBuildMihomoHysteria2(t *testing.T) {
	o := buildMihomoProxy(Node{Protocol: "hysteria2", Address: "example.com", Port: 443, Password: "pw", SNI: "example.com", Obfs: "salamander", ObfsPassword: "op", UpMbps: 30, DownMbps: 200})
	if o["type"] != "hysteria2" || o["obfs"] != "salamander" || o["obfs-password"] != "op" {
		t.Fatalf("unexpected hysteria2 proxy: %+v", o)
	}
	if o["up"] != "30 Mbps" || o["down"] != "200 Mbps" {
		t.Fatalf("unexpected hysteria2 bandwidth: %+v", o)
	}
	if _, hasTLS := o["tls"]; hasTLS {
		t.Fatalf("hysteria2 must not set tls explicitly: %+v", o)
	}
}

func TestBuildMihomoProxyPassthrough(t *testing.T) {
	n := Node{Protocol: "tuic", Address: "example.com", Port: 443, Proxy: map[string]any{"type": "tuic", "token": "abc", "server": "example.com", "port": 443}}
	o := buildMihomoProxy(n)
	if o["name"] != "proxy" || o["token"] != "abc" {
		t.Fatalf("passthrough proxy not preserved: %+v", o)
	}
}

func TestNormalizeFingerprint(t *testing.T) {
	cases := map[string]string{"ios": "ios", "iOS": "ios", "chrome_auto": "chrome", "android_11_okhttp": "android", "randomized": "random", "": ""}
	for in, want := range cases {
		if got := normalizeFingerprint(in); got != want {
			t.Fatalf("normalizeFingerprint(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestBuildMihomoConfigIsValidYAML(t *testing.T) {
	old := cfg
	defer func() { cfg = old }()
	cfg.SocksPort = 10808
	cfg.HTTPPort = 10809
	cfg.ListenAddress = "0.0.0.0"
	cfg.LogLevel = "info"
	cfg.DomainStrategy = "AsIs"

	c := buildMihomoConfig(Node{Protocol: "vless", Address: "example.com", Port: 443, UUID: "uuid", TLS: true, SNI: "example.com"}, 19090, "secret")
	b, err := yaml.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	var back map[string]any
	if err := yaml.Unmarshal(b, &back); err != nil {
		t.Fatalf("generated config is not valid yaml: %v\n%s", err, b)
	}
	if back["socks-port"] != 10808 || back["port"] != 10809 {
		t.Fatalf("ports missing: %v", back)
	}
	if back["external-controller"] != "127.0.0.1:19090" || back["secret"] != "secret" {
		t.Fatalf("controller missing: %v", back)
	}
	rules, ok := back["rules"].([]any)
	if !ok || len(rules) != 1 || rules[0] != "MATCH,PROXY" {
		t.Fatalf("unexpected rules: %v", back["rules"])
	}
}

func TestReadConnectionsMapsMihomoSnapshot(t *testing.T) {
	const payload = `{"downloadTotal":10,"uploadTotal":5,"connections":[{"id":"abc","metadata":{"network":"tcp","type":"HTTP","sourceIP":"127.0.0.1","sourcePort":"5000","destinationIP":"1.2.3.4","destinationPort":"443","host":"example.com"},"upload":1,"download":2,"start":"2024-01-02T03:04:05+08:00","chains":["proxy","PROXY"],"rule":"Match"}]}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer testsecret" {
			t.Errorf("missing auth header: %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(payload))
	}))
	defer srv.Close()

	procMu.Lock()
	oldAddr, oldSecret := runtimeController, runtimeSecret
	runtimeController = strings.TrimPrefix(srv.URL, "http://")
	runtimeSecret = "testsecret"
	procMu.Unlock()
	defer func() {
		procMu.Lock()
		runtimeController, runtimeSecret = oldAddr, oldSecret
		procMu.Unlock()
	}()

	conns := readConnections()
	if len(conns) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(conns))
	}
	c := conns[0]
	if c.ID != "abc" || c.Source != "127.0.0.1:5000" || c.Target != "example.com:443" {
		t.Fatalf("unexpected connection: %+v", c)
	}
	if c.Status != "TCP" || c.Time != "03:04:05" || c.Detour != "proxy <- PROXY" {
		t.Fatalf("unexpected connection metadata: %+v", c)
	}
}

func TestStateJSONIncludesNodeName(t *testing.T) {
	b, err := json.Marshal(State{Running: true, NodeID: "b303d4bf9751ab8e", NodeName: "日本东京09", Error: "sample"})
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got["nodeName"] != "日本东京09" {
		t.Fatalf("nodeName missing or wrong: %s", b)
	}
	if got["error"] != "sample" {
		t.Fatalf("error missing or wrong: %s", b)
	}
}

func TestResolveDataDirPrefersFnOSEnv(t *testing.T) {
	t.Setenv("TRIM_PKGVAR", "/var/packages/v2rayweb/var")
	t.Setenv("V2RAY_WEB_DATA_DIR", "/tmp/ignored")
	if got := resolveDataDir(); got != "/var/packages/v2rayweb/var" {
		t.Fatalf("unexpected data dir: %q", got)
	}
}

func TestResolveListenAddrPrefersExplicitThenFnOSPort(t *testing.T) {
	t.Setenv("V2RAY_WEB_ADDR", ":18080")
	t.Setenv("TRIM_SERVICE_PORT", "8080")
	if got := resolveListenAddr(); got != ":18080" {
		t.Fatalf("explicit addr not respected: %q", got)
	}

	t.Setenv("V2RAY_WEB_ADDR", "")
	if got := resolveListenAddr(); got != ":8080" {
		t.Fatalf("fnOS port not respected: %q", got)
	}
}

func TestCorePathPrefersExplicitEnv(t *testing.T) {
	t.Setenv("MIHOMO_CORE_PATH", "/opt/mihomo/mihomo")
	t.Setenv("V2RAY_CORE_PATH", "")
	if got := corePath(); got != "/opt/mihomo/mihomo" {
		t.Fatalf("explicit mihomo core path not respected: %q", got)
	}
	t.Setenv("MIHOMO_CORE_PATH", "")
	t.Setenv("V2RAY_CORE_PATH", "/opt/legacy/v2ray")
	if got := corePath(); got != "/opt/legacy/v2ray" {
		t.Fatalf("legacy core path fallback not respected: %q", got)
	}
}

func TestGatewayBasePathHandler(t *testing.T) {
	t.Setenv("V2RAY_WEB_BASE_PATH", "/app/v2rayweb")
	r := httptest.NewRequest(http.MethodGet, "/app/v2rayweb/healthz", nil)
	w := httptest.NewRecorder()
	newHTTPHandler().ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", w.Code, w.Body.String())
	}
}

func TestRenderPageInjectsBasePath(t *testing.T) {
	t.Setenv("V2RAY_WEB_BASE_PATH", "/app/v2rayweb")
	page := renderPageHTML()
	if !strings.Contains(page, `window.__BASE_PATH__="/app/v2rayweb"`) {
		t.Fatalf("base path was not injected")
	}
}

func TestWriteRuntimeConfigAcceptedByBundledMihomo(t *testing.T) {
	core := bundledTestCore(t)
	if core == "" {
		return
	}
	oldCfg, oldDataDir := cfg, dataDir
	defer func() {
		cfg, dataDir = oldCfg, oldDataDir
	}()
	dataDir = t.TempDir()
	if err := os.MkdirAll(filepath.Join(dataDir, "runtime"), 0755); err != nil {
		t.Fatal(err)
	}
	cfg.ListenAddress = "127.0.0.1"
	cfg.SocksPort = 11808
	cfg.HTTPPort = 11809
	cfg.LogLevel = "info"
	cfg.DomainStrategy = "AsIs"

	n := Node{Protocol: "vless", Address: "example.com", Port: 443, UUID: "b22681d8-daad-4eb8-95bc-1078148358f9", Network: "ws", TLS: true, SNI: "example.com", Host: "example.com", Path: "/pq/jp5", Fingerprint: "ios"}
	if err := writeRuntime(n); err != nil {
		t.Fatal(err)
	}
	if runtimeConfigFormat != "yaml" {
		t.Fatalf("expected yaml format, got %q", runtimeConfigFormat)
	}
	cases := []struct {
		name string
		node Node
	}{
		{"vless-ws", Node{Protocol: "vless", Address: "example.com", Port: 443, UUID: "b22681d8-daad-4eb8-95bc-1078148358f9", Network: "ws", TLS: true, SNI: "example.com", Host: "example.com", Path: "/pq/jp5", Fingerprint: "ios"}},
		{"vless-reality", Node{Protocol: "vless", Address: "example.com", Port: 8443, UUID: "b22681d8-daad-4eb8-95bc-1078148358f9", Network: "tcp", TLS: true, SNI: "www.microsoft.com", PublicKey: "gl6wkTv5bnZhBw29pGEqgimq39p74PcDwlUGBnQc3BM", ShortID: "abcd", Flow: "xtls-rprx-vision", Fingerprint: "chrome", ALPN: "h2,http/1.1"}},
		{"vmess-ws", Node{Protocol: "vmess", Address: "example.com", Port: 443, UUID: "b22681d8-daad-4eb8-95bc-1078148358f9", Security: "auto", Network: "ws", TLS: true, SNI: "example.com", Host: "example.com", Path: "/vm"}},
		{"trojan-grpc", Node{Protocol: "trojan", Address: "example.com", Port: 443, Password: "pw", Network: "grpc", Path: "svc", SNI: "example.com"}},
		{"shadowsocks", Node{Protocol: "shadowsocks", Address: "example.com", Port: 8388, Method: "aes-256-gcm", Password: "pw"}},
		{"ssr", Node{Protocol: "ssr", Address: "example.com", Port: 443, Method: "aes-256-cfb", Password: "pw", SSRProtocol: "auth_sha1_v4", SSRObfs: "tls1.2_ticket_auth"}},
		{"hysteria2", Node{Protocol: "hysteria2", Address: "example.com", Port: 443, Password: "pw", SNI: "example.com", Obfs: "salamander", ObfsPassword: "op", UpMbps: 30, DownMbps: 200}},
		{"tuic", Node{Protocol: "tuic", Address: "example.com", Port: 443, UUID: "b22681d8-daad-4eb8-95bc-1078148358f9", Password: "pw", SNI: "example.com", CongestionController: "bbr", UDPRelayMode: "native"}},
	}
	for _, tc := range cases {
		if err := writeRuntime(tc.node); err != nil {
			t.Fatalf("%s: writeRuntime: %v", tc.name, err)
		}
		cmd := exec.Command(core, "-t", "-d", runtimeDirPath(), "-f", runtimeConfigPath())
		cmd.Dir = filepath.Dir(core)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%s: mihomo rejected generated config: %v\n%s", tc.name, err, out)
		}
	}
}

func TestMihomoProxySurvivesJSONRoundTrip(t *testing.T) {
	// Nodes imported from Clash YAML are persisted as JSON; numbers become
	// float64 after reload. The emitted YAML must still use integer ports.
	n := Node{Protocol: "hysteria2", Address: "x.example.com", Port: 443, Proxy: map[string]any{"type": "hysteria2", "server": "x.example.com", "port": 443, "password": "p"}}
	b, err := json.Marshal(n)
	if err != nil {
		t.Fatal(err)
	}
	var back Node
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	o := buildMihomoProxy(back)
	out, err := yaml.Marshal(map[string]any{"proxies": []any{o}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "port: 443") {
		t.Fatalf("port was not emitted as an integer:\n%s", out)
	}
}

func TestStartProxyKeepsControllerCredentials(t *testing.T) {
	core := bundledTestCore(t)
	if core == "" {
		return
	}
	oldCfg, oldDataDir, oldRunning, oldState := cfg, dataDir, running, state
	defer func() {
		stopProxyInternal()
		cfg, dataDir, running, state = oldCfg, oldDataDir, oldRunning, oldState
		procMu.Lock()
		runtimeController, runtimeSecret = "", ""
		procMu.Unlock()
	}()
	dataDir = t.TempDir()
	if err := os.MkdirAll(filepath.Join(dataDir, "runtime"), 0755); err != nil {
		t.Fatal(err)
	}
	socksPort, err := freeLocalPort()
	if err != nil {
		t.Fatal(err)
	}
	httpPort, err := freeLocalPort()
	if err != nil {
		t.Fatal(err)
	}
	node := Node{ID: "n1", Protocol: "vless", Address: "example.com", Port: 443, UUID: "b22681d8-daad-4eb8-95bc-1078148358f9", TLS: true, SNI: "example.com"}
	cfg = Config{ListenAddress: "127.0.0.1", SocksPort: socksPort, HTTPPort: httpPort, LogLevel: "info", DomainStrategy: "AsIs", Subscriptions: []Subscription{{ID: "s1", Nodes: []Node{node}}}}

	req := httptest.NewRequest(http.MethodPost, "/api/proxy/start", strings.NewReader(`{"nodeId":"n1"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	startProxy(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("startProxy failed: %d %s", w.Code, w.Body.String())
	}

	procMu.Lock()
	controller, secret := runtimeController, runtimeSecret
	procMu.Unlock()
	if controller == "" || secret == "" {
		t.Fatal("controller credentials were cleared during start sequence")
	}
	b, err := os.ReadFile(runtimeConfigPath())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), controller) || !strings.Contains(string(b), secret) {
		t.Fatalf("generated config does not reference runtime controller/secret\n%s", b)
	}
}

func TestBuildMihomoProxyNamed(t *testing.T) {
	n := Node{Protocol: "tuic", Address: "example.com", Port: 443, Proxy: map[string]any{"type": "tuic", "token": "abc"}}
	o := buildMihomoProxyNamed(n, "probe-7")
	if o["name"] != "probe-7" || o["token"] != "abc" {
		t.Fatalf("unexpected named proxy: %+v", o)
	}
}

func TestValidTestURL(t *testing.T) {
	good := []string{"", "http://www.gstatic.com/generate_204", "https://cp.cloudflare.com/generate_204"}
	bad := []string{"ftp://example.com", "genera", "://x"}
	for _, v := range good {
		if !validTestURL(v) {
			t.Fatalf("expected valid: %q", v)
		}
	}
	for _, v := range bad {
		if validTestURL(v) {
			t.Fatalf("expected invalid: %q", v)
		}
	}
}

func TestLoadConfigAppliesTestDefaults(t *testing.T) {
	oldCfg, oldDataDir := cfg, dataDir
	defer func() { cfg, dataDir = oldCfg, oldDataDir }()
	dataDir = t.TempDir()
	cfg = Config{}
	loadConfig()
	if cfg.TestURL != defaultTestURL {
		t.Fatalf("unexpected test url: %q", cfg.TestURL)
	}
	if cfg.TestTimeout != defaultTestTimeout {
		t.Fatalf("unexpected test timeout: %d", cfg.TestTimeout)
	}
}

func TestProxyProbeMeasuresDirectNode(t *testing.T) {
	skipWithoutBundledCore(t)
	oldCfg, oldDataDir := cfg, dataDir
	defer func() { cfg, dataDir = oldCfg, oldDataDir }()
	dataDir = t.TempDir()

	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// A non-zero delay is required: mihomo reports delay==0 as a failure.
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer target.Close()

	node := Node{ID: "direct-1", Protocol: "direct", Address: "127.0.0.1", Port: 1, Proxy: map[string]any{"type": "direct"}}
	probe, err := startProxyProbe([]Node{node})
	if err != nil {
		t.Fatalf("startProxyProbe: %v", err)
	}
	defer probe.close()

	delay, err := probe.delay("direct-1", target.URL, 5000)
	if err != nil {
		t.Fatalf("delay test failed: %v", err)
	}
	if delay <= 0 {
		t.Fatalf("unexpected delay: %d", delay)
	}
}

func TestProxyProbeGroupDelay(t *testing.T) {
	skipWithoutBundledCore(t)
	oldDataDir := dataDir
	defer func() { dataDir = oldDataDir }()
	dataDir = t.TempDir()

	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(30 * time.Millisecond)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer target.Close()

	nodes := []Node{
		{ID: "d1", Protocol: "direct", Address: "127.0.0.1", Port: 1, Proxy: map[string]any{"type": "direct"}},
		{ID: "d2", Protocol: "direct", Address: "127.0.0.1", Port: 1, Proxy: map[string]any{"type": "direct"}},
	}
	probe, err := startProxyProbe(nodes)
	if err != nil {
		t.Fatalf("startProxyProbe: %v", err)
	}
	defer probe.close()

	delays, err := probe.groupDelayAll(target.URL, 5000)
	if err != nil {
		t.Fatalf("groupDelay: %v", err)
	}
	if len(delays) != 2 {
		t.Fatalf("expected 2 delays, got %v", delays)
	}
	for _, name := range []string{"probe-0", "probe-1"} {
		if delays[name] <= 0 {
			t.Fatalf("missing delay for %s: %v", name, delays)
		}
	}
}

func TestProxyProbeSplitsIntoChunks(t *testing.T) {
	skipWithoutBundledCore(t)
	oldDataDir := dataDir
	defer func() { dataDir = oldDataDir }()
	dataDir = t.TempDir()

	nodes := make([]Node, probeChunkSize+3)
	for i := range nodes {
		nodes[i] = Node{ID: fmt.Sprintf("n%d", i), Protocol: "direct", Address: "127.0.0.1", Port: 1, Proxy: map[string]any{"type": "direct"}}
	}
	probe, err := startProxyProbe(nodes)
	if err != nil {
		t.Fatalf("startProxyProbe: %v", err)
	}
	defer probe.close()
	want := (len(nodes) + probeChunkSize - 1) / probeChunkSize
	if len(probe.groupNames) != want {
		t.Fatalf("expected %d chunks, got %d: %v", want, len(probe.groupNames), probe.groupNames)
	}
}

func skipWithoutBundledCore(t *testing.T) {
	t.Helper()
	bundledTestCore(t)
}

func bundledTestCore(t *testing.T) string {
	t.Helper()
	var p string
	if runtime.GOOS == "windows" {
		p = filepath.Join("mihomo-windows-64", "mihomo.exe")
	} else {
		p = filepath.Join("mihomo-linux-64", "mihomo")
	}
	if _, err := os.Stat(p); err != nil {
		t.Skipf("bundled mihomo core not found: %s", p)
		return ""
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		t.Fatal(err)
	}
	return abs
}
