import { useState } from "react";

export default function Topbar() {
  const [token, setToken] = useState(() => sessionStorage.getItem("atcloud.portalToken") || "");
  const [connected, setConnected] = useState(Boolean(token));

  function connect(event) {
    event.preventDefault();
    const trimmed = token.trim();
    if (!trimmed) return;
    sessionStorage.setItem("atcloud.portalToken", trimmed);
    setToken(trimmed);
    setConnected(true);
  }

  function disconnect() {
    sessionStorage.removeItem("atcloud.portalToken");
    setToken("");
    setConnected(false);
  }

  return (
    <header className="flex items-center justify-between gap-4 border-b bg-white px-6 py-3">
      <h1 className="font-semibold text-gray-800">Developer portal</h1>
      <form className="flex items-center gap-2" onSubmit={connect}>
        <label className="sr-only" htmlFor="portal-token">Portal access token</label>
        <input
          id="portal-token"
          className="w-64 rounded border px-3 py-2 text-sm"
          type="password"
          autoComplete="off"
          placeholder="Portal access token"
          value={token}
          onChange={(event) => {
            setToken(event.target.value);
            setConnected(false);
          }}
        />
        {connected ? (
          <button className="rounded border px-3 py-2 text-sm" type="button" onClick={disconnect}>
            Disconnect
          </button>
        ) : (
          <button className="rounded bg-blue-700 px-3 py-2 text-sm text-white" type="submit">
            Connect
          </button>
        )}
      </form>
    </header>
  );
}