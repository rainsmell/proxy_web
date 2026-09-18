package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

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

func hasCoreDir(root string) bool {
	if _, e := os.Stat(filepath.Join(root, "v2ray-windows-64")); e == nil {
		return true
	}
	if _, e := os.Stat(filepath.Join(root, "v2ray-linux-64")); e == nil {
		return true
	}
	return false
}
func corePath() string {
	if p := strings.TrimSpace(os.Getenv("V2RAY_CORE_PATH")); p != "" {
		return p
	}
	root := appRoot()
	if runtime.GOOS == "windows" {
		p := filepath.Join(root, "v2ray-windows-64", "v2ray.exe")
		if _, e := os.Stat(p); e == nil {
			return p
		}
	} else {
		p := filepath.Join(root, "v2ray-linux-64", "v2ray")
		if _, e := os.Stat(p); e == nil {
			return p
		}
	}
	return "v2ray"
}

var runtimeConfigFile = "config.json"
var runtimeConfigFormat = "json"

func runtimeConfigPath() string { return filepath.Join(dataDir, "runtime", runtimeConfigFile) }
func writeRuntime(n Node) error {
	if shouldUseV5UTLS(n) {
		runtimeConfigFile = "config.v5.json"
		runtimeConfigFormat = "jsonv5"
		_ = os.Remove(filepath.Join(dataDir, "runtime", "config.json"))
		log.Printf("runtime config mode=v5-utls format=%s node=%q fingerprint=%q path=%s", runtimeConfigFormat, n.Name, n.Fingerprint, runtimeConfigPath())
		return writeRuntimeV5(n)
	}
	runtimeConfigFile = "config.json"
	runtimeConfigFormat = "json"
	_ = os.Remove(filepath.Join(dataDir, "runtime", "config.v5.json"))
	log.Printf("runtime config mode=v4 format=%s node=%q fingerprint=%q path=%s", runtimeConfigFormat, n.Name, n.Fingerprint, runtimeConfigPath())
	c := map[string]any{"log": map[string]any{"loglevel": cfg.LogLevel, "access": filepath.Join(dataDir, "runtime", "access.log"), "error": filepath.Join(dataDir, "runtime", "error.log")}, "inbounds": []any{map[string]any{"listen": cfg.ListenAddress, "port": cfg.SocksPort, "protocol": "socks", "settings": map[string]any{"auth": "noauth", "udp": true}}, map[string]any{"listen": cfg.ListenAddress, "port": cfg.HTTPPort, "protocol": "http", "settings": map[string]any{}}}, "outbounds": []any{buildOutbound(n), map[string]any{"protocol": "freedom", "tag": "direct"}}}
	b, e := json.MarshalIndent(c, "", "  ")
	if e != nil {
		return e
	}
	return os.WriteFile(runtimeConfigPath(), b, 0644)
}

func shouldUseV5UTLS(n Node) bool {
	if !n.TLS || strings.TrimSpace(n.Fingerprint) == "" {
		return false
	}
	switch n.Protocol {
	case "vless":
		return strings.TrimSpace(n.Flow) == ""
	case "trojan", "shadowsocks":
		return true
	default:
		return false
	}
}

func writeRuntimeV5(n Node) error {
	c := map[string]any{
		"log": logConfigV5(),
		"inbounds": []any{
			map[string]any{"listen": cfg.ListenAddress, "port": fmt.Sprint(cfg.SocksPort), "protocol": "socks", "settings": map[string]any{"udpEnabled": true}},
			map[string]any{"listen": cfg.ListenAddress, "port": fmt.Sprint(cfg.HTTPPort), "protocol": "http", "settings": map[string]any{}},
		},
		"outbounds": []any{buildOutboundV5(n), map[string]any{"protocol": "freedom", "settings": map[string]any{}, "tag": "direct"}},
	}
	b, e := json.MarshalIndent(c, "", "  ")
	if e != nil {
		return e
	}
	return os.WriteFile(runtimeConfigPath(), b, 0644)
}

func logConfigV5() map[string]any {
	return map[string]any{
		"access": map[string]any{
			"type": "File",
			"path": filepath.Join(dataDir, "runtime", "access.log"),
		},
		"error": map[string]any{
			"type":  "File",
			"level": logLevelV5(cfg.LogLevel),
			"path":  filepath.Join(dataDir, "runtime", "error.log"),
		},
	}
}

func logLevelV5(level string) string {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return "Debug"
	case "info":
		return "Info"
	case "error":
		return "Error"
	case "none":
		return "Error"
	default:
		return "Warning"
	}
}

func buildOutboundV5(n Node) map[string]any {
	o := map[string]any{"protocol": n.Protocol, "settings": buildOutboundSettingsV5(n)}
	ss := map[string]any{"transport": v5TransportName(n.Network)}
	if n.Network == "ws" {
		w := map[string]any{"path": n.Path}
		if n.Host != "" {
			w["header"] = []any{map[string]any{"key": "Host", "value": n.Host}}
		}
		ss["transportSettings"] = w
	}
	if n.Network == "grpc" {
		ss["transportSettings"] = map[string]any{"serviceName": n.Path}
	}
	if n.TLS {
		security := "tls"
		securitySettings := map[string]any{"serverName": n.SNI}
		if n.Fingerprint != "" {
			security = "utls"
			securitySettings = map[string]any{"tlsConfig": map[string]any{"serverName": n.SNI}, "imitate": normalizeFingerprint(n.Fingerprint)}
		}
		ss["security"] = security
		ss["securitySettings"] = securitySettings
	} else {
		ss["security"] = "none"
	}
	o["streamSettings"] = ss
	return o
}

func buildOutboundSettingsV5(n Node) map[string]any {
	switch n.Protocol {
	case "vless":
		return map[string]any{"address": addressV5(n.Address), "port": n.Port, "uuid": n.UUID}
	case "trojan":
		return map[string]any{"address": addressV5(n.Address), "port": n.Port, "password": n.Password}
	case "shadowsocks":
		return map[string]any{"address": addressV5(n.Address), "port": n.Port, "method": n.Method, "password": n.Password}
	default:
		return map[string]any{}
	}
}

func addressV5(address string) string {
	return address
}

func v5TransportName(network string) string {
	switch network {
	case "ws", "websocket":
		return "ws"
	case "grpc", "gun":
		return "grpc"
	case "":
		return "tcp"
	default:
		return network
	}
}

func normalizeFingerprint(fp string) string {
	switch strings.ToLower(strings.TrimSpace(fp)) {
	case "chrome":
		return "chrome_auto"
	case "firefox":
		return "firefox_auto"
	case "safari":
		return "safari_auto"
	case "ios":
		return "ios_14"
	case "edge":
		return "edge_auto"
	case "360":
		return "360_auto"
	case "qq":
		return "qq_auto"
	case "android":
		return "android_11_okhttp"
	case "random", "randomized":
		return "randomized"
	case "randomizedalpn":
		return "randomizedalpn"
	case "randomizednoalpn":
		return "randomizednoalpn"
	default:
		return strings.ToLower(strings.TrimSpace(fp))
	}
}

func buildOutbound(n Node) map[string]any {
	var settings map[string]any
	switch n.Protocol {
	case "vmess":
		settings = map[string]any{"vnext": []any{map[string]any{"address": n.Address, "port": n.Port, "users": []any{map[string]any{"id": n.UUID, "alterId": n.AlterID, "security": "auto"}}}}}
	case "vless":
		u := map[string]any{"id": n.UUID, "encryption": "none"}
		if n.Flow != "" {
			u["flow"] = n.Flow
		}
		settings = map[string]any{"vnext": []any{map[string]any{"address": n.Address, "port": n.Port, "users": []any{u}}}}
	case "trojan":
		settings = map[string]any{"servers": []any{map[string]any{"address": n.Address, "port": n.Port, "password": n.Password}}}
	case "shadowsocks":
		settings = map[string]any{"servers": []any{map[string]any{"address": n.Address, "port": n.Port, "method": n.Method, "password": n.Password}}}
	default:
		settings = map[string]any{}
	}
	o := map[string]any{"protocol": n.Protocol, "settings": settings}
	if cfg.DomainStrategy != "" && cfg.DomainStrategy != "AsIs" {
		o["domainStrategy"] = cfg.DomainStrategy
	}
	ss := map[string]any{}
	network := n.Network
	if network == "" {
		network = "tcp"
	}
	ss["network"] = network
	if network == "ws" {
		w := map[string]any{"path": n.Path}
		if n.Host != "" {
			w["headers"] = map[string]any{"Host": n.Host}
		}
		ss["wsSettings"] = w
	}
	if network == "grpc" {
		ss["grpcSettings"] = map[string]any{"serviceName": n.Path}
	}
	if n.TLS {
		tls := map[string]any{"serverName": n.SNI}
		if n.Fingerprint != "" {
			tls["fingerprint"] = n.Fingerprint
		}
		ss["security"] = "tls"
		ss["tlsSettings"] = tls
	}
	if len(ss) > 1 || network != "tcp" {
		o["streamSettings"] = ss
	}
	return o
}
func validateRuntime() error {
	cmd := exec.Command(corePath(), "test", "-format", runtimeConfigFormat, "-config", runtimeConfigPath())
	cmd.Dir = filepath.Dir(corePath())
	out, e := cmd.CombinedOutput()
	if e != nil {
		return fmt.Errorf("v2ray config invalid: %s", strings.TrimSpace(string(out)))
	}
	return nil
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
	if err := writeRuntime(n); err != nil {
		jsonOut(w, 500, map[string]string{"error": err.Error()})
		return
	}
	if err := validateRuntime(); err != nil {
		log.Printf("start rejected for node %s: %v", n.ID, err)
		jsonOut(w, 500, map[string]string{"error": err.Error()})
		return
	}
	stopProxyInternal()
	cmd := exec.Command(corePath(), "run", "-format", runtimeConfigFormat, "-config", runtimeConfigPath())
	cmd.Dir = filepath.Dir(corePath())
	logFile, _ := os.OpenFile(filepath.Join(dataDir, "runtime", "process.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err := cmd.Start(); err != nil {
		logFile.Close()
		jsonOut(w, 500, map[string]string{"error": err.Error()})
		return
	}
	procMu.Lock()
	running = cmd
	state = State{Running: true, PID: cmd.Process.Pid, NodeID: n.ID, NodeName: n.Name, SocksPort: cfg.SocksPort, HTTPPort: cfg.HTTPPort, ConfigPath: runtimeConfigPath(), ConfigFormat: runtimeConfigFormat, Version: appVersion, StartedAt: time.Now()}
	procMu.Unlock()
	go func() {
		err := cmd.Wait()
		_ = logFile.Close()
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
		jsonOut(w, 502, map[string]any{"error": "V2Ray started but SOCKS5 port was not opened: " + err.Error(), "logs": tailLog("error.log", 30)})
		return
	}
	jsonOut(w, 200, state)
}
func waitPort(addr string, port int, d time.Duration) error {
	ctx, c := context.WithTimeout(context.Background(), d)
	defer c()
	for {
		var d net.Dialer
		conn, e := d.DialContext(ctx, "tcp", fmt.Sprintf("127.0.0.1:%d", port))
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
	state.Running = false
	state.PID = 0
}
