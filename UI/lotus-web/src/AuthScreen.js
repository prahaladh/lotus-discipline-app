import React, { useState } from "react";
import api from "./api";

export default function AuthScreen({ onAuth }) {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [isRegister, setIsRegister] = useState(false);

  const handleAuth = async (e) => {
    e.preventDefault();

    const endpoint = isRegister ? "/register" : "/login";

    try {
      const habits = [
        { name: "Lotus Sit", goalMinutes: 20, unit: "minutes" }
      ];

      const payload = isRegister ? { username, password, habits } : { username, password };

      const res = await api.post(endpoint, payload);

      localStorage.setItem("lotus_token", res.data.token);
      onAuth(res.data.token);

    } catch (err) {
      alert(err.response?.data?.error || "Backend connection failed.");
    }
  };

  return (
    <div className="flex items-center justify-center min-h-screen p-4">
      <form
        onSubmit={handleAuth}
        className="w-full max-w-md bg-white p-8 rounded-3xl shadow-lg border"
      >
        <h2 className="text-3xl font-bold text-center text-indigo-600 mb-6 italic">
          Lotus Discipline
        </h2>

        <input
          type="text"
          placeholder="Username"
          required
          className="w-full p-3 rounded-xl border mb-4"
          value={username}
          onChange={(e) => setUsername(e.target.value)}
        />

        <input
          type="password"
          placeholder="Password"
          required
          className="w-full p-3 rounded-xl border mb-4"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
        />

        <button className="w-full bg-indigo-600 text-white py-3 rounded-xl font-bold hover:bg-indigo-700">
          {isRegister ? "Start Journey" : "Enter Path"}
        </button>

        <button
          type="button"
          className="w-full mt-4 text-sm underline"
          onClick={() => setIsRegister(!isRegister)}
        >
          {isRegister ? "Already registered? Sign In" : "New user? Register"}
        </button>
      </form>
    </div>
  );
}