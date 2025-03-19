// src/components/AuthForm.js
import React, { useState } from "react";
import { useNavigate, Link } from "react-router-dom";

function AuthForm({ isLogin, setToken }) {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const navigate = useNavigate();

  const handleSubmit = async (e) => {
    e.preventDefault();
    const endpoint = isLogin ? "http://localhost:8080/login" : "http://localhost:8080/register";

    const response = await fetch(endpoint, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ username, password }),
    });

    const data = await response.json();
    if (response.ok) {
      setToken(data.token);
      localStorage.setItem("token", data.token);
      navigate("/dashboard");
    } else {
      setError(data.error || "An error occurred");
    }
  };

  return (
    <div className="flex items-center justify-center min-h-screen bg-gray-900">
      <div className="bg-gray-800 p-10 rounded-xl shadow-2xl w-full max-w-md transform transition-all hover:scale-105">
        <h2 className="text-4xl font-extrabold text-center text-white mb-6">
          {isLogin ? "Login" : "Register"}
        </h2>
        {error && (
          <p className="text-red-500 text-sm text-center mb-4">{error}</p>
        )}
        <form onSubmit={handleSubmit} className="space-y-6">
          <div>
            <label className="block text-sm font-medium text-gray-400 mb-2">Username</label>
            <input
              type="text"
              placeholder="Enter your username"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              className="w-full px-4 py-3 border border-gray-600 rounded-lg bg-gray-700 text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
              required
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-400 mb-2">Password</label>
            <input
              type="password"
              placeholder="Enter your password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="w-full px-4 py-3 border border-gray-600 rounded-lg bg-gray-700 text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
              required
            />
          </div>
          <button
            type="submit"
            className="w-full bg-blue-600 text-white py-3 rounded-lg font-semibold hover:bg-blue-700 transition-all"
          >
            {isLogin ? "Login" : "Register"}
          </button>
        </form>
        <p className="mt-6 text-center text-gray-400">
          {isLogin ? (
            <>Don't have an account? <Link to="/register" className="text-blue-400 hover:underline font-semibold">Sign up</Link></>
          ) : (
            <>Already have an account? <Link to="/login" className="text-blue-400 hover:underline font-semibold">Login</Link></>
          )}
        </p>
      </div>
    </div>
  );
}

export default AuthForm;
