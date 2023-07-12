"use client";
import { createContext, useContext, useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { ACCESS_TOKEN_KEY, API_URL } from "../constants";

const Context = createContext();

const Provider = ({ children }) => {
  const navigate = useNavigate();
  const [user, setUser] = useState(null);

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
    getUserProfile();
  }, []);

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
      return res.statusText;
    } else {
      const data = await res.json();
      sessionStorage.setItem(ACCESS_TOKEN_KEY, data.token);
      getUserProfile();
      navigate("/");
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
