export function getApiBase() {
  const envBase = import.meta.env.VITE_API_BASE;
  if (envBase) return envBase;

  // Local dev ergonomics:
  // - `docker compose up`: backend is exposed on host 8081 (ports: 8081:8080)
  // - `npm run dev`: frontend runs on 5173
  if (typeof window !== "undefined") {
    const { protocol, hostname, port } = window.location;
    const isLocalhost = hostname === "localhost" || hostname === "127.0.0.1";
    if (isLocalhost && port === "5173") {
      return `${protocol}//${hostname}:8081/api/v1`;
    }
    if (isLocalhost) {
      return `${protocol}//${hostname}:8080/api/v1`;
    }
    // Production-style default: same-origin reverse proxy.
    return `${window.location.origin}/api/v1`;
  }

  // Build-time fallback.
  return "http://localhost:8080/api/v1";
}

export function getWsBase(apiBase = getApiBase()) {
  const envWs = import.meta.env.VITE_WS_BASE;
  if (envWs) return envWs;
  return apiBase.replace(/^http(s?):\/\//, "ws$1://");
}

