package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	defaultTestURL        = "http://www.gstatic.com/generate_204"
	defaultTestTimeout    = 8000
	probeGroupName        = "PROBE"
	probeChunkSize        = 12
	probeChunkConcurrency = 2
)

// proxyProbe runs a short-lived mihomo instance that loads every node as a
// separate outbound proxy. It is used to measure real proxy latency through
// mihomo's native URLTest REST API instead of a raw TCP dial.
type proxyProbe struct {
	cmd        *exec.Cmd
	dir        string
	addr       string
	secret     string
	names      map[string]string // node ID -> unique probe proxy name
	groupNames []string          // chunked proxy group names
	client     *http.Client
	logPath    string
}

// startProxyProbe writes a probe config containing all nodes, validates it and
// starts a mihomo instance exposing the REST API on loopback only.
func startProxyProbe(nodes []Node) (*proxyProbe, error) {
	if len(nodes) == 0 {
		return nil, fmt.Errorf("no nodes to probe")
	}
	core := corePath()
	dir := filepath.Join(runtimeDirPath(), "probe", randomToken(6))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	port, err := freeLocalPort()
	if err != nil {
		return nil, err
	}
	secret := randomToken(16)

	names := make(map[string]string, len(nodes))
	proxies := make([]any, 0, len(nodes))
	allNames := make([]any, 0, len(nodes))
	for i, n := range nodes {
		name := fmt.Sprintf("probe-%d", i)
		if n.ID != "" {
			names[n.ID] = name
		}
		proxies = append(proxies, buildMihomoProxyNamed(n, name))
		allNames = append(allNames, name)
	}

	// Split the nodes into chunks. Each chunk is tested by one group delay
	// call with its own timeout, which avoids one slow node consuming the
	// whole shared deadline and keeps concurrent sockets bounded.
	var groups []any
	var groupNames []string
	for i := 0; i < len(allNames); i += probeChunkSize {
		end := i + probeChunkSize
		if end > len(allNames) {
			end = len(allNames)
		}
		gname := fmt.Sprintf("%s-%d", probeGroupName, len(groupNames))
		groupNames = append(groupNames, gname)
		groups = append(groups, map[string]any{
			"name":    gname,
			"type":    "select",
			"proxies": allNames[i:end],
		})
	}

	config := map[string]any{
		"mode":                "rule",
		"log-level":           "warning",
		"ipv6":                true,
		"external-controller": "127.0.0.1:" + strconv.Itoa(port),
		"secret":              secret,
		"geodata-mode":        true,
		"geo-auto-update":     false,
		"find-process-mode":   "off",
		"proxies":             proxies,
		"proxy-groups":        groups,
		"rules":               []any{"MATCH,DIRECT"},
	}
	b, err := yaml.Marshal(config)
	if err != nil {
		return nil, err
	}
	cfgPath := filepath.Join(dir, "probe.yaml")
	if err := os.WriteFile(cfgPath, b, 0644); err != nil {
		return nil, err
	}

	validate := exec.Command(core, "-t", "-d", dir, "-f", cfgPath)
	validate.Dir = filepath.Dir(core)
	if out, err := validate.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("probe config invalid: %s", trimOutput(string(out)))
	}

	cmd := exec.Command(core, "-d", dir, "-f", cfgPath)
	cmd.Dir = filepath.Dir(core)
	logPath := filepath.Join(dir, "probe.log")
	logFile, _ := os.OpenFile(logPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if logFile != nil {
		cmd.Stdout = logFile
		cmd.Stderr = logFile
	}
	if err := cmd.Start(); err != nil {
		if logFile != nil {
			logFile.Close()
		}
		return nil, err
	}
	go func() {
		_ = cmd.Wait()
		if logFile != nil {
			_ = logFile.Close()
		}
	}()

	p := &proxyProbe{
		cmd:        cmd,
		dir:        dir,
		addr:       "127.0.0.1:" + strconv.Itoa(port),
		secret:     secret,
		names:      names,
		groupNames: groupNames,
		client:     &http.Client{Timeout: 3 * time.Second},
		logPath:    logPath,
	}

	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		if err := p.ping(); err == nil {
			return p, nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	p.close()
	return nil, fmt.Errorf("probe core did not become ready: %s", tailFile(logPath, 5))
}

func (p *proxyProbe) ping() error {
	req, err := http.NewRequest(http.MethodGet, "http://"+p.addr+"/version", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+p.secret)
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	return nil
}

// delay asks mihomo to run a real request through the given node and returns
// the measured round-trip in milliseconds.
func (p *proxyProbe) delay(nodeID, testURL string, timeoutMs int) (int64, error) {
	name, ok := p.names[nodeID]
	if !ok {
		return 0, fmt.Errorf("node not loaded into probe")
	}
	if timeoutMs < 1000 {
		timeoutMs = 1000
	}
	if timeoutMs > 32767 {
		timeoutMs = 32767
	}
	q := url.Values{}
	q.Set("url", testURL)
	q.Set("timeout", strconv.Itoa(timeoutMs))
	endpoint := "http://" + p.addr + "/proxies/" + url.PathEscape(name) + "/delay?" + q.Encode()
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer "+p.secret)
	client := &http.Client{Timeout: time.Duration(timeoutMs)*time.Millisecond + 3*time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	if resp.StatusCode != http.StatusOK {
		var e struct {
			Message string `json:"message"`
		}
		if json.Unmarshal(body, &e) == nil && e.Message != "" {
			return 0, fmt.Errorf("%s", e.Message)
		}
		return 0, fmt.Errorf("delay test failed: %s", resp.Status)
	}
	var out struct {
		Delay int64 `json:"delay"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return 0, err
	}
	if out.Delay <= 0 {
		return 0, fmt.Errorf("delay test returned no result")
	}
	return out.Delay, nil
}

// groupDelay asks mihomo to test every probe proxy against the URL. mihomo
// runs the node tests concurrently server-side, so this is one HTTP call for
// all nodes. It returns proxy name -> delay in milliseconds; nodes that failed
// are simply absent from the map.
func (p *proxyProbe) groupDelay(group, testURL string, timeoutMs int) (map[string]int64, error) {
	if timeoutMs < 1000 {
		timeoutMs = 1000
	}
	if timeoutMs > 32767 {
		timeoutMs = 32767
	}
	q := url.Values{}
	q.Set("url", testURL)
	q.Set("timeout", strconv.Itoa(timeoutMs))
	endpoint := "http://" + p.addr + "/group/" + url.PathEscape(group) + "/delay?" + q.Encode()
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+p.secret)
	client := &http.Client{Timeout: time.Duration(timeoutMs)*time.Millisecond + 5*time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	// 504 means every node failed; that is a normal result, not an error.
	if resp.StatusCode == http.StatusGatewayTimeout {
		return map[string]int64{}, nil
	}
	if resp.StatusCode != http.StatusOK {
		var e struct {
			Message string `json:"message"`
		}
		if json.Unmarshal(body, &e) == nil && e.Message != "" {
			return nil, fmt.Errorf("%s", e.Message)
		}
		return nil, fmt.Errorf("group delay test failed: %s", resp.Status)
	}
	raw := map[string]int64{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	out := make(map[string]int64, len(raw))
	for name, d := range raw {
		if d > 0 {
			out[name] = d
		}
	}
	return out, nil
}

// groupDelayAll tests every chunk and merges the results. A few chunks run at a
// time so a small NAS is not asked to open hundreds of sockets at once.
func (p *proxyProbe) groupDelayAll(testURL string, timeoutMs int) (map[string]int64, error) {
	out := map[string]int64{}
	if len(p.groupNames) == 0 {
		return out, nil
	}
	sem := make(chan struct{}, probeChunkConcurrency)
	var mu sync.Mutex
	var wg sync.WaitGroup
	var firstErr error
	for _, group := range p.groupNames {
		wg.Add(1)
		go func(group string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			delays, err := p.groupDelay(group, testURL, timeoutMs)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				if firstErr == nil {
					firstErr = err
				}
				return
			}
			for name, d := range delays {
				out[name] = d
			}
		}(group)
	}
	wg.Wait()
	return out, firstErr
}

func (p *proxyProbe) close() {
	if p == nil {
		return
	}
	if p.cmd != nil && p.cmd.Process != nil {
		_ = p.cmd.Process.Kill()
	}
	// Give the child a moment to exit before removing its data dir (best
	// effort: on Windows a still-open file may keep the dir around).
	time.Sleep(50 * time.Millisecond)
	if p.dir != "" {
		_ = os.RemoveAll(p.dir)
	}
}

func tailFile(path string, n int) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	lines := strings.Split(strings.TrimRight(string(b), "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}

func probeTestURL() string {
	if cfg.TestURL != "" {
		return cfg.TestURL
	}
	return defaultTestURL
}

func probeTestTimeoutMs() int {
	if cfg.TestTimeout >= 1000 && cfg.TestTimeout <= 32767 {
		return cfg.TestTimeout
	}
	return defaultTestTimeout
}

// testNodeTCP measures raw TCP reachability to the node server.
func testNodeTCP(n *Node) {
	n.TestError = ""
	n.TCPDelay = 0
	start := time.Now()
	dialer := net.Dialer{Timeout: 5 * time.Second}
	conn, err := dialer.Dial("tcp", fmt.Sprintf("%s:%d", n.Address, n.Port))
	if err == nil {
		_ = conn.Close()
		n.TCPDelay = time.Since(start).Milliseconds()
	} else {
		n.TestError = "tcp: " + err.Error()
	}
	n.TestedAt = time.Now()
}

// applyProxyDelay records a real proxy latency result. A successful result
// clears any TCP error (UDP-only protocols fail the raw TCP probe but work
// fine through mihomo); a failed result only fills TestError if the TCP probe
// did not already report something.
func applyProxyDelay(n *Node, delay int64, err error) {
	n.ProxyDelay = 0
	if err == nil {
		n.ProxyDelay = delay
		n.TestError = ""
		return
	}
	if n.TestError == "" {
		n.TestError = err.Error()
	}
}

// testNode measures TCP reachability and, when a probe is available, the real
// proxy delay through mihomo (used for single-node tests).
func testNode(n *Node, probe *proxyProbe) {
	testNodeTCP(n)
	if probe != nil {
		delay, err := probe.delay(n.ID, probeTestURL(), probeTestTimeoutMs())
		applyProxyDelay(n, delay, err)
	}
	n.TestedAt = time.Now()
}

// startProbeForNodes is a small helper that logs probe startup failures and
// returns nil so callers can fall back to TCP-only testing.
func startProbeForNodes(nodes []Node) (*proxyProbe, error) {
	probe, err := startProxyProbe(nodes)
	if err != nil {
		log.Printf("proxy delay probe unavailable: %v", err)
		appendRuntimeLog("proxy delay probe unavailable: %v", err)
		return nil, err
	}
	return probe, nil
}

// appendRuntimeLog writes an app-side message into runtime/process.log so it
// shows up in the web UI "核心日志" panel together with the core logs.
func appendRuntimeLog(format string, args ...any) {
	path := filepath.Join(runtimeDirPath(), "process.log")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = fmt.Fprintf(f, "%s [app] %s\n", time.Now().Format("2006-01-02T15:04:05"), fmt.Sprintf(format, args...))
}
