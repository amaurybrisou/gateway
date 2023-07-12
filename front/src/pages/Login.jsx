'use client'
import { useUser } from "@/context/user";
import { useState } from "react";

export default function Login() {
  const [errorMsg, setErrorMsg] = useState("");

  const { login } = useUser();

  const handleLogin = async (evt) => {
    const res = await login(evt)
    setErrorMsg(res)
  };

  return (
    <div className="flex flex-col items-center justify-center min-h-screen bg-gray-100">
      <form
        onSubmit={handleLogin}
        className="flex flex-col max-w-md w-full px-6 py-8 bg-white border-2 border-gray-300 rounded-lg"
      >
        <label htmlFor="email" className="text-2xl font-semibold mb-6">Login</label>
        <input
          type="email"
          className="border border-gray-300 px-3 py-2 mb-4 rounded-md"
          placeholder="Email"
          id="email"
          name="email"
          autoComplete="email"
        />
        <input
          type="password"
          className="border border-gray-300 px-3 py-2 mb-4 rounded-md"
          placeholder="Password"
          id="password"
          name="password"
          autoComplete="password"
        />
        {errorMsg}
        <button
          className="bg-blue-500 hover:bg-blue-600 text-white px-4 py-2 rounded-md"
          type="submit"
        >
          Login
        </button>
      </form>
    </div>
  );
};
