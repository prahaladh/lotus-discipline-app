import React, { useState, useEffect } from "react";
import AuthScreen from "./AuthScreen";
import Dashboard from "./Dashboard";

export default function App() {
  const [token, setToken] = useState(localStorage.getItem("lotus_token"));
  const [view, setView] = useState(token ? "dashboard" : "auth");

  const logout = () => {
    localStorage.removeItem("lotus_token");
    setToken(null);
    setView("auth");
  };

  const handleAuth = (token) => {
    setToken(token);
    setView("dashboard");
  };

  return (
    <div className="min-h-screen bg-slate-50 text-slate-900 font-sans">
      {view === "auth" ? (
        <AuthScreen onAuth={handleAuth} />
      ) : (
        <Dashboard onLogout={logout} />
      )}
    </div>
  );
}