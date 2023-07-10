import React, { useState } from "react";
import { Login } from "../svc/svc";

const LoginPage = () => {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState(null);

  const handleLogin = async (e) => {
    e.preventDefault();
    setError(null);
    Login(email, password).then((res) => {
      if (res) {
        window.location.pathname = '/'
      } else {
        setError(res);
      }
    });
  };

  return (
    <div className="flex flex-col items-center justify-center min-h-screen bg-gray-100">
      <form
        onSubmit={handleLogin}
        className="flex flex-col max-w-md w-full px-6 py-8 bg-white border-2 border-gray-300 rounded-lg"
      >
        <h2 className="text-2xl font-semibold mb-6">Login</h2>
        <input
          type="email"
          className="border border-gray-300 px-3 py-2 mb-4 rounded-md"
          placeholder="Email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          autoComplete="email"
        />
        <input
          type="password"
          className="border border-gray-300 px-3 py-2 mb-4 rounded-md"
          placeholder="Password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          autoComplete="password"
        />
        {error && <div>{error}</div>}
        <button
          className="bg-blue-500 hover:bg-blue-600 text-white px-4 py-2 rounded-md"
          onClick={handleLogin}
        >
          Login
        </button>
      </form>
    </div>
  );
};

export default LoginPage;
