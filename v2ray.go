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
	exe, _ := os.Executable()
	root := filepath.Dir(exe)
	if _, e := os.Stat(filepath.Join(root, "v2ray-windows-64")); e == nil {
		return root
	}
	if cwd, e := os.Getwd(); e == nil {
		return cwd
	}
	return root
}
func corePath() string {
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
func runtimeConfigPath() string { return filepath.Join(dataDir, "runtime", "config.json") }
func writeRuntime(n Node) error {
	c := map[string]any{"log": map[string]any{"loglevel": "warning", "access": filepath.Join(dataDir, "runtime", "access.log"), "error": filepath.Join(dataDir, "runtime", "error.log")}, "inbounds": []any{map[string]any{"listen": cfg.ListenAddress, "port": cfg.SocksPort, "protocol": "socks", "settings": map[string]any{"auth": "noauth", "udp": true}}, map[string]any{"listen": cfg.ListenAddress, "port": cfg.HTTPPort, "protocol": "http", "settings": map[string]any{}}}, "outbounds": []any{buildOutbound(n), map[string]any{"protocol": "freedom", "tag": "direct"}}}
	b, e := json.MarshalIndent(c, "", "  ")
	if e != nil {
		return e
	}
	return os.WriteFile(runtimeConfigPath(), b, 0644)
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
	cmd := exec.Command(corePath(), "test", "-config", runtimeConfigPath())
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
	cmd := exec.Command(corePath(), "run", "-config", runtimeConfigPath())
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
	state = State{Running: true, PID: cmd.Process.Pid, NodeID: n.ID, NodeName: n.Name, SocksPort: cfg.SocksPort, HTTPPort: cfg.HTTPPort, StartedAt: time.Now()}
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
