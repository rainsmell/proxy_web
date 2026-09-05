package main

const pageHTML = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>V2Ray 控制台</title>
  <style>
    :root {
      color-scheme: light;
      --bg: #eef1f4;
      --surface: #ffffff;
      --surface-soft: #f7f9fb;
      --line: #d8dee7;
      --line-strong: #bcc7d4;
      --text: #141b2d;
      --subtle: #5f6b7a;
      --muted: #8490a1;
      --primary: #1958d8;
      --primary-ink: #0d3f9f;
      --primary-soft: #e8efff;
      --success: #0e7a43;
      --success-soft: #e8f7ef;
      --danger: #bd2f3a;
      --danger-soft: #fff0f2;
      --warning: #9a5b00;
      --warning-soft: #fff6df;
      --shadow: 0 10px 28px rgba(30, 41, 59, .08);
    }

    * { box-sizing: border-box; }
    html, body { min-height: 100%; }
    body {
      margin: 0;
      background: var(--bg);
      color: var(--text);
      font: 14px/1.48 "Segoe UI", system-ui, -apple-system, BlinkMacSystemFont, sans-serif;
      letter-spacing: 0;
      overflow-x: hidden;
    }

    .app {
      min-height: 100vh;
      display: grid;
      grid-template-rows: auto 1fr;
      max-width: 100vw;
      overflow-x: hidden;
    }
    .header {
      position: sticky;
      top: 0;
      z-index: 10;
      background: rgba(255, 255, 255, .94);
      border-bottom: 1px solid var(--line);
      backdrop-filter: blur(10px);
    }
    .header-inner {
      width: 100%;
      max-width: 1440px;
      margin: 0 auto;
      padding: 14px 22px;
      display: grid;
      grid-template-columns: minmax(220px, 1fr) auto;
      gap: 16px;
      align-items: center;
    }
    .title h1 {
      margin: 0;
      font-size: 20px;
      line-height: 1.2;
      font-weight: 760;
    }
    .title p {
      margin: 4px 0 0;
      color: var(--subtle);
      font-size: 13px;
    }
    .status-strip {
      display: flex;
      gap: 10px;
      align-items: center;
      justify-content: flex-end;
      flex-wrap: wrap;
    }
    .status-badge {
      min-height: 36px;
      display: inline-flex;
      align-items: center;
      gap: 8px;
      padding: 7px 11px;
      border: 1px solid var(--line);
      border-radius: 8px;
      background: var(--surface-soft);
      color: var(--subtle);
      font-weight: 650;
      white-space: nowrap;
    }
    .status-badge.ok {
      color: var(--success);
      border-color: #a8dbc1;
      background: var(--success-soft);
    }
    .status-badge.err {
      color: var(--danger);
      border-color: #f0b5bd;
      background: var(--danger-soft);
    }
    .dot {
      width: 8px;
      height: 8px;
      border-radius: 50%;
      background: var(--muted);
      flex: 0 0 auto;
    }
    .ok .dot { background: var(--success); }
    .err .dot { background: var(--danger); }
    .metric {
      min-height: 36px;
      display: inline-flex;
      gap: 10px;
      align-items: center;
      justify-content: flex-start;
      padding: 6px 10px;
      border: 1px solid var(--line);
      border-radius: 8px;
      background: #fff;
      color: var(--subtle);
      white-space: nowrap;
    }
    .metric span {
      min-width: 0;
      overflow: hidden;
      text-overflow: ellipsis;
    }
    .metric b { color: var(--text); font-size: 13px; }

    .layout {
      width: 100%;
      max-width: 1440px;
      margin: 0 auto;
      padding: 18px 22px 22px;
      display: grid;
      grid-template-columns: 340px minmax(0, 1fr);
      gap: 16px;
      align-items: start;
      overflow-x: hidden;
    }
    .side, .main {
      display: grid;
      gap: 14px;
      min-width: 0;
      max-width: 100%;
    }
    .panel {
      background: var(--surface);
      border: 1px solid var(--line);
      border-radius: 8px;
      box-shadow: var(--shadow);
      min-width: 0;
      max-width: 100%;
    }
    .panel-head {
      min-height: 48px;
      padding: 13px 15px;
      display: flex;
      align-items: center;
      gap: 10px;
      border-bottom: 1px solid var(--line);
    }
    .panel-head h2 {
      margin: 0;
      font-size: 15px;
      line-height: 1.2;
      font-weight: 760;
    }
    .panel-head .spacer { margin-left: auto; }
    .panel-body { padding: 14px 15px 15px; }
    .caption {
      color: var(--muted);
      font-size: 12px;
      margin-top: 3px;
    }

    .field-stack { display: grid; gap: 10px; }
    label.field {
      display: grid;
      gap: 5px;
      color: var(--subtle);
      font-size: 12px;
      font-weight: 650;
    }
    .field-row {
      display: grid;
      grid-template-columns: 1fr 1fr;
      gap: 10px;
    }
    input, select {
      width: 100%;
      min-width: 0;
      height: 38px;
      padding: 8px 10px;
      border: 1px solid var(--line-strong);
      border-radius: 6px;
      background: #fff;
      color: var(--text);
      outline: none;
      font: inherit;
    }
    input:focus, select:focus {
      border-color: var(--primary);
      box-shadow: 0 0 0 3px rgba(25, 88, 216, .13);
    }

    .actions {
      display: flex;
      gap: 8px;
      align-items: center;
      flex-wrap: wrap;
    }
    button {
      min-height: 36px;
      border: 1px solid var(--primary);
      border-radius: 6px;
      background: var(--primary);
      color: #fff;
      padding: 0 12px;
      cursor: pointer;
      font: inherit;
      font-weight: 700;
      white-space: nowrap;
    }
    button.secondary {
      background: var(--primary-soft);
      color: var(--primary-ink);
      border-color: #c9d8ff;
    }
    button.ghost {
      background: #fff;
      color: var(--text);
      border-color: var(--line-strong);
    }
    button.danger {
      background: var(--danger);
      color: #fff;
      border-color: var(--danger);
    }
    button:disabled {
      opacity: .55;
      cursor: not-allowed;
    }

    .message {
      min-height: 38px;
      display: flex;
      align-items: center;
      padding: 8px 10px;
      border-radius: 6px;
      border: 1px solid var(--line);
      background: var(--surface-soft);
      color: var(--subtle);
      font-size: 13px;
    }
    .message.ok {
      background: var(--success-soft);
      border-color: #b6dec8;
      color: var(--success);
    }
    .message.err {
      background: var(--danger-soft);
      border-color: #f0b5bd;
      color: var(--danger);
    }
    .message.warn {
      background: var(--warning-soft);
      border-color: #efd18b;
      color: var(--warning);
    }

    .sub-list {
      display: grid;
      gap: 8px;
    }
    .sub-item {
      border: 1px solid var(--line);
      border-radius: 7px;
      background: #fff;
      padding: 10px;
      display: grid;
      gap: 9px;
    }
    .sub-title {
      min-width: 0;
      font-weight: 760;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }
    .sub-url {
      color: var(--muted);
      font-size: 12px;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }
    .sub-meta {
      display: flex;
      align-items: center;
      gap: 8px;
      flex-wrap: wrap;
      color: var(--subtle);
      font-size: 12px;
    }
    .sub-item .actions {
      display: grid;
      grid-template-columns: 1fr 1fr;
      width: 100%;
    }

    .pill {
      display: inline-flex;
      align-items: center;
      gap: 5px;
      min-height: 24px;
      padding: 2px 8px;
      border-radius: 999px;
      border: 1px solid transparent;
      background: #eef2f6;
      color: #394556;
      font-size: 12px;
      font-weight: 650;
      white-space: nowrap;
    }
    .pill.blue {
      color: var(--primary-ink);
      background: var(--primary-soft);
      border-color: #d3defa;
    }
    .pill.green {
      color: var(--success);
      background: var(--success-soft);
      border-color: #b6dec8;
    }
    .pill.red {
      color: var(--danger);
      background: var(--danger-soft);
      border-color: #f0b5bd;
    }

    .workbar {
      display: grid;
      grid-template-columns: minmax(180px, 1fr) 150px auto;
      gap: 10px;
      align-items: center;
      padding: 12px;
      border-bottom: 1px solid var(--line);
      background: var(--surface-soft);
      border-radius: 8px 8px 0 0;
    }
    .workbar .actions {
      justify-content: flex-end;
    }
    .selected-line {
      padding: 10px 12px;
      display: grid;
      grid-template-columns: 1fr auto;
      gap: 10px;
      align-items: center;
      border-bottom: 1px solid var(--line);
      background: #fff;
    }
    .selected-node {
      min-width: 0;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
      color: var(--subtle);
    }
    .selected-node b { color: var(--text); }
    .progress-wrap {
      display: grid;
      gap: 6px;
      min-width: 190px;
    }
    .progress-text {
      color: var(--subtle);
      font-size: 12px;
      text-align: right;
    }
    .progress-track {
      height: 6px;
      overflow: hidden;
      border-radius: 999px;
      background: #e5e9ef;
    }
    .progress-fill {
      width: 0%;
      height: 100%;
      border-radius: inherit;
      background: var(--primary);
      transition: width .18s ease;
    }

    .table-wrap {
      overflow: auto;
      max-width: 100%;
    }
    .node-table {
      max-height: min(54vh, 560px);
      min-height: 320px;
    }
    table {
      width: 100%;
      min-width: 980px;
      border-collapse: separate;
      border-spacing: 0;
      background: #fff;
    }
    th, td {
      padding: 10px 12px;
      border-bottom: 1px solid #edf0f4;
      text-align: left;
      vertical-align: middle;
    }
    th {
      position: sticky;
      top: 0;
      z-index: 2;
      background: #f8fafc;
      color: #536172;
      font-size: 12px;
      font-weight: 760;
    }
    td { color: #273244; }
    tbody tr {
      cursor: pointer;
    }
    tbody tr:hover {
      background: #f8fbff;
    }
    tbody tr.selected {
      background: #eef5ff;
      box-shadow: inset 3px 0 0 var(--primary);
    }
    .radio-cell { width: 42px; }
    .node-name {
      max-width: 360px;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
      font-weight: 720;
    }
    .server {
      font: 12px/1.4 Consolas, "SFMono-Regular", Menlo, monospace;
      color: #475467;
    }
    .delay {
      font-variant-numeric: tabular-nums;
      white-space: nowrap;
      color: var(--muted);
    }
    .delay.good { color: var(--success); font-weight: 760; }
    .delay.slow { color: var(--warning); font-weight: 760; }
    .delay.bad { color: var(--danger); font-weight: 760; }
    .error-cell {
      max-width: 240px;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
      color: var(--danger);
      font-size: 12px;
    }
    .empty {
      padding: 34px 16px;
      text-align: center;
      color: var(--muted);
    }

    .bottom-grid {
      display: grid;
      grid-template-columns: minmax(0, 1fr) minmax(320px, .6fr);
      gap: 14px;
    }
    .conn-table {
      max-height: 260px;
      min-height: 180px;
    }
    .log-box {
      min-height: 180px;
      max-height: 260px;
      overflow: auto;
      padding: 12px;
      font: 12px/1.55 Consolas, "SFMono-Regular", Menlo, monospace;
      background: #111827;
      color: #e5e7eb;
      border-radius: 0 0 8px 8px;
      white-space: pre-wrap;
    }

    .toast {
      position: fixed;
      right: 18px;
      bottom: 18px;
      z-index: 30;
      max-width: min(440px, calc(100vw - 36px));
      display: none;
      padding: 11px 13px;
      border-radius: 8px;
      background: #111827;
      color: #fff;
      box-shadow: 0 18px 40px rgba(17, 24, 39, .22);
    }

    @media (max-width: 1120px) {
      .layout {
        grid-template-columns: 1fr;
      }
      .side {
        grid-template-columns: 1fr 1fr;
      }
      .table-wrap {
        max-height: none;
      }
    }
    @media (max-width: 760px) {
      .header-inner {
        grid-template-columns: 1fr;
      }
      .status-strip {
        justify-content: flex-start;
        min-width: 0;
        display: grid;
        grid-template-columns: 1fr;
        width: 100%;
      }
      .status-badge { grid-column: auto; }
      .metric { min-width: 0; }
      .layout {
        padding: 12px;
        overflow-x: hidden;
      }
      .side, .bottom-grid {
        grid-template-columns: 1fr;
      }
      .workbar {
        grid-template-columns: 1fr;
      }
      .workbar .actions, .selected-line {
        justify-content: stretch;
        grid-template-columns: 1fr;
      }
      .actions button {
        flex: 1 1 auto;
        min-width: 0;
      }
      .sub-item .actions {
        grid-template-columns: 1fr;
      }
      .progress-text {
        text-align: left;
      }
    }
  </style>
</head>
<body>
  <div class="app">
    <header class="header">
      <div class="header-inner">
        <div class="title">
          <h1>V2Ray 控制台</h1>
          <p>节点选择、订阅刷新、代理启动和运行观测</p>
        </div>
        <div class="status-strip">
          <strong id="run" class="status-badge"><span class="dot"></span><span>检查状态中...</span></strong>
          <div class="metric"><span>SOCKS5</span><b id="metric-socks">-</b></div>
          <div class="metric"><span>HTTP</span><b id="metric-http">-</b></div>
          <div class="metric"><span>节点</span><b id="metric-nodes">0</b></div>
        </div>
      </div>
    </header>

    <div class="layout">
      <aside class="side">
        <section class="panel">
          <div class="panel-head">
            <div>
              <h2>订阅源</h2>
              <div class="caption">添加后立即拉取节点</div>
            </div>
          </div>
          <div class="panel-body">
            <form id="sub-form" class="field-stack">
              <label class="field">订阅链接
                <input id="url" autocomplete="off" placeholder="https://example.com/subscribe">
              </label>
              <label class="field">名称
                <input id="name" autocomplete="off" placeholder="可选">
              </label>
              <button id="add-sub" type="submit">添加并拉取</button>
            </form>
            <div id="sub-message" class="message" style="margin-top:12px">等待操作</div>
          </div>
        </section>

        <section class="panel">
          <div class="panel-head">
            <div>
              <h2>订阅列表</h2>
              <div class="caption">刷新或删除已有订阅</div>
            </div>
          </div>
          <div class="panel-body">
            <div id="subs" class="sub-list">
              <div class="empty">正在加载订阅...</div>
            </div>
          </div>
        </section>

        <section class="panel">
          <div class="panel-head">
            <div>
              <h2>监听端口</h2>
              <div class="caption">修改后重新启动代理生效</div>
            </div>
          </div>
          <div class="panel-body">
            <div class="field-stack">
              <label class="field">监听地址
                <input id="addr">
              </label>
              <div class="field-row">
                <label class="field">SOCKS5
                  <input id="socks" type="number" min="1" max="65535">
                </label>
                <label class="field">HTTP
                  <input id="http" type="number" min="1" max="65535">
                </label>
              </div>
              <button id="save-config" class="secondary" type="button">保存端口</button>
              <div class="message warn">当前无登录认证，只适合受信任网络使用。</div>
            </div>
          </div>
        </section>
      </aside>

      <main class="main">
        <section class="panel">
          <div class="workbar">
            <input id="node-search" autocomplete="off" placeholder="搜索名称、服务器或协议">
            <select id="protocol-filter">
              <option value="">全部协议</option>
              <option value="vless">VLESS</option>
              <option value="vmess">VMess</option>
              <option value="trojan">Trojan</option>
              <option value="shadowsocks">Shadowsocks</option>
            </select>
            <div class="actions">
              <button id="test-all" class="secondary" type="button">全部测速</button>
              <button id="start-proxy" type="button">启动选中</button>
              <button id="stop-proxy" class="danger" type="button">停止代理</button>
            </div>
          </div>
          <div class="selected-line">
            <div id="selected-node" class="selected-node">未选择节点</div>
            <div class="progress-wrap">
              <div id="progress" class="progress-text">未测速</div>
              <div class="progress-track"><div id="progress-fill" class="progress-fill"></div></div>
            </div>
          </div>
          <div class="table-wrap node-table">
            <table>
              <thead>
                <tr>
                  <th class="radio-cell"></th>
                  <th>节点</th>
                  <th>协议</th>
                  <th>服务器</th>
                  <th>TCP</th>
                  <th>代理</th>
                  <th>错误</th>
                  <th>操作</th>
                </tr>
              </thead>
              <tbody id="nodes">
                <tr><td colspan="8" class="empty">正在加载节点...</td></tr>
              </tbody>
            </table>
          </div>
        </section>

        <div class="bottom-grid">
          <section class="panel">
            <div class="panel-head">
              <div>
                <h2>最近连接</h2>
                <div class="caption">来自 V2Ray access.log</div>
              </div>
            </div>
            <div class="table-wrap conn-table">
              <table>
                <thead><tr><th>来源</th><th>目标</th><th>状态</th><th>时间</th></tr></thead>
                <tbody id="connections"><tr><td colspan="4" class="empty">暂无连接记录</td></tr></tbody>
              </table>
            </div>
          </section>

          <section class="panel">
            <div class="panel-head">
              <div>
                <h2>核心日志</h2>
                <div class="caption">启动失败和节点错误会显示在这里</div>
              </div>
              <span class="spacer"></span>
              <button id="refresh-logs" class="ghost" type="button">刷新</button>
            </div>
            <pre id="logs" class="log-box">正在加载日志...</pre>
          </section>
        </div>
      </main>
    </div>
  </div>

  <div id="toast" class="toast"></div>

  <script>
  var selected = "";
  var busy = false;
  var app = {
    subscriptions: [],
    nodes: [],
    visibleNodes: [],
    config: null,
    status: null
  };

  function E(id) { return document.getElementById(id); }
  function text(v) { return String(v == null ? "" : v); }
  function clear(el) { while (el.firstChild) el.removeChild(el.firstChild); }
  function setText(id, value) { E(id).textContent = value; }
  function toast(message) {
    var t = E("toast");
    t.textContent = message;
    t.style.display = "block";
    clearTimeout(toast.timer);
    toast.timer = setTimeout(function () { t.style.display = "none"; }, 3800);
  }
  function message(id, value, type) {
    var el = E(id);
    el.textContent = value;
    el.className = "message" + (type ? " " + type : "");
  }
  function setBusy(value) {
    busy = value;
    ["add-sub", "test-all", "start-proxy", "stop-proxy", "save-config", "refresh-logs"].forEach(function (id) {
      E(id).disabled = value;
    });
  }
  function hostOf(raw) {
    try { return new URL(raw).host || raw; } catch (e) { return raw; }
  }
  function fmtTime(raw) {
    if (!raw || raw.indexOf("0001-01-01") === 0) return "-";
    try { return new Date(raw).toLocaleString(); } catch (e) { return raw; }
  }
  function findSelected() {
    for (var i = 0; i < app.nodes.length; i++) {
      if (app.nodes[i].id === selected) return app.nodes[i];
    }
    return null;
  }
  function setProgress(done, total, label) {
    var pct = total ? Math.round(done * 100 / total) : 0;
    E("progress-fill").style.width = pct + "%";
    setText("progress", label || (done + "/" + total));
  }
  function delayText(v) {
    if (!v) return "-";
    return v + " ms";
  }
  function delayClass(v) {
    if (!v) return "delay";
    if (v < 600) return "delay good";
    if (v < 1500) return "delay slow";
    return "delay bad";
  }

  async function request(path, options) {
    var controller = new AbortController();
    var timeout = setTimeout(function () { controller.abort(); }, 30000);
    try {
      var opts = options || {};
      opts.signal = controller.signal;
      var res = await fetch(path, opts);
      var data = {};
      try { data = await res.json(); } catch (e) {}
      if (!res.ok) throw new Error(data.error || res.statusText || String(res.status));
      return data;
    } catch (e) {
      if (e.name === "AbortError") throw new Error("请求超时");
      throw e;
    } finally {
      clearTimeout(timeout);
    }
  }

  async function loadStatus() {
    try {
      var st = await request("/api/proxy/status");
      app.status = st;
      var run = E("run");
      run.className = "status-badge" + (st.running ? " ok" : "");
      run.innerHTML = "<span class=\"dot\"></span><span></span>";
      run.querySelector("span:last-child").textContent = st.running ? "运行中: " + (st.nodeName || st.nodeId || "未命名节点") : "已停止";
    } catch (e) {
      var run = E("run");
      run.className = "status-badge err";
      run.innerHTML = "<span class=\"dot\"></span><span></span>";
      run.querySelector("span:last-child").textContent = "状态检查失败: " + e.message;
    }
  }

  async function loadConfig() {
    try {
      var c = await request("/api/config");
      app.config = c;
      E("addr").value = c.listenAddress || "0.0.0.0";
      E("socks").value = c.socksPort || 10808;
      E("http").value = c.httpPort || 10809;
      setText("metric-socks", c.socksPort || "-");
      setText("metric-http", c.httpPort || "-");
    } catch (e) {
      message("sub-message", "配置加载失败: " + e.message, "err");
    }
  }

  async function loadSubscriptions() {
    try {
      app.subscriptions = await request("/api/subscriptions");
      renderSubscriptions();
    } catch (e) {
      message("sub-message", "订阅加载失败: " + e.message, "err");
    }
  }

  async function loadNodes() {
    try {
      app.nodes = await request("/api/nodes");
      if (selected && !findSelected()) selected = "";
      renderNodes();
      renderSelected();
    } catch (e) {
      var body = E("nodes");
      clear(body);
      var tr = body.insertRow();
      var td = tr.insertCell();
      td.colSpan = 8;
      td.className = "empty";
      td.textContent = "节点加载失败: " + e.message;
    }
  }

  async function loadConnections() {
    try {
      var rows = await request("/api/connections");
      var body = E("connections");
      clear(body);
      if (!rows.length) {
        var empty = body.insertRow();
        var td = empty.insertCell();
        td.colSpan = 4;
        td.className = "empty";
        td.textContent = "暂无连接记录";
        return;
      }
      rows.slice().reverse().forEach(function (x) {
        var tr = body.insertRow();
        tr.insertCell().textContent = text(x.source);
        tr.insertCell().textContent = text(x.target);
        tr.insertCell().textContent = text(x.status);
        tr.insertCell().textContent = text(x.time);
      });
    } catch (e) {}
  }

  async function loadLogs() {
    try {
      var rows = await request("/api/proxy/logs");
      E("logs").textContent = rows.length ? rows.join("\n") : "暂无核心错误日志";
    } catch (e) {
      E("logs").textContent = "日志加载失败: " + e.message;
    }
  }

  function renderSubscriptions() {
    var box = E("subs");
    clear(box);
    if (!app.subscriptions.length) {
      var empty = document.createElement("div");
      empty.className = "empty";
      empty.textContent = "暂无订阅";
      box.appendChild(empty);
      return;
    }
    app.subscriptions.forEach(function (s) {
      var item = document.createElement("div");
      item.className = "sub-item";

      var title = document.createElement("div");
      title.className = "sub-title";
      title.title = s.name || s.url;
      title.textContent = s.name || s.url;

      var url = document.createElement("div");
      url.className = "sub-url";
      url.title = s.url;
      url.textContent = hostOf(s.url);

      var meta = document.createElement("div");
      meta.className = "sub-meta";
      var count = document.createElement("span");
      count.className = "pill blue";
      count.textContent = (s.nodes ? s.nodes.length : 0) + " 个节点";
      var time = document.createElement("span");
      time.textContent = "更新: " + fmtTime(s.updatedAt);
      meta.appendChild(count);
      meta.appendChild(time);

      var actions = document.createElement("div");
      actions.className = "actions";
      actions.appendChild(actionButton("刷新", "refresh-sub", s.id, "secondary"));
      actions.appendChild(actionButton("删除", "delete-sub", s.id, "ghost"));

      item.appendChild(title);
      item.appendChild(url);
      item.appendChild(meta);
      item.appendChild(actions);
      box.appendChild(item);
    });
  }

  function renderNodes() {
    var query = E("node-search").value.trim().toLowerCase();
    var protocol = E("protocol-filter").value;
    app.visibleNodes = app.nodes.filter(function (n) {
      var q = [n.name, n.protocol, n.address, n.port].join(" ").toLowerCase();
      return (!protocol || n.protocol === protocol) && (!query || q.indexOf(query) >= 0);
    });
    setText("metric-nodes", app.nodes.length);

    var body = E("nodes");
    clear(body);
    if (!app.visibleNodes.length) {
      var empty = body.insertRow();
      var td = empty.insertCell();
      td.colSpan = 8;
      td.className = "empty";
      td.textContent = app.nodes.length ? "没有匹配的节点" : "暂无节点，请先添加订阅";
      return;
    }
    app.visibleNodes.forEach(function (n) {
      var tr = body.insertRow();
      tr.dataset.action = "select-node";
      tr.dataset.id = n.id;
      if (n.id === selected) tr.className = "selected";

      var radioCell = tr.insertCell();
      radioCell.className = "radio-cell";
      var radio = document.createElement("input");
      radio.type = "radio";
      radio.name = "node";
      radio.checked = n.id === selected;
      radio.dataset.action = "select-node";
      radio.dataset.id = n.id;
      radioCell.appendChild(radio);

      var name = tr.insertCell();
      name.className = "node-name";
      name.title = n.name || "未命名";
      name.textContent = n.name || "未命名";

      var protocol = tr.insertCell();
      var pill = document.createElement("span");
      pill.className = "pill";
      pill.textContent = n.protocol || "-";
      protocol.appendChild(pill);

      var server = tr.insertCell();
      server.className = "server";
      server.textContent = text(n.address) + ":" + text(n.port);

      var tcp = tr.insertCell();
      tcp.className = delayClass(n.tcpDelay);
      tcp.textContent = delayText(n.tcpDelay);

      var proxy = tr.insertCell();
      proxy.className = delayClass(n.proxyDelay);
      proxy.textContent = delayText(n.proxyDelay);

      var err = tr.insertCell();
      err.className = "error-cell";
      err.title = n.testError || "";
      err.textContent = n.testError || "";

      var op = tr.insertCell();
      op.appendChild(actionButton("测速", "test-node", n.id, "secondary"));
    });
  }

  function renderSelected() {
    var n = findSelected();
    if (!n) {
      setText("selected-node", "未选择节点");
      return;
    }
    var el = E("selected-node");
    el.innerHTML = "";
    var b = document.createElement("b");
    b.textContent = n.name || "未命名";
    el.appendChild(b);
    el.appendChild(document.createTextNode(" · " + (n.protocol || "-") + " · " + n.address + ":" + n.port));
  }

  function actionButton(label, action, id, cls) {
    var btn = document.createElement("button");
    btn.type = "button";
    btn.textContent = label;
    btn.dataset.action = action;
    btn.dataset.id = id;
    if (cls) btn.className = cls;
    return btn;
  }

  async function reloadAll() {
    await Promise.all([loadStatus(), loadConfig(), loadSubscriptions(), loadNodes(), loadLogs()]);
    loadConnections();
  }

  async function addSubscription(event) {
    event.preventDefault();
    if (busy) return;
    var url = E("url").value.trim();
    if (!url) {
      message("sub-message", "请输入订阅链接", "err");
      return;
    }
    setBusy(true);
    try {
      message("sub-message", "正在添加订阅...", "");
      var s = await request("/api/subscriptions", {
        method: "POST",
        headers: {"Content-Type": "application/json"},
        body: JSON.stringify({url: url, name: E("name").value.trim()})
      });
      message("sub-message", "订阅已添加，正在拉取节点...", "");
      var d = await request("/api/subscriptions/" + encodeURIComponent(s.id) + "/refresh", {method: "POST"});
      message("sub-message", "拉取完成: " + d.count + " 个节点", "ok");
      E("url").value = "";
      E("name").value = "";
      await reloadAll();
    } catch (e) {
      message("sub-message", "拉取失败: " + e.message, "err");
      toast(e.message);
    } finally {
      setBusy(false);
    }
  }

  async function refreshSubscription(id) {
    if (busy) return;
    setBusy(true);
    try {
      message("sub-message", "正在拉取订阅...", "");
      var d = await request("/api/subscriptions/" + encodeURIComponent(id) + "/refresh", {method: "POST"});
      message("sub-message", "拉取完成: " + d.count + " 个节点", "ok");
      await reloadAll();
    } catch (e) {
      message("sub-message", "拉取失败: " + e.message, "err");
      toast(e.message);
    } finally {
      setBusy(false);
    }
  }

  async function deleteSubscription(id) {
    if (busy || !confirm("确定删除这个订阅？")) return;
    setBusy(true);
    try {
      await request("/api/subscriptions/" + encodeURIComponent(id), {method: "DELETE"});
      message("sub-message", "订阅已删除", "ok");
      await reloadAll();
    } catch (e) {
      message("sub-message", "删除失败: " + e.message, "err");
    } finally {
      setBusy(false);
    }
  }

  async function testNode(id) {
    if (busy) return;
    setBusy(true);
    try {
      setProgress(0, 1, "正在测速 0/1");
      await request("/api/nodes/" + encodeURIComponent(id) + "/test", {method: "POST"});
      setProgress(1, 1, "测速完成 1/1");
      await loadNodes();
    } catch (e) {
      setProgress(0, 1, "测速失败");
      toast(e.message);
    } finally {
      setBusy(false);
    }
  }

  async function testAll() {
    if (busy) return;
    setBusy(true);
    try {
      var task = await request("/api/nodes/test-all", {method: "POST"});
      await pollTest(task.id);
    } catch (e) {
      setProgress(0, 1, "测速失败");
      toast(e.message);
    } finally {
      setBusy(false);
    }
  }

  async function pollTest(id) {
    while (true) {
      var p = await request("/api/nodes/test-progress?id=" + encodeURIComponent(id));
      setProgress(p.done, p.total, "测速进度 " + p.done + "/" + p.total);
      await loadNodes();
      if (!p.running) {
        setProgress(p.done, p.total, "测速完成 " + p.done + "/" + p.total);
        return;
      }
      await new Promise(function (resolve) { setTimeout(resolve, 700); });
    }
  }

  async function startProxy() {
    if (busy) return;
    if (!selected) {
      toast("请选择节点");
      return;
    }
    setBusy(true);
    try {
      await request("/api/proxy/start", {
        method: "POST",
        headers: {"Content-Type": "application/json"},
        body: JSON.stringify({nodeId: selected})
      });
      toast("代理已启动");
      await Promise.all([loadStatus(), loadLogs()]);
    } catch (e) {
      toast(e.message);
      await Promise.all([loadStatus(), loadLogs()]);
    } finally {
      setBusy(false);
    }
  }

  async function stopProxy() {
    if (busy) return;
    setBusy(true);
    try {
      await request("/api/proxy/stop", {method: "POST"});
      toast("代理已停止");
      await loadStatus();
    } catch (e) {
      toast(e.message);
    } finally {
      setBusy(false);
    }
  }

  async function saveConfig() {
    if (busy) return;
    setBusy(true);
    try {
      await request("/api/config", {
        method: "PUT",
        headers: {"Content-Type": "application/json"},
        body: JSON.stringify({
          listenAddress: E("addr").value.trim() || "0.0.0.0",
          socksPort: Number(E("socks").value),
          httpPort: Number(E("http").value)
        })
      });
      toast("端口配置已保存");
      await loadConfig();
    } catch (e) {
      toast(e.message);
    } finally {
      setBusy(false);
    }
  }

  document.addEventListener("click", function (event) {
    var el = event.target.closest("[data-action]");
    if (!el) return;
    var action = el.dataset.action;
    var id = el.dataset.id;
    if (action === "select-node") {
      selected = id;
      renderNodes();
      renderSelected();
      return;
    }
    event.stopPropagation();
    if (action === "refresh-sub") refreshSubscription(id);
    if (action === "delete-sub") deleteSubscription(id);
    if (action === "test-node") testNode(id);
  });
  E("sub-form").addEventListener("submit", addSubscription);
  E("test-all").addEventListener("click", testAll);
  E("start-proxy").addEventListener("click", startProxy);
  E("stop-proxy").addEventListener("click", stopProxy);
  E("save-config").addEventListener("click", saveConfig);
  E("refresh-logs").addEventListener("click", loadLogs);
  E("node-search").addEventListener("input", function () { renderNodes(); });
  E("protocol-filter").addEventListener("change", function () { renderNodes(); });

  reloadAll();
  setInterval(loadStatus, 3000);
  setInterval(loadConnections, 2000);
  setInterval(loadLogs, 5000);
  </script>
</body>
</html>`
