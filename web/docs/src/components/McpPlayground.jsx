import { useState, useEffect, useCallback, useRef } from "react";

// ─── Configuration ───────────────────────────────────────────────
const DEFAULT_CONFIG = {
  serverUrl: "http://localhost:3001/mcp",
  serverName: "MCP Server",
  apiKey: "",
};

// ─── JSON-RPC Helper ─────────────────────────────────────────────
let rpcId = 0;
const jsonRpc = (method, params = {}) => ({
  jsonrpc: "2.0",
  id: ++rpcId,
  method,
  params,
});

// ─── MCP Client Hook ────────────────────────────────────────────
function useMcpClient(serverUrl, apiKey) {
  const [tools, setTools] = useState([]);
  const [resources, setResources] = useState([]);
  const [prompts, setPrompts] = useState([]);
  const [connected, setConnected] = useState(false);
  const [error, setError] = useState(null);
  const [loading, setLoading] = useState(false);
  const sessionId = useRef(null);

  const call = useCallback(
    async (method, params = {}) => {
      const headers = {
        "Content-Type": "application/json",
      };
      if (apiKey) headers["Authorization"] = `Bearer ${apiKey}`;
      if (sessionId.current) headers["Mcp-Session-Id"] = sessionId.current;

      const res = await fetch(serverUrl, {
        method: "POST",
        headers,
        body: JSON.stringify(jsonRpc(method, params)),
      });

      const sid = res.headers.get("Mcp-Session-Id");
      if (sid) sessionId.current = sid;

      if (!res.ok) throw new Error(`HTTP ${res.status}: ${res.statusText}`);
      return res.json();
    },
    [serverUrl, apiKey]
  );

  const connect = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      // Initialize
      const initRes = await call("initialize", {
        protocolVersion: "2025-03-26",
        capabilities: {},
        clientInfo: { name: "mcp-playground", version: "1.0.0" },
      });

      if (initRes.error) throw new Error(initRes.error.message);

      // Send initialized notification
      await fetch(serverUrl, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          ...(apiKey ? { Authorization: `Bearer ${apiKey}` } : {}),
          ...(sessionId.current
            ? { "Mcp-Session-Id": sessionId.current }
            : {}),
        },
        body: JSON.stringify({
          jsonrpc: "2.0",
          method: "notifications/initialized",
        }),
      });

      // List tools
      const toolsRes = await call("tools/list");
      setTools(toolsRes.result?.tools || []);

      // List resources (may not be supported)
      try {
        const resRes = await call("resources/list");
        setResources(resRes.result?.resources || []);
      } catch {}

      // List prompts (may not be supported)
      try {
        const promptsRes = await call("prompts/list");
        setPrompts(promptsRes.result?.prompts || []);
      } catch {}

      setConnected(true);
    } catch (err) {
      setError(err.message);
      setConnected(false);
    } finally {
      setLoading(false);
    }
  }, [call, serverUrl, apiKey]);

  const callTool = useCallback(
    async (name, args) => {
      return call("tools/call", { name, arguments: args });
    },
    [call]
  );

  return {
    tools,
    resources,
    prompts,
    connected,
    error,
    loading,
    connect,
    callTool,
  };
}

// ─── Schema Form Generator ──────────────────────────────────────
function SchemaForm({ schema, values, onChange }) {
  if (!schema?.properties) {
    return (
      <div className="mcp-no-params">
        This tool requires no parameters
      </div>
    );
  }

  const required = schema.required || [];

  return (
    <div className="mcp-schema-form">
      {Object.entries(schema.properties).map(([key, prop]) => {
        const isRequired = required.includes(key);
        return (
          <div key={key} className="mcp-field">
            <label className="mcp-label">
              {key}
              {isRequired && (
                <span className="mcp-required">REQUIRED</span>
              )}
              <span className="mcp-type">{prop.type}</span>
            </label>
            {prop.description && (
              <div className="mcp-description">{prop.description}</div>
            )}
            {prop.type === "boolean" ? (
              <button
                onClick={() => onChange(key, !values[key])}
                className={`mcp-bool-btn ${values[key] ? 'active' : ''}`}
              >
                {values[key] ? "true" : "false"}
              </button>
            ) : prop.enum ? (
              <select
                value={values[key] || ""}
                onChange={(e) => onChange(key, e.target.value)}
                className="mcp-select"
              >
                <option value="">Select...</option>
                {prop.enum.map((v) => (
                  <option key={v} value={v}>{v}</option>
                ))}
              </select>
            ) : prop.type === "object" || prop.type === "array" ? (
              <textarea
                value={values[key] || ""}
                onChange={(e) => onChange(key, e.target.value)}
                placeholder={`JSON ${prop.type}...`}
                rows={3}
                className="mcp-textarea"
              />
            ) : (
              <input
                type={prop.type === "number" || prop.type === "integer" ? "number" : "text"}
                value={values[key] || ""}
                onChange={(e) => onChange(key, e.target.value)}
                placeholder={prop.default != null ? `Default: ${prop.default}` : `Enter ${key}...`}
                className="mcp-input"
              />
            )}
          </div>
        );
      })}
    </div>
  );
}

// ─── Tool Card ───────────────────────────────────────────────────
function ToolCard({ tool, onExecute }) {
  const [expanded, setExpanded] = useState(false);
  const [values, setValues] = useState({});
  const [result, setResult] = useState(null);
  const [executing, setExecuting] = useState(false);
  const [elapsed, setElapsed] = useState(null);

  const handleChange = (key, value) => {
    setValues((prev) => ({ ...prev, [key]: value }));
  };

  const handleExecute = async () => {
    setExecuting(true);
    setResult(null);
    const start = performance.now();
    try {
      const args = {};
      const props = tool.inputSchema?.properties || {};
      for (const [k, v] of Object.entries(values)) {
        if (!v && v !== false && v !== 0) continue;
        const propType = props[k]?.type;
        if (propType === "object" || propType === "array") {
          try {
            args[k] = JSON.parse(v);
          } catch {
            args[k] = v;
          }
        } else if (propType === "number" || propType === "integer") {
          args[k] = Number(v);
        } else if (propType === "boolean") {
          args[k] = v;
        } else {
          args[k] = v;
        }
      }
      const res = await onExecute(tool.name, args);
      setElapsed(Math.round(performance.now() - start));
      setResult(res);
    } catch (err) {
      setElapsed(Math.round(performance.now() - start));
      setResult({ error: { message: err.message } });
    } finally {
      setExecuting(false);
    }
  };

  const paramCount = Object.keys(tool.inputSchema?.properties || {}).length;
  const requiredCount = (tool.inputSchema?.required || []).length;

  return (
    <div className={`mcp-tool-card ${expanded ? 'expanded' : ''}`}>
      <div className="mcp-tool-header" onClick={() => setExpanded(!expanded)}>
        <div className="mcp-tool-dot" />
        <div className="mcp-tool-info">
          <div className="mcp-tool-name">{tool.name}</div>
          {tool.description && (
            <div className="mcp-tool-desc">{tool.description}</div>
          )}
        </div>
        <div className="mcp-tool-meta">
          <span className="mcp-tool-params">
            {paramCount} params{requiredCount > 0 && ` - ${requiredCount} req`}
          </span>
          <span className={`mcp-tool-arrow ${expanded ? 'expanded' : ''}`}>
            ▸
          </span>
        </div>
      </div>

      {expanded && (
        <div className="mcp-tool-body">
          <div className="mcp-tool-form">
            <SchemaForm
              schema={tool.inputSchema}
              values={values}
              onChange={handleChange}
            />
          </div>

          <button
            onClick={handleExecute}
            disabled={executing}
            className={`mcp-execute-btn ${executing ? 'loading' : ''}`}
          >
            {executing ? (
              <>
                <span className="mcp-spinner" />
                Executing...
              </>
            ) : (
              <>Execute</>
            )}
          </button>

          {result && (
            <div className="mcp-result">
              <div className="mcp-result-header">
                <span className={`mcp-result-status ${result.error ? 'error' : 'success'}`}>
                  {result.error ? "Error" : "Response"}
                </span>
                {elapsed != null && (
                  <span className="mcp-result-time">{elapsed}ms</span>
                )}
              </div>
              <pre className={`mcp-result-code ${result.error ? 'error' : ''}`}>
                {JSON.stringify(result, null, 2)}
              </pre>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

// ─── Resource List ───────────────────────────────────────────────
function ResourceList({ resources }) {
  if (!resources.length) return null;
  return (
    <div className="mcp-resource-list">
      {resources.map((r, i) => (
        <div key={i} className="mcp-resource-card">
          <span className="mcp-resource-icon">📄</span>
          <div className="mcp-resource-content">
            <div className="mcp-resource-name">{r.name || r.uri}</div>
            {r.description && (
              <div className="mcp-resource-desc">{r.description}</div>
            )}
            {r.uri && (
              <div className="mcp-resource-uri">{r.uri}</div>
            )}
          </div>
        </div>
      ))}
    </div>
  );
}

// ─── Main Playground Component ───────────────────────────────────
export default function McpPlayground({
  serverUrl: propUrl,
  serverName: propName,
  apiKey: propKey,
  showConfig = true,
}) {
  const [config, setConfig] = useState({
    serverUrl: propUrl || DEFAULT_CONFIG.serverUrl,
    serverName: propName || DEFAULT_CONFIG.serverName,
    apiKey: propKey || DEFAULT_CONFIG.apiKey,
  });
  const [activeTab, setActiveTab] = useState("tools");
  const [filter, setFilter] = useState("");

  const mcp = useMcpClient(config.serverUrl, config.apiKey);

  const filteredTools = mcp.tools.filter(
    (t) =>
      !filter ||
      t.name.toLowerCase().includes(filter.toLowerCase()) ||
      t.description?.toLowerCase().includes(filter.toLowerCase())
  );

  const tabs = [
    { id: "tools", label: "Tools", count: mcp.tools.length, icon: "⚡" },
    { id: "resources", label: "Resources", count: mcp.resources.length, icon: "📄" },
    { id: "prompts", label: "Prompts", count: mcp.prompts.length, icon: "💬" },
  ];

  return (
    <div className="mcp-playground">
      <style>{`
        .mcp-playground {
          --mcp-bg: #0d0f1e;
          --mcp-bg-card: #13152a;
          --mcp-bg-input: #1a1c2e;
          --mcp-border: #1e2038;
          --mcp-border-active: #7c3aed;
          --mcp-text: #e0e2ed;
          --mcp-text-muted: #6b6f85;
          --mcp-text-label: #c4c7d4;
          --mcp-primary: #7c3aed;
          --mcp-primary-hover: #8b5cf6;
          --mcp-success: #3fb982;
          --mcp-error: #f47067;
          --mcp-font-mono: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Monaco, Consolas, monospace;
          --mcp-font-sans: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;

          font-family: var(--mcp-font-sans);
          background: linear-gradient(180deg, var(--mcp-bg) 0%, #111328 100%);
          border-radius: 14px;
          border: 1px solid var(--mcp-border);
          overflow: hidden;
          max-width: 100%;
          color: var(--mcp-text);
        }

        @keyframes spin { to { transform: rotate(360deg) } }

        /* Header */
        .mcp-header {
          padding: 20px 24px;
          border-bottom: 1px solid var(--mcp-border);
          background: linear-gradient(135deg, rgba(124,58,237,0.06) 0%, rgba(63,185,130,0.04) 100%);
        }

        .mcp-header-top {
          display: flex;
          align-items: center;
          justify-content: space-between;
          flex-wrap: wrap;
          gap: 12px;
        }

        .mcp-header-left {
          display: flex;
          align-items: center;
          gap: 12px;
        }

        .mcp-header-icon {
          width: 36px;
          height: 36px;
          border-radius: 10px;
          background: linear-gradient(135deg, var(--mcp-primary) 0%, var(--mcp-primary-hover) 100%);
          display: flex;
          align-items: center;
          justify-content: center;
          font-size: 18px;
          box-shadow: 0 2px 12px rgba(124,58,237,0.3);
        }

        .mcp-header-title {
          margin: 0;
          font-size: 16px;
          font-weight: 700;
          color: var(--mcp-text);
        }

        .mcp-header-subtitle {
          font-size: 12px;
          color: var(--mcp-text-muted);
        }

        .mcp-header-right {
          display: flex;
          align-items: center;
          gap: 10px;
        }

        .mcp-connected-badge {
          display: flex;
          align-items: center;
          gap: 6px;
          font-size: 12px;
          color: var(--mcp-success);
          background: rgba(63,185,130,0.1);
          padding: 4px 10px;
          border-radius: 6px;
          font-weight: 600;
        }

        .mcp-connected-dot {
          width: 6px;
          height: 6px;
          border-radius: 50%;
          background: var(--mcp-success);
          box-shadow: 0 0 6px rgba(63,185,130,0.6);
        }

        .mcp-connect-btn {
          padding: 8px 16px;
          background: linear-gradient(135deg, var(--mcp-primary) 0%, var(--mcp-primary-hover) 100%);
          border: none;
          border-radius: 8px;
          color: #fff;
          font-size: 13px;
          font-weight: 600;
          cursor: pointer;
          font-family: inherit;
          transition: all 0.15s;
        }

        .mcp-connect-btn.connected {
          background: rgba(63,185,130,0.1);
          border: 1px solid rgba(63,185,130,0.3);
          color: var(--mcp-success);
        }

        .mcp-connect-btn:disabled {
          cursor: wait;
          opacity: 0.7;
        }

        /* Config panel */
        .mcp-config {
          margin-top: 14px;
          display: flex;
          gap: 10px;
          flex-wrap: wrap;
        }

        .mcp-config-input {
          flex: 1 1 250px;
          padding: 8px 12px;
          background: var(--mcp-bg-input);
          border: 1px solid var(--mcp-border);
          border-radius: 6px;
          color: var(--mcp-text);
          font-size: 13px;
          font-family: var(--mcp-font-mono);
          outline: none;
          box-sizing: border-box;
        }

        .mcp-config-input:focus {
          border-color: var(--mcp-border-active);
        }

        .mcp-config-key {
          flex: 0 1 200px;
        }

        /* Error */
        .mcp-error {
          margin-top: 12px;
          padding: 10px 14px;
          background: rgba(244,112,103,0.08);
          border: 1px solid rgba(244,112,103,0.2);
          border-radius: 8px;
          font-size: 12px;
          color: var(--mcp-error);
          font-family: var(--mcp-font-mono);
        }

        /* Tabs */
        .mcp-tabs {
          display: flex;
          border-bottom: 1px solid var(--mcp-border);
          padding: 0 24px;
        }

        .mcp-tab {
          padding: 12px 16px;
          background: none;
          border: none;
          border-bottom: 2px solid transparent;
          color: var(--mcp-text-muted);
          font-size: 13px;
          font-weight: 600;
          cursor: pointer;
          font-family: inherit;
          display: flex;
          align-items: center;
          gap: 6px;
          transition: all 0.15s;
        }

        .mcp-tab.active {
          border-bottom-color: var(--mcp-primary);
          color: var(--mcp-text);
        }

        .mcp-tab-count {
          background: var(--mcp-bg-input);
          color: var(--mcp-text-muted);
          padding: 1px 7px;
          border-radius: 10px;
          font-size: 11px;
          font-weight: 700;
        }

        .mcp-tab.active .mcp-tab-count {
          background: rgba(124,58,237,0.2);
          color: #a78bfa;
        }

        /* Content */
        .mcp-content {
          padding: 18px 24px 24px;
        }

        /* Search */
        .mcp-search {
          width: 100%;
          padding: 10px 14px;
          background: var(--mcp-bg-input);
          border: 1px solid var(--mcp-border);
          border-radius: 8px;
          color: var(--mcp-text);
          font-size: 13px;
          font-family: inherit;
          outline: none;
          margin-bottom: 14px;
          box-sizing: border-box;
        }

        .mcp-search:focus {
          border-color: var(--mcp-border-active);
        }

        /* Tool Card */
        .mcp-tool-list {
          display: flex;
          flex-direction: column;
          gap: 8px;
        }

        .mcp-tool-card {
          background: var(--mcp-bg-card);
          border: 1px solid var(--mcp-border);
          border-radius: 10px;
          overflow: hidden;
          transition: all 0.25s ease;
        }

        .mcp-tool-card.expanded {
          background: linear-gradient(135deg, #13152a 0%, #181b30 100%);
          border-color: var(--mcp-primary);
          box-shadow: 0 4px 24px rgba(124,58,237,0.08);
        }

        .mcp-tool-header {
          padding: 14px 18px;
          cursor: pointer;
          display: flex;
          align-items: center;
          gap: 12px;
          user-select: none;
        }

        .mcp-tool-dot {
          width: 8px;
          height: 8px;
          border-radius: 50%;
          background: var(--mcp-success);
          flex-shrink: 0;
          box-shadow: 0 0 8px rgba(63,185,130,0.4);
        }

        .mcp-tool-info {
          flex: 1;
          min-width: 0;
        }

        .mcp-tool-name {
          font-family: var(--mcp-font-mono);
          font-size: 14px;
          font-weight: 700;
          color: var(--mcp-text);
        }

        .mcp-tool-desc {
          font-size: 12px;
          color: var(--mcp-text-muted);
          margin-top: 2px;
          white-space: nowrap;
          overflow: hidden;
          text-overflow: ellipsis;
        }

        .mcp-tool-meta {
          display: flex;
          align-items: center;
          gap: 8px;
        }

        .mcp-tool-params {
          font-size: 11px;
          color: var(--mcp-text-muted);
          background: var(--mcp-bg-input);
          padding: 2px 8px;
          border-radius: 4px;
        }

        .mcp-tool-arrow {
          color: var(--mcp-text-muted);
          font-size: 16px;
          transition: transform 0.2s;
        }

        .mcp-tool-arrow.expanded {
          transform: rotate(90deg);
        }

        .mcp-tool-body {
          padding: 0 18px 18px;
          border-top: 1px solid var(--mcp-border);
        }

        .mcp-tool-form {
          margin-top: 14px;
        }

        /* Schema Form */
        .mcp-schema-form {
          display: flex;
          flex-direction: column;
          gap: 12px;
        }

        .mcp-no-params {
          color: var(--mcp-text-muted);
          font-style: italic;
          font-size: 13px;
        }

        .mcp-field {
        }

        .mcp-label {
          display: flex;
          align-items: center;
          gap: 6px;
          margin-bottom: 4px;
          font-size: 12px;
          font-weight: 600;
          color: var(--mcp-text-label);
          text-transform: uppercase;
          letter-spacing: 0.5px;
        }

        .mcp-required {
          color: var(--mcp-error);
          font-size: 9px;
          background: rgba(244,112,103,0.12);
          padding: 1px 5px;
          border-radius: 3px;
          font-weight: 700;
        }

        .mcp-type {
          color: var(--mcp-text-muted);
          font-weight: 400;
          font-size: 11px;
          text-transform: none;
          letter-spacing: 0;
        }

        .mcp-description {
          font-size: 12px;
          color: var(--mcp-text-muted);
          margin-bottom: 4px;
        }

        .mcp-input, .mcp-textarea, .mcp-select {
          width: 100%;
          padding: 8px 10px;
          background: var(--mcp-bg-input);
          border: 1px solid var(--mcp-border);
          border-radius: 6px;
          color: var(--mcp-text);
          font-size: 13px;
          font-family: var(--mcp-font-mono);
          outline: none;
          box-sizing: border-box;
        }

        .mcp-input:focus, .mcp-textarea:focus, .mcp-select:focus {
          border-color: var(--mcp-border-active);
        }

        .mcp-textarea {
          resize: vertical;
        }

        .mcp-bool-btn {
          background: rgba(107,111,133,0.1);
          border: 1px solid var(--mcp-border);
          color: var(--mcp-text-muted);
          padding: 6px 14px;
          border-radius: 6px;
          cursor: pointer;
          font-size: 13px;
          font-weight: 600;
          font-family: inherit;
          transition: all 0.15s;
        }

        .mcp-bool-btn.active {
          background: rgba(63,185,130,0.15);
          border-color: var(--mcp-success);
          color: var(--mcp-success);
        }

        /* Execute Button */
        .mcp-execute-btn {
          margin-top: 14px;
          width: 100%;
          padding: 10px;
          background: linear-gradient(135deg, var(--mcp-primary) 0%, var(--mcp-primary-hover) 100%);
          border: none;
          border-radius: 8px;
          color: #fff;
          font-size: 13px;
          font-weight: 700;
          cursor: pointer;
          font-family: inherit;
          letter-spacing: 0.3px;
          transition: all 0.15s;
          display: flex;
          align-items: center;
          justify-content: center;
          gap: 8px;
        }

        .mcp-execute-btn.loading {
          background: rgba(124,58,237,0.3);
          cursor: wait;
        }

        .mcp-spinner {
          display: inline-block;
          width: 14px;
          height: 14px;
          border: 2px solid rgba(255,255,255,0.3);
          border-top-color: #fff;
          border-radius: 50%;
          animation: spin 0.6s linear infinite;
        }

        /* Result */
        .mcp-result {
          margin-top: 14px;
        }

        .mcp-result-header {
          display: flex;
          align-items: center;
          justify-content: space-between;
          margin-bottom: 6px;
        }

        .mcp-result-status {
          font-size: 11px;
          font-weight: 700;
          text-transform: uppercase;
          letter-spacing: 0.5px;
        }

        .mcp-result-status.success {
          color: var(--mcp-success);
        }

        .mcp-result-status.error {
          color: var(--mcp-error);
        }

        .mcp-result-time {
          font-size: 11px;
          color: var(--mcp-text-muted);
        }

        .mcp-result-code {
          background: #0d0f1e;
          border: 1px solid var(--mcp-border);
          border-radius: 8px;
          padding: 12px;
          margin: 0;
          font-size: 12px;
          font-family: var(--mcp-font-mono);
          color: var(--mcp-text-label);
          overflow: auto;
          max-height: 300px;
          white-space: pre-wrap;
          word-break: break-word;
        }

        .mcp-result-code.error {
          border-color: rgba(244,112,103,0.2);
        }

        /* Resource List */
        .mcp-resource-list {
          display: flex;
          flex-direction: column;
          gap: 6px;
        }

        .mcp-resource-card {
          background: var(--mcp-bg-card);
          border: 1px solid var(--mcp-border);
          border-radius: 8px;
          padding: 12px 16px;
          display: flex;
          align-items: center;
          gap: 10px;
        }

        .mcp-resource-icon {
          font-size: 16px;
        }

        .mcp-resource-content {
        }

        .mcp-resource-name {
          font-family: var(--mcp-font-mono);
          font-size: 13px;
          color: var(--mcp-text);
          font-weight: 600;
        }

        .mcp-resource-desc {
          font-size: 12px;
          color: var(--mcp-text-muted);
          margin-top: 2px;
        }

        .mcp-resource-uri {
          font-size: 11px;
          color: var(--mcp-primary);
          font-family: var(--mcp-font-mono);
          margin-top: 2px;
        }

        /* Prompt List */
        .mcp-prompt-list {
          display: flex;
          flex-direction: column;
          gap: 6px;
        }

        .mcp-prompt-card {
          background: var(--mcp-bg-card);
          border: 1px solid var(--mcp-border);
          border-radius: 8px;
          padding: 12px 16px;
        }

        .mcp-prompt-name {
          font-family: var(--mcp-font-mono);
          font-size: 13px;
          color: var(--mcp-text);
          font-weight: 600;
        }

        .mcp-prompt-desc {
          font-size: 12px;
          color: var(--mcp-text-muted);
          margin-top: 4px;
        }

        /* Empty State */
        .mcp-empty {
          padding: 60px 24px;
          text-align: center;
          color: var(--mcp-text-muted);
        }

        .mcp-empty-icon {
          font-size: 40px;
          margin-bottom: 14px;
        }

        .mcp-empty-title {
          font-size: 15px;
          font-weight: 600;
          color: #8b8fa3;
        }

        .mcp-empty-desc {
          font-size: 13px;
          margin-top: 6px;
        }

        .mcp-no-items {
          text-align: center;
          padding: 40px;
          color: var(--mcp-text-muted);
        }
      `}</style>

      {/* Header */}
      <div className="mcp-header">
        <div className="mcp-header-top">
          <div className="mcp-header-left">
            <div className="mcp-header-icon">🔌</div>
            <div>
              <h3 className="mcp-header-title">{config.serverName}</h3>
              <div className="mcp-header-subtitle">MCP Playground</div>
            </div>
          </div>

          <div className="mcp-header-right">
            {mcp.connected && (
              <div className="mcp-connected-badge">
                <span className="mcp-connected-dot" />
                Connected
              </div>
            )}
            <button
              onClick={mcp.connect}
              disabled={mcp.loading}
              className={`mcp-connect-btn ${mcp.connected ? 'connected' : ''}`}
            >
              {mcp.loading
                ? "Connecting..."
                : mcp.connected
                  ? "⟳ Reconnect"
                  : "Connect"}
            </button>
          </div>
        </div>

        {showConfig && (
          <div className="mcp-config">
            <input
              type="text"
              value={config.serverUrl}
              onChange={(e) => setConfig((c) => ({ ...c, serverUrl: e.target.value }))}
              placeholder="Server URL"
              className="mcp-config-input"
            />
            <input
              type="password"
              value={config.apiKey}
              onChange={(e) => setConfig((c) => ({ ...c, apiKey: e.target.value }))}
              placeholder="API Key (optional)"
              className="mcp-config-input mcp-config-key"
            />
          </div>
        )}

        {mcp.error && (
          <div className="mcp-error">✕ {mcp.error}</div>
        )}
      </div>

      {/* Tabs */}
      {mcp.connected && (
        <>
          <div className="mcp-tabs">
            {tabs.map((tab) => (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id)}
                className={`mcp-tab ${activeTab === tab.id ? 'active' : ''}`}
              >
                <span>{tab.icon}</span>
                {tab.label}
                <span className="mcp-tab-count">{tab.count}</span>
              </button>
            ))}
          </div>

          {/* Content */}
          <div className="mcp-content">
            {/* Search */}
            {activeTab === "tools" && mcp.tools.length > 3 && (
              <input
                type="text"
                value={filter}
                onChange={(e) => setFilter(e.target.value)}
                placeholder="🔍 Filter tools..."
                className="mcp-search"
              />
            )}

            {/* Tool List */}
            {activeTab === "tools" && (
              <div className="mcp-tool-list">
                {filteredTools.length === 0 ? (
                  <div className="mcp-no-items">
                    {filter ? "No tools found" : "No tools registered"}
                  </div>
                ) : (
                  filteredTools.map((tool) => (
                    <ToolCard
                      key={tool.name}
                      tool={tool}
                      onExecute={mcp.callTool}
                    />
                  ))
                )}
              </div>
            )}

            {/* Resources */}
            {activeTab === "resources" && (
              <ResourceList resources={mcp.resources} />
            )}

            {/* Prompts */}
            {activeTab === "prompts" && (
              <div className="mcp-prompt-list">
                {mcp.prompts.length === 0 ? (
                  <div className="mcp-no-items">No prompts registered</div>
                ) : (
                  mcp.prompts.map((p, i) => (
                    <div key={i} className="mcp-prompt-card">
                      <div className="mcp-prompt-name">💬 {p.name}</div>
                      {p.description && (
                        <div className="mcp-prompt-desc">{p.description}</div>
                      )}
                    </div>
                  ))
                )}
              </div>
            )}
          </div>
        </>
      )}

      {/* Empty State */}
      {!mcp.connected && !mcp.loading && !mcp.error && (
        <div className="mcp-empty">
          <div className="mcp-empty-icon">🔌</div>
          <div className="mcp-empty-title">Configure the URL and click Connect</div>
          <div className="mcp-empty-desc">
            The playground will automatically discover tools, resources and prompts
          </div>
        </div>
      )}
    </div>
  );
}
