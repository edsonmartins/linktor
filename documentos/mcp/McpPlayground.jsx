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
      <div style={{ color: "#8b8fa3", fontStyle: "italic", fontSize: "13px" }}>
        Esta tool não requer parâmetros
      </div>
    );
  }

  const required = schema.required || [];

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: "12px" }}>
      {Object.entries(schema.properties).map(([key, prop]) => {
        const isRequired = required.includes(key);
        return (
          <div key={key}>
            <label
              style={{
                display: "flex",
                alignItems: "center",
                gap: "6px",
                marginBottom: "4px",
                fontSize: "12px",
                fontWeight: 600,
                color: "#c4c7d4",
                textTransform: "uppercase",
                letterSpacing: "0.5px",
              }}
            >
              {key}
              {isRequired && (
                <span
                  style={{
                    color: "#f47067",
                    fontSize: "9px",
                    background: "rgba(244,112,103,0.12)",
                    padding: "1px 5px",
                    borderRadius: "3px",
                    fontWeight: 700,
                  }}
                >
                  REQUIRED
                </span>
              )}
              <span
                style={{
                  color: "#6b6f85",
                  fontWeight: 400,
                  fontSize: "11px",
                  textTransform: "none",
                  letterSpacing: 0,
                }}
              >
                {prop.type}
              </span>
            </label>
            {prop.description && (
              <div
                style={{
                  fontSize: "12px",
                  color: "#6b6f85",
                  marginBottom: "4px",
                }}
              >
                {prop.description}
              </div>
            )}
            {prop.type === "boolean" ? (
              <button
                onClick={() => onChange(key, !values[key])}
                style={{
                  background: values[key]
                    ? "rgba(63,185,130,0.15)"
                    : "rgba(107,111,133,0.1)",
                  border: `1px solid ${values[key] ? "#3fb982" : "#2a2d3e"}`,
                  color: values[key] ? "#3fb982" : "#6b6f85",
                  padding: "6px 14px",
                  borderRadius: "6px",
                  cursor: "pointer",
                  fontSize: "13px",
                  fontWeight: 600,
                  fontFamily: "inherit",
                  transition: "all 0.15s",
                }}
              >
                {values[key] ? "true" : "false"}
              </button>
            ) : prop.enum ? (
              <select
                value={values[key] || ""}
                onChange={(e) => onChange(key, e.target.value)}
                style={{
                  width: "100%",
                  padding: "8px 10px",
                  background: "#1a1c2e",
                  border: "1px solid #2a2d3e",
                  borderRadius: "6px",
                  color: "#e0e2ed",
                  fontSize: "13px",
                  fontFamily: "'JetBrains Mono', monospace",
                  outline: "none",
                }}
              >
                <option value="">Selecionar...</option>
                {prop.enum.map((v) => (
                  <option key={v} value={v}>
                    {v}
                  </option>
                ))}
              </select>
            ) : prop.type === "object" || prop.type === "array" ? (
              <textarea
                value={values[key] || ""}
                onChange={(e) => onChange(key, e.target.value)}
                placeholder={`JSON ${prop.type}...`}
                rows={3}
                style={{
                  width: "100%",
                  padding: "8px 10px",
                  background: "#1a1c2e",
                  border: "1px solid #2a2d3e",
                  borderRadius: "6px",
                  color: "#e0e2ed",
                  fontSize: "13px",
                  fontFamily: "'JetBrains Mono', monospace",
                  outline: "none",
                  resize: "vertical",
                  boxSizing: "border-box",
                }}
              />
            ) : (
              <input
                type={prop.type === "number" || prop.type === "integer" ? "number" : "text"}
                value={values[key] || ""}
                onChange={(e) => onChange(key, e.target.value)}
                placeholder={prop.default != null ? `Default: ${prop.default}` : `Enter ${key}...`}
                style={{
                  width: "100%",
                  padding: "8px 10px",
                  background: "#1a1c2e",
                  border: "1px solid #2a2d3e",
                  borderRadius: "6px",
                  color: "#e0e2ed",
                  fontSize: "13px",
                  fontFamily: "'JetBrains Mono', monospace",
                  outline: "none",
                  boxSizing: "border-box",
                }}
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
      // Parse JSON fields and numbers
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
    <div
      style={{
        background: expanded
          ? "linear-gradient(135deg, #13152a 0%, #181b30 100%)"
          : "#13152a",
        border: `1px solid ${expanded ? "#3d5afe" : "#1e2038"}`,
        borderRadius: "10px",
        overflow: "hidden",
        transition: "all 0.25s ease",
        boxShadow: expanded ? "0 4px 24px rgba(61,90,254,0.08)" : "none",
      }}
    >
      {/* Header */}
      <div
        onClick={() => setExpanded(!expanded)}
        style={{
          padding: "14px 18px",
          cursor: "pointer",
          display: "flex",
          alignItems: "center",
          gap: "12px",
          userSelect: "none",
        }}
      >
        <div
          style={{
            width: "8px",
            height: "8px",
            borderRadius: "50%",
            background: "#3fb982",
            flexShrink: 0,
            boxShadow: "0 0 8px rgba(63,185,130,0.4)",
          }}
        />
        <div style={{ flex: 1, minWidth: 0 }}>
          <div
            style={{
              fontFamily: "'JetBrains Mono', monospace",
              fontSize: "14px",
              fontWeight: 700,
              color: "#e0e2ed",
            }}
          >
            {tool.name}
          </div>
          {tool.description && (
            <div
              style={{
                fontSize: "12px",
                color: "#6b6f85",
                marginTop: "2px",
                whiteSpace: "nowrap",
                overflow: "hidden",
                textOverflow: "ellipsis",
              }}
            >
              {tool.description}
            </div>
          )}
        </div>
        <div style={{ display: "flex", alignItems: "center", gap: "8px" }}>
          <span
            style={{
              fontSize: "11px",
              color: "#6b6f85",
              background: "#1a1c2e",
              padding: "2px 8px",
              borderRadius: "4px",
            }}
          >
            {paramCount} params{requiredCount > 0 && ` · ${requiredCount} req`}
          </span>
          <span
            style={{
              color: "#6b6f85",
              fontSize: "16px",
              transform: expanded ? "rotate(90deg)" : "rotate(0deg)",
              transition: "transform 0.2s",
            }}
          >
            ▸
          </span>
        </div>
      </div>

      {/* Expanded Content */}
      {expanded && (
        <div
          style={{
            padding: "0 18px 18px",
            borderTop: "1px solid #1e2038",
          }}
        >
          {/* Parameters */}
          <div style={{ marginTop: "14px" }}>
            <SchemaForm
              schema={tool.inputSchema}
              values={values}
              onChange={handleChange}
            />
          </div>

          {/* Execute Button */}
          <button
            onClick={handleExecute}
            disabled={executing}
            style={{
              marginTop: "14px",
              width: "100%",
              padding: "10px",
              background: executing
                ? "rgba(61,90,254,0.3)"
                : "linear-gradient(135deg, #3d5afe 0%, #536dfe 100%)",
              border: "none",
              borderRadius: "8px",
              color: "#fff",
              fontSize: "13px",
              fontWeight: 700,
              cursor: executing ? "wait" : "pointer",
              fontFamily: "inherit",
              letterSpacing: "0.3px",
              transition: "all 0.15s",
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
              gap: "8px",
            }}
          >
            {executing ? (
              <>
                <span
                  style={{
                    display: "inline-block",
                    width: "14px",
                    height: "14px",
                    border: "2px solid rgba(255,255,255,0.3)",
                    borderTopColor: "#fff",
                    borderRadius: "50%",
                    animation: "spin 0.6s linear infinite",
                  }}
                />
                Executando...
              </>
            ) : (
              <>▶ Executar</>
            )}
          </button>

          {/* Result */}
          {result && (
            <div style={{ marginTop: "14px" }}>
              <div
                style={{
                  display: "flex",
                  alignItems: "center",
                  justifyContent: "space-between",
                  marginBottom: "6px",
                }}
              >
                <span
                  style={{
                    fontSize: "11px",
                    fontWeight: 700,
                    color: result.error ? "#f47067" : "#3fb982",
                    textTransform: "uppercase",
                    letterSpacing: "0.5px",
                  }}
                >
                  {result.error ? "✕ Error" : "✓ Response"}
                </span>
                {elapsed != null && (
                  <span style={{ fontSize: "11px", color: "#6b6f85" }}>
                    {elapsed}ms
                  </span>
                )}
              </div>
              <pre
                style={{
                  background: "#0d0f1e",
                  border: `1px solid ${result.error ? "rgba(244,112,103,0.2)" : "#1e2038"}`,
                  borderRadius: "8px",
                  padding: "12px",
                  margin: 0,
                  fontSize: "12px",
                  fontFamily: "'JetBrains Mono', monospace",
                  color: "#c4c7d4",
                  overflow: "auto",
                  maxHeight: "300px",
                  whiteSpace: "pre-wrap",
                  wordBreak: "break-word",
                }}
              >
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
    <div style={{ display: "flex", flexDirection: "column", gap: "6px" }}>
      {resources.map((r, i) => (
        <div
          key={i}
          style={{
            background: "#13152a",
            border: "1px solid #1e2038",
            borderRadius: "8px",
            padding: "12px 16px",
            display: "flex",
            alignItems: "center",
            gap: "10px",
          }}
        >
          <span style={{ fontSize: "16px" }}>📄</span>
          <div>
            <div
              style={{
                fontFamily: "'JetBrains Mono', monospace",
                fontSize: "13px",
                color: "#e0e2ed",
                fontWeight: 600,
              }}
            >
              {r.name || r.uri}
            </div>
            {r.description && (
              <div style={{ fontSize: "12px", color: "#6b6f85", marginTop: "2px" }}>
                {r.description}
              </div>
            )}
            {r.uri && (
              <div
                style={{
                  fontSize: "11px",
                  color: "#3d5afe",
                  fontFamily: "'JetBrains Mono', monospace",
                  marginTop: "2px",
                }}
              >
                {r.uri}
              </div>
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
    <div
      style={{
        fontFamily:
          "'DM Sans', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif",
        background: "linear-gradient(180deg, #0d0f1e 0%, #111328 100%)",
        borderRadius: "14px",
        border: "1px solid #1e2038",
        overflow: "hidden",
        maxWidth: "100%",
        color: "#e0e2ed",
      }}
    >
      <style>{`
        @import url('https://fonts.googleapis.com/css2?family=DM+Sans:wght@400;500;600;700&family=JetBrains+Mono:wght@400;500;600;700&display=swap');
        @keyframes spin { to { transform: rotate(360deg) } }
        @keyframes pulse { 0%,100% { opacity:1 } 50% { opacity:0.5 } }
      `}</style>

      {/* ── Header ──────────────────────────────────────────── */}
      <div
        style={{
          padding: "20px 24px",
          borderBottom: "1px solid #1e2038",
          background:
            "linear-gradient(135deg, rgba(61,90,254,0.06) 0%, rgba(63,185,130,0.04) 100%)",
        }}
      >
        <div
          style={{
            display: "flex",
            alignItems: "center",
            justifyContent: "space-between",
            flexWrap: "wrap",
            gap: "12px",
          }}
        >
          <div style={{ display: "flex", alignItems: "center", gap: "12px" }}>
            <div
              style={{
                width: "36px",
                height: "36px",
                borderRadius: "10px",
                background:
                  "linear-gradient(135deg, #3d5afe 0%, #536dfe 100%)",
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
                fontSize: "18px",
                boxShadow: "0 2px 12px rgba(61,90,254,0.3)",
              }}
            >
              🔌
            </div>
            <div>
              <h3
                style={{
                  margin: 0,
                  fontSize: "16px",
                  fontWeight: 700,
                  color: "#e0e2ed",
                }}
              >
                {config.serverName}
              </h3>
              <div style={{ fontSize: "12px", color: "#6b6f85" }}>
                MCP Playground
              </div>
            </div>
          </div>

          <div style={{ display: "flex", alignItems: "center", gap: "10px" }}>
            {mcp.connected && (
              <div
                style={{
                  display: "flex",
                  alignItems: "center",
                  gap: "6px",
                  fontSize: "12px",
                  color: "#3fb982",
                  background: "rgba(63,185,130,0.1)",
                  padding: "4px 10px",
                  borderRadius: "6px",
                  fontWeight: 600,
                }}
              >
                <span
                  style={{
                    width: "6px",
                    height: "6px",
                    borderRadius: "50%",
                    background: "#3fb982",
                    boxShadow: "0 0 6px rgba(63,185,130,0.6)",
                  }}
                />
                Conectado
              </div>
            )}
            <button
              onClick={mcp.connect}
              disabled={mcp.loading}
              style={{
                padding: "8px 16px",
                background: mcp.connected
                  ? "rgba(63,185,130,0.1)"
                  : "linear-gradient(135deg, #3d5afe 0%, #536dfe 100%)",
                border: mcp.connected
                  ? "1px solid rgba(63,185,130,0.3)"
                  : "none",
                borderRadius: "8px",
                color: mcp.connected ? "#3fb982" : "#fff",
                fontSize: "13px",
                fontWeight: 600,
                cursor: mcp.loading ? "wait" : "pointer",
                fontFamily: "inherit",
                transition: "all 0.15s",
              }}
            >
              {mcp.loading
                ? "Conectando..."
                : mcp.connected
                  ? "⟳ Reconectar"
                  : "Conectar"}
            </button>
          </div>
        </div>

        {/* Config panel */}
        {showConfig && (
          <div
            style={{
              marginTop: "14px",
              display: "flex",
              gap: "10px",
              flexWrap: "wrap",
            }}
          >
            <input
              type="text"
              value={config.serverUrl}
              onChange={(e) =>
                setConfig((c) => ({ ...c, serverUrl: e.target.value }))
              }
              placeholder="Server URL"
              style={{
                flex: "1 1 250px",
                padding: "8px 12px",
                background: "#1a1c2e",
                border: "1px solid #2a2d3e",
                borderRadius: "6px",
                color: "#e0e2ed",
                fontSize: "13px",
                fontFamily: "'JetBrains Mono', monospace",
                outline: "none",
                boxSizing: "border-box",
              }}
            />
            <input
              type="password"
              value={config.apiKey}
              onChange={(e) =>
                setConfig((c) => ({ ...c, apiKey: e.target.value }))
              }
              placeholder="API Key (opcional)"
              style={{
                flex: "0 1 200px",
                padding: "8px 12px",
                background: "#1a1c2e",
                border: "1px solid #2a2d3e",
                borderRadius: "6px",
                color: "#e0e2ed",
                fontSize: "13px",
                fontFamily: "'JetBrains Mono', monospace",
                outline: "none",
                boxSizing: "border-box",
              }}
            />
          </div>
        )}

        {/* Error */}
        {mcp.error && (
          <div
            style={{
              marginTop: "12px",
              padding: "10px 14px",
              background: "rgba(244,112,103,0.08)",
              border: "1px solid rgba(244,112,103,0.2)",
              borderRadius: "8px",
              fontSize: "12px",
              color: "#f47067",
              fontFamily: "'JetBrains Mono', monospace",
            }}
          >
            ✕ {mcp.error}
          </div>
        )}
      </div>

      {/* ── Tabs ───────────────────────────────────────────── */}
      {mcp.connected && (
        <>
          <div
            style={{
              display: "flex",
              borderBottom: "1px solid #1e2038",
              padding: "0 24px",
            }}
          >
            {tabs.map((tab) => (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id)}
                style={{
                  padding: "12px 16px",
                  background: "none",
                  border: "none",
                  borderBottom:
                    activeTab === tab.id
                      ? "2px solid #3d5afe"
                      : "2px solid transparent",
                  color: activeTab === tab.id ? "#e0e2ed" : "#6b6f85",
                  fontSize: "13px",
                  fontWeight: 600,
                  cursor: "pointer",
                  fontFamily: "inherit",
                  display: "flex",
                  alignItems: "center",
                  gap: "6px",
                  transition: "all 0.15s",
                }}
              >
                <span>{tab.icon}</span>
                {tab.label}
                <span
                  style={{
                    background:
                      activeTab === tab.id
                        ? "rgba(61,90,254,0.2)"
                        : "#1a1c2e",
                    color: activeTab === tab.id ? "#8fa4ff" : "#6b6f85",
                    padding: "1px 7px",
                    borderRadius: "10px",
                    fontSize: "11px",
                    fontWeight: 700,
                  }}
                >
                  {tab.count}
                </span>
              </button>
            ))}
          </div>

          {/* ── Content ─────────────────────────────────────── */}
          <div style={{ padding: "18px 24px 24px" }}>
            {/* Search */}
            {activeTab === "tools" && mcp.tools.length > 3 && (
              <input
                type="text"
                value={filter}
                onChange={(e) => setFilter(e.target.value)}
                placeholder="🔍 Filtrar tools..."
                style={{
                  width: "100%",
                  padding: "10px 14px",
                  background: "#1a1c2e",
                  border: "1px solid #2a2d3e",
                  borderRadius: "8px",
                  color: "#e0e2ed",
                  fontSize: "13px",
                  fontFamily: "inherit",
                  outline: "none",
                  marginBottom: "14px",
                  boxSizing: "border-box",
                }}
              />
            )}

            {/* Tool List */}
            {activeTab === "tools" && (
              <div
                style={{
                  display: "flex",
                  flexDirection: "column",
                  gap: "8px",
                }}
              >
                {filteredTools.length === 0 ? (
                  <div
                    style={{
                      textAlign: "center",
                      padding: "40px",
                      color: "#6b6f85",
                    }}
                  >
                    {filter
                      ? "Nenhuma tool encontrada"
                      : "Nenhuma tool registrada"}
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
              <div
                style={{
                  display: "flex",
                  flexDirection: "column",
                  gap: "6px",
                }}
              >
                {mcp.prompts.length === 0 ? (
                  <div
                    style={{
                      textAlign: "center",
                      padding: "40px",
                      color: "#6b6f85",
                    }}
                  >
                    Nenhum prompt registrado
                  </div>
                ) : (
                  mcp.prompts.map((p, i) => (
                    <div
                      key={i}
                      style={{
                        background: "#13152a",
                        border: "1px solid #1e2038",
                        borderRadius: "8px",
                        padding: "12px 16px",
                      }}
                    >
                      <div
                        style={{
                          fontFamily: "'JetBrains Mono', monospace",
                          fontSize: "13px",
                          color: "#e0e2ed",
                          fontWeight: 600,
                        }}
                      >
                        💬 {p.name}
                      </div>
                      {p.description && (
                        <div
                          style={{
                            fontSize: "12px",
                            color: "#6b6f85",
                            marginTop: "4px",
                          }}
                        >
                          {p.description}
                        </div>
                      )}
                    </div>
                  ))
                )}
              </div>
            )}
          </div>
        </>
      )}

      {/* ── Empty State ────────────────────────────────────── */}
      {!mcp.connected && !mcp.loading && !mcp.error && (
        <div
          style={{
            padding: "60px 24px",
            textAlign: "center",
            color: "#6b6f85",
          }}
        >
          <div style={{ fontSize: "40px", marginBottom: "14px" }}>🔌</div>
          <div style={{ fontSize: "15px", fontWeight: 600, color: "#8b8fa3" }}>
            Configure a URL e clique em Conectar
          </div>
          <div style={{ fontSize: "13px", marginTop: "6px" }}>
            O playground irá descobrir automaticamente tools, resources e prompts
          </div>
        </div>
      )}
    </div>
  );
}
