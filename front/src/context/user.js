"use client";
import { createContext, useContext, useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { ACCESS_TOKEN_KEY, API_URL } from "../constants";

const Context = createContext();

const Provider = ({ children }) => {
  const navigate = useNavigate();
  const [user, setUser] = useState(null);
  const [expiresAt, setExpiresAt] = useState(null);

  const getUserProfile = async () => {
    const token = sessionStorage.getItem(ACCESS_TOKEN_KEY);
    if (token) {
      const res = await fetch(API_URL + "/auth/user", {
        headers: {
          Authorization: "Bearer " + token,
        },
      });

      if (!res.ok) {
        throw new Error("failed to fetch user");
      }

      const profile = await res.json();
      setUser(profile);
    }
  };

  useEffect(() => {
    refreshTokenInt();
    getUserProfile();
  }, []);

  useEffect(() => {
    const token = sessionStorage.getItem(ACCESS_TOKEN_KEY);
    if (token && expiresAt) {
      const currentTime = Math.floor(Date.now() / 1000);
      const timeRemaining = expiresAt - currentTime;
      const refreshThreshold = 10; // Specify the delta in seconds
      const timeout = setTimeout(() => {
        refreshTokenInt();
      }, (timeRemaining - refreshThreshold) * 1000);
  
      return () => clearTimeout(timeout);
    }
  }, [expiresAt]);

  const login = async (event) => {
    event.preventDefault();
    const res = await fetch(API_URL + "/login", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        email: event.target.email.value,
        password: event.target.password.value,
      }),
    });

    if (!res.ok) {
      setUser(null);
      sessionStorage.removeItem(ACCESS_TOKEN_KEY);
      return res.statusText;
    } else {
      const data = await res.json();
      sessionStorage.setItem(ACCESS_TOKEN_KEY, data.token);
      setExpiresAt(data.expires_at);
      getUserProfile();
      navigate("/");
    }
  };

  const refreshTokenInt = async function () {
    const token = sessionStorage.getItem(ACCESS_TOKEN_KEY);
    if (token) {
      const res = await fetch(API_URL + "/auth/refresh-token", {
        credentials: "same-origin",
        headers: {
          Authorization: "Bearer " + token,
        },
      });

      if (!res.ok) {
        setUser(null);
        sessionStorage.removeItem(ACCESS_TOKEN_KEY);
        navigate("/");
        return;
      }

      const data = await res.json();
      sessionStorage.setItem(ACCESS_TOKEN_KEY, data.token);
      setExpiresAt(data.expires_at);
    }
  };

  const logout = async () => {
    const token = sessionStorage.getItem(ACCESS_TOKEN_KEY);
    if (token) {
      const res = await fetch(API_URL + "/auth/logout", {
        headers: {
          Authorization: "Bearer " + token,
        },
      });
      if (!res.ok) {
        return await res.Error;
      } else {
        setUser(null);
        sessionStorage.removeItem(ACCESS_TOKEN_KEY);
        navigate("/");
        return true;
      }
    }
  };

  const exposed = {
    user,
    login,
    logout,
  };

  return <Context.Provider value={exposed}>{children}</Context.Provider>;
};

export const useUser = () => useContext(Context);
export default Provider;
