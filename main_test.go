package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestParseVLESS(t *testing.T) {
	n, err := parseVLESS("vless://11111111-1111-1111-1111-111111111111@example.com:443?type=ws&security=tls&sni=cdn.example.com&host=cdn.example.com&path=%2Fws#demo")
	if err != nil {
		t.Fatal(err)
	}
	if n.Protocol != "vless" || n.Port != 443 || n.Network != "ws" || !n.TLS || n.Path != "/ws" {
		t.Fatalf("unexpected node: %+v", n)
	}
}

func TestVLESSOutboundEncryption(t *testing.T) {
	o := buildOutbound(Node{Protocol: "vless", Address: "example.com", Port: 443, UUID: "11111111-1111-1111-1111-111111111111"})
	settings := o["settings"].(map[string]any)
	vnext := settings["vnext"].([]any)
	users := vnext[0].(map[string]any)["users"].([]any)
	if users[0].(map[string]any)["encryption"] != "none" {
		t.Fatal("VLESS encryption must be none")
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
	t.Setenv("V2RAY_CORE_PATH", "/opt/v2ray/v2ray")
	if got := corePath(); got != "/opt/v2ray/v2ray" {
		t.Fatalf("explicit core path not respected: %q", got)
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

func TestBuildOutboundOmitsDomainStrategyByDefault(t *testing.T) {
	old := cfg
	defer func() { cfg = old }()
	cfg.DomainStrategy = "AsIs"
	o := buildOutbound(Node{Protocol: "vless", Address: "example.com", Port: 443, UUID: "11111111-1111-1111-1111-111111111111"})
	if _, ok := o["domainStrategy"]; ok {
		t.Fatalf("domainStrategy should be omitted by default: %+v", o)
	}
}

func TestBuildOutboundV5UsesUTLSForFingerprint(t *testing.T) {
	old := cfg
	defer func() { cfg = old }()
	cfg.DomainStrategy = "AsIs"
	n := Node{Protocol: "vless", Address: "example.com", Port: 443, UUID: "11111111-1111-1111-1111-111111111111", Network: "ws", TLS: true, SNI: "sni.example.com", Host: "host.example.com", Path: "/ws", Fingerprint: "ios"}
	o := buildOutboundV5(n)
	ss := o["streamSettings"].(map[string]any)
	if ss["security"] != "utls" {
		t.Fatalf("expected utls security: %+v", ss)
	}
	sec := ss["securitySettings"].(map[string]any)
	if sec["imitate"] != "ios_14" {
		t.Fatalf("expected ios_14 imitate: %+v", sec)
	}
	if ss["transport"] != "ws" {
		t.Fatalf("expected ws transport: %+v", ss)
	}
}

func TestWriteRuntimeV5ConfigAcceptedByBundledCore(t *testing.T) {
	core := bundledTestCore(t)
	if core == "" {
		return
	}
	oldCfg, oldDataDir, oldFile, oldFormat := cfg, dataDir, runtimeConfigFile, runtimeConfigFormat
	defer func() {
		cfg, dataDir, runtimeConfigFile, runtimeConfigFormat = oldCfg, oldDataDir, oldFile, oldFormat
	}()
	dataDir = t.TempDir()
	if err := os.MkdirAll(filepath.Join(dataDir, "runtime"), 0755); err != nil {
		t.Fatal(err)
	}
	cfg.ListenAddress = "127.0.0.1"
	cfg.SocksPort = 11808
	cfg.HTTPPort = 11809
	cfg.LogLevel = "info"

	n := Node{Protocol: "vless", Address: "unamecf2.xn--ghqu5fm27b67w.com", Port: 443, UUID: "b22681d8-daad-4eb8-95bc-1078148358f9", Network: "ws", TLS: true, SNI: "ujp5.xn--ghqu5fm27b67w.com", Host: "ujp5.xn--ghqu5fm27b67w.com", Path: "/pq/jp5", Fingerprint: "ios"}
	if err := writeRuntime(n); err != nil {
		t.Fatal(err)
	}
	if runtimeConfigFormat != "jsonv5" {
		t.Fatalf("expected jsonv5 format, got %q", runtimeConfigFormat)
	}
	cmd := exec.Command(core, "test", "-format", runtimeConfigFormat, "-config", runtimeConfigPath())
	cmd.Dir = filepath.Dir(core)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("v2ray rejected generated config: %v\n%s", err, out)
	}
}

func bundledTestCore(t *testing.T) string {
	t.Helper()
	var p string
	if runtime.GOOS == "windows" {
		p = filepath.Join("v2ray-windows-64", "v2ray.exe")
	} else {
		p = filepath.Join("v2ray-linux-64", "v2ray")
	}
	if _, err := os.Stat(p); err != nil {
		t.Skipf("bundled v2ray core not found: %s", p)
		return ""
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		t.Fatal(err)
	}
	return abs
}
