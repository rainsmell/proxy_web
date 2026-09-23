package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// proxyName is the single outbound name used in the generated Clash config.
const proxyName = "proxy"

// ---------- core discovery ----------

func appRoot() string {
	if dest := strings.TrimSpace(os.Getenv("TRIM_APPDEST")); dest != "" {
		for _, p := range []string{filepath.Join(dest, "app"), dest} {
			if hasCoreDir(p) {
				return p
			}
		}
	}
	exe, _ := os.Executable()
	root := filepath.Dir(exe)
	if hasCoreDir(root) {
		return root
	}
	if cwd, e := os.Getwd(); e == nil {
		if hasCoreDir(cwd) {
			return cwd
		}
		return cwd
	}
	return root
}

// hasCoreDir reports whether root looks like an unpacked release directory.
// Legacy v2ray directories are still recognized so dataDir resolution keeps
// working for users upgrading in place.
func hasCoreDir(root string) bool {
	for _, d := range []string{"mihomo-windows-64", "mihomo-linux-64", "v2ray-windows-64", "v2ray-linux-64"} {
		if _, e := os.Stat(filepath.Join(root, d)); e == nil {
			return true
		}
	}
	return false
}

func corePath() string {
	for _, k := range []string{"MIHOMO_CORE_PATH", "V2RAY_CORE_PATH"} {
		if p := strings.TrimSpace(os.Getenv(k)); p != "" {
			return p
		}
	}
	root := appRoot()
	var candidates []string
	if runtime.GOOS == "windows" {
		candidates = []string{
			filepath.Join(root, "mihomo-windows-64", "mihomo.exe"),
			filepath.Join(root, "mihomo-windows-64", "mihomo"),
			filepath.Join(root, "v2ray-windows-64", "v2ray.exe"),
		}
	} else {
		candidates = []string{
			filepath.Join(root, "mihomo-linux-64", "mihomo"),
			filepath.Join(root, "v2ray-linux-64", "v2ray"),
		}
	}
	for _, p := range candidates {
		if _, e := os.Stat(p); e == nil {
			return p
		}
	}
	if runtime.GOOS == "windows" {
		return "mihomo.exe"
	}
	return "mihomo"
}

// ---------- runtime config ----------

var runtimeConfigFile = "config.yaml"
var runtimeConfigFormat = "yaml"

// runtimeController and runtimeSecret describe the loopback REST API of the
// running mihomo child. Guarded by procMu.
var (
	runtimeController string
	runtimeSecret     string
)

func runtimeDirPath() string    { return filepath.Join(dataDir, "runtime") }
func runtimeConfigPath() string { return filepath.Join(runtimeDirPath(), runtimeConfigFile) }

func freeLocalPort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func randomToken(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 16)
	}
	return hex.EncodeToString(b)
}

func writeRuntime(n Node) error {
	if err := os.MkdirAll(runtimeDirPath(), 0755); err != nil {
		return err
	}
	port, err := freeLocalPort()
	if err != nil {
		return err
	}
	secret := randomToken(16)
	b, err := yaml.Marshal(buildMihomoConfig(n, port, secret))
	if err != nil {
		return err
	}
	// Remove configs written by the previous v2ray-based runtime.
	_ = os.Remove(filepath.Join(runtimeDirPath(), "config.json"))
	_ = os.Remove(filepath.Join(runtimeDirPath(), "config.v5.json"))
	if err := os.WriteFile(runtimeConfigPath(), b, 0644); err != nil {
		return err
	}
	procMu.Lock()
	runtimeController = "127.0.0.1:" + strconv.Itoa(port)
	runtimeSecret = secret
	procMu.Unlock()
	log.Printf("runtime config core=mihomo node=%q type=%s path=%s", n.Name, n.Protocol, runtimeConfigPath())
	return nil
}

func buildMihomoConfig(n Node, controllerPort int, secret string) map[string]any {
	return map[string]any{
		"socks-port":          cfg.SocksPort,
		"port":                cfg.HTTPPort,
		"mode":                "rule",
		"log-level":           mihomoLogLevel(cfg.LogLevel),
		"ipv6":                cfg.DomainStrategy != "UseIPv4",
		"allow-lan":           allowLAN(cfg.ListenAddress),
		"bind-address":        bindAddress(cfg.ListenAddress),
		"external-controller": "127.0.0.1:" + strconv.Itoa(controllerPort),
		"secret":              secret,
		"geodata-mode":        true,
		"geo-auto-update":     false,
		"find-process-mode":   "off",
		"profile": map[string]any{
			"store-selected": true,
			"store-fake-ip":  true,
		},
		"proxies": []any{buildMihomoProxy(n)},
		"proxy-groups": []any{map[string]any{
			"name":    "PROXY",
			"type":    "select",
			"proxies": []any{proxyName, "DIRECT"},
		}},
		"rules": []any{"MATCH,PROXY"},
	}
}

func allowLAN(addr string) bool {
	switch strings.TrimSpace(addr) {
	case "", "0.0.0.0", "*", "::", "[::]":
		return true
	default:
		return false
	}
}

func bindAddress(addr string) string {
	switch a := strings.TrimSpace(addr); a {
	case "", "0.0.0.0", "::", "[::]":
		return "*"
	default:
		return a
	}
}

func mihomoLogLevel(level string) string {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return "debug"
	case "info":
		return "info"
	case "warning", "warn":
		return "warning"
	case "error":
		return "error"
	case "none", "silent":
		return "silent"
	default:
		return "info"
	}
}

// buildMihomoProxy converts a Node into a mihomo (Clash.Meta) proxy entry.
// Nodes imported from a Clash YAML subscription keep their original mapping.
func buildMihomoProxy(n Node) map[string]any {
	return buildMihomoProxyNamed(n, proxyName)
}

// buildMihomoProxyNamed is buildMihomoProxy with an explicit outbound name,
// used by the latency probe to load many nodes into one config.
func buildMihomoProxyNamed(n Node, name string) map[string]any {
	if len(n.Proxy) > 0 {
		m := make(map[string]any, len(n.Proxy)+1)
		for k, v := range n.Proxy {
			m[k] = v
		}
		m["name"] = name
		return m
	}

	typ := mihomoProxyType(n.Protocol)
	m := map[string]any{
		"name":   name,
		"type":   typ,
		"server": n.Address,
		"port":   n.Port,
	}
	switch typ {
	case "vmess":
		m["uuid"] = n.UUID
		m["alterId"] = n.AlterID
		m["cipher"] = firstNonEmpty(n.Security, "auto")
		m["udp"] = true
	case "vless":
		m["uuid"] = n.UUID
		m["udp"] = true
		if n.Flow != "" {
			m["flow"] = n.Flow
		}
	case "trojan":
		m["password"] = n.Password
		m["udp"] = true
	case "ss":
		m["cipher"] = n.Method
		m["password"] = n.Password
		m["udp"] = true
	case "ssr":
		m["cipher"] = n.Method
		m["password"] = n.Password
		if n.SSRProtocol != "" {
			m["protocol"] = n.SSRProtocol
		}
		if n.SSRProtocolParam != "" {
			m["protocol-param"] = n.SSRProtocolParam
		}
		if n.SSRObfs != "" {
			m["obfs"] = n.SSRObfs
		}
		if n.SSRObfsParam != "" {
			m["obfs-param"] = n.SSRObfsParam
		}
		m["udp"] = true
	case "hysteria2":
		m["password"] = n.Password
		if n.Obfs != "" {
			m["obfs"] = n.Obfs
			if n.ObfsPassword != "" {
				m["obfs-password"] = n.ObfsPassword
			}
		}
		applyBrutal(m, n)
	case "hysteria":
		if n.Password != "" {
			m["auth-str"] = n.Password
		}
		if n.Obfs != "" {
			m["protocol"] = n.Obfs
		}
		applyBrutal(m, n)
	case "tuic":
		if n.Token != "" {
			m["token"] = n.Token
		} else {
			m["uuid"] = n.UUID
			m["password"] = n.Password
		}
		if n.CongestionController != "" {
			m["congestion-controller"] = n.CongestionController
		}
		if n.UDPRelayMode != "" {
			m["udp-relay-mode"] = n.UDPRelayMode
		}
	case "snell":
		m["psk"] = n.Password
		if n.SnellVersion > 0 {
			m["version"] = n.SnellVersion
		}
		m["udp"] = true
	}
	applyMihomoTLS(m, n, typ)
	applyMihomoTransport(m, n, typ)
	return m
}

func applyBrutal(m map[string]any, n Node) {
	if n.UpMbps > 0 {
		m["up"] = fmt.Sprintf("%d Mbps", n.UpMbps)
	}
	if n.DownMbps > 0 {
		m["down"] = fmt.Sprintf("%d Mbps", n.DownMbps)
	}
}

func applyMihomoTLS(m map[string]any, n Node, typ string) {
	tls := n.TLS
	switch typ {
	case "trojan", "hysteria", "hysteria2", "tuic":
		tls = true
	}
	reality := typ == "vless" && n.PublicKey != ""
	if reality {
		tls = true
	}
	if !tls {
		return
	}
	switch typ {
	case "hysteria", "hysteria2", "tuic":
		// QUIC based, tls is implicit.
	default:
		m["tls"] = true
	}
	if n.SNI != "" {
		if typ == "vmess" || typ == "vless" {
			m["servername"] = n.SNI
		} else {
			m["sni"] = n.SNI
		}
	}
	if reality {
		ro := map[string]any{"public-key": n.PublicKey}
		if n.ShortID != "" {
			ro["short-id"] = n.ShortID
		}
		m["reality-opts"] = ro
	}
	if n.SkipCertVerify {
		m["skip-cert-verify"] = true
	}
	if alpn := splitALPN(n.ALPN); len(alpn) > 0 {
		m["alpn"] = alpn
	}
	switch typ {
	case "vmess", "vless", "trojan":
		fp := normalizeFingerprint(n.Fingerprint)
		if fp == "" && reality {
			fp = "chrome"
		}
		if fp != "" {
			m["client-fingerprint"] = fp
		}
	}
}

func applyMihomoTransport(m map[string]any, n Node, typ string) {
	switch typ {
	case "ss", "ssr", "hysteria", "hysteria2", "tuic", "snell":
		return
	}
	switch strings.ToLower(strings.TrimSpace(n.Network)) {
	case "ws", "websocket":
		m["network"] = "ws"
		ws := map[string]any{}
		if n.Path != "" {
			ws["path"] = n.Path
		}
		if n.Host != "" {
			ws["headers"] = map[string]any{"Host": n.Host}
		}
		if len(ws) > 0 {
			m["ws-opts"] = ws
		}
	case "grpc", "gun":
		m["network"] = "grpc"
		if n.Path != "" {
			m["grpc-opts"] = map[string]any{"grpc-service-name": n.Path}
		}
	case "h2", "http":
		m["network"] = "h2"
		h2 := map[string]any{}
		if n.Host != "" {
			h2["host"] = []any{n.Host}
		}
		if n.Path != "" {
			h2["path"] = n.Path
		}
		if len(h2) > 0 {
			m["h2-opts"] = h2
		}
	}
}

// mihomoProxyType maps the internal protocol name to the mihomo `type` value.
func mihomoProxyType(protocol string) string {
	switch normalizeProtocol(protocol) {
	case "shadowsocks":
		return "ss"
	case "shadowsocksr":
		return "ssr"
	default:
		return normalizeProtocol(protocol)
	}
}

func splitALPN(raw string) []string {
	var out []string
	for _, part := range strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == ';' || r == ' ' }) {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// normalizeFingerprint maps v2ray/xray fingerprint names to mihomo uTLS names.
func normalizeFingerprint(fp string) string {
	switch strings.ToLower(strings.TrimSpace(fp)) {
	case "":
		return ""
	case "chrome", "chrome_auto":
		return "chrome"
	case "firefox", "firefox_auto":
		return "firefox"
	case "safari", "safari_auto":
		return "safari"
	case "ios", "ios_14", "ios_15", "ios_auto":
		return "ios"
	case "edge", "edge_auto":
		return "edge"
	case "360", "360_auto":
		return "360"
	case "qq", "qq_auto":
		return "qq"
	case "android", "android_11_okhttp":
		return "android"
	case "random", "randomized", "randomizedalpn", "randomizednoalpn":
		return "random"
	default:
		return strings.ToLower(strings.TrimSpace(fp))
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// ---------- run / stop ----------

func validateRuntime() error {
	cmd := exec.Command(corePath(), "-t", "-d", runtimeDirPath(), "-f", runtimeConfigPath())
	cmd.Dir = filepath.Dir(corePath())
	out, e := cmd.CombinedOutput()
	if e != nil {
		return fmt.Errorf("mihomo config invalid: %s%s", trimOutput(string(out)), coreArchHint(string(out)))
	}
	return nil
}

// coreArchHint explains the x86-64-v3 failure emitted by the upstream mihomo
// build when the host CPU lacks AVX2 (common on low-power NAS CPUs).
func coreArchHint(out string) string {
	if strings.Contains(out, "v3 microarchitecture") || strings.Contains(out, "only be run on AMD64") {
		return "\n提示: 当前 CPU 不支持 x86-64-v3/AVX2，请改用 mihomo 的 -compatible 内核（scripts/fetch-core.sh 默认就是 compatible 版本）"
	}
	return ""
}

func trimOutput(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "(no output)"
	}
	lines := strings.Split(s, "\n")
	if len(lines) > 20 {
		lines = lines[len(lines)-20:]
	}
	out := strings.Join(lines, "\n")
	if len(out) > 2000 {
		out = out[len(out)-2000:]
	}
	return out
}

func startProxy(w http.ResponseWriter, r *http.Request) {
	var x struct {
		NodeID string `json:"nodeId"`
	}
	if bodyJSON(r, &x) != nil {
		jsonOut(w, 400, map[string]string{"error": "invalid body"})
		return
	}
	n, ok := findNode(x.NodeID)
	if !ok {
		jsonOut(w, 404, map[string]string{"error": "node not found"})
		return
	}
	if cfg.SocksPort == cfg.HTTPPort {
		jsonOut(w, 400, map[string]string{"error": "SOCKS5 and HTTP ports must differ"})
		return
	}
	// Stop any previous child (and clear its controller credentials) before
	// writing the new runtime config, otherwise the freshly generated
	// controller address/secret would be wiped.
	stopProxyInternal()
	if err := writeRuntime(n); err != nil {
		jsonOut(w, 500, map[string]string{"error": err.Error()})
		return
	}
	if err := validateRuntime(); err != nil {
		log.Printf("start rejected for node %s: %v", n.ID, err)
		stopProxyInternal()
		jsonOut(w, 500, map[string]string{"error": err.Error()})
		return
	}
	cmd := exec.Command(corePath(), "-d", runtimeDirPath(), "-f", runtimeConfigPath())
	cmd.Dir = filepath.Dir(corePath())
	logFile, _ := os.OpenFile(filepath.Join(runtimeDirPath(), "process.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if logFile != nil {
		cmd.Stdout = logFile
		cmd.Stderr = logFile
	}
	if err := cmd.Start(); err != nil {
		if logFile != nil {
			logFile.Close()
		}
		jsonOut(w, 500, map[string]string{"error": err.Error()})
		return
	}
	procMu.Lock()
	running = cmd
	state = State{Running: true, PID: cmd.Process.Pid, NodeID: n.ID, NodeName: n.Name, SocksPort: cfg.SocksPort, HTTPPort: cfg.HTTPPort, ConfigPath: runtimeConfigPath(), ConfigFormat: runtimeConfigFormat, Version: appVersion, StartedAt: time.Now()}
	procMu.Unlock()
	go func() {
		err := cmd.Wait()
		if logFile != nil {
			_ = logFile.Close()
		}
		procMu.Lock()
		defer procMu.Unlock()
		if running == cmd {
			running = nil
			state.Running = false
			state.PID = 0
			if err != nil {
				state.Error = err.Error()
			}
		}
	}()
	if err := waitPort(cfg.ListenAddress, cfg.SocksPort, 3*time.Second); err != nil {
		stopProxyInternal()
		jsonOut(w, 502, map[string]any{"error": "mihomo started but SOCKS5 port was not opened: " + err.Error(), "logs": tailLog("process.log", 30)})
		return
	}
	jsonOut(w, 200, state)
}

func waitPort(addr string, port int, d time.Duration) error {
	ctx, c := context.WithTimeout(context.Background(), d)
	defer c()
	for {
		var dialer net.Dialer
		conn, e := dialer.DialContext(ctx, "tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if e == nil {
			conn.Close()
			return nil
		}
		select {
		case <-ctx.Done():
			return e
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func stopProxy(w http.ResponseWriter) { stopProxyInternal(); jsonOut(w, 200, state) }

func stopProxyInternal() {
	procMu.Lock()
	defer procMu.Unlock()
	if running != nil && running.Process != nil {
		_ = running.Process.Kill()
		running = nil
	}
	runtimeController = ""
	runtimeSecret = ""
	state.Running = false
	state.PID = 0
}

// ---------- live connections via mihomo REST API ----------

type mihomoConnectionMeta struct {
	Network         string `json:"network"`
	Type            string `json:"type"`
	SourceIP        string `json:"sourceIP"`
	SourcePort      string `json:"sourcePort"`
	DestinationIP   string `json:"destinationIP"`
	DestinationPort string `json:"destinationPort"`
	Host            string `json:"host"`
	SniffHost       string `json:"sniffHost"`
}

type mihomoConnection struct {
	ID       string               `json:"id"`
	Metadata mihomoConnectionMeta `json:"metadata"`
	Upload   int64                `json:"upload"`
	Download int64                `json:"download"`
	Start    time.Time            `json:"start"`
	Chains   []string             `json:"chains"`
	Rule     string               `json:"rule"`
}

type mihomoConnectionSnapshot struct {
	Connections []mihomoConnection `json:"connections"`
}

func readConnections() []Connection {
	procMu.Lock()
	addr, secret := runtimeController, runtimeSecret
	procMu.Unlock()
	if addr == "" {
		return []Connection{}
	}
	req, err := http.NewRequest(http.MethodGet, "http://"+addr+"/connections", nil)
	if err != nil {
		return []Connection{}
	}
	if secret != "" {
		req.Header.Set("Authorization", "Bearer "+secret)
	}
	client := &http.Client{Timeout: 1500 * time.Millisecond}
	resp, err := client.Do(req)
	if err != nil {
		return []Connection{}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return []Connection{}
	}
	var snap mihomoConnectionSnapshot
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&snap); err != nil {
		return []Connection{}
	}
	out := make([]Connection, 0, len(snap.Connections))
	for _, c := range snap.Connections {
		m := c.Metadata
		host := firstNonEmpty(m.SniffHost, m.Host, m.DestinationIP)
		status := strings.ToUpper(strings.TrimSpace(m.Network))
		if status == "" {
			status = "active"
		}
		out = append(out, Connection{
			ID:     c.ID,
			Source: net.JoinHostPort(m.SourceIP, m.SourcePort),
			Target: net.JoinHostPort(host, m.DestinationPort),
			Status: status,
			Detour: strings.Join(c.Chains, " <- "),
			Time:   c.Start.Format("15:04:05"),
		})
	}
	return out
}
