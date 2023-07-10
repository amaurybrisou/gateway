import { ACCESS_TOKEN_KEY, API_URL } from "../constants";

const getServices = async () => {
  let url = "/services";
  if (hasToken) {
    url = "/auth/services";
  }

  const response = await fetch(API_URL + url, {
    headers: {
      Authorization: sessionStorage.getItem(ACCESS_TOKEN_KEY),
    },
  });

  if (response.ok) {
    const data = await response.json();
    return data;
  } else {
    // Logout()
  }

  return [];
};

var hasToken = false;
var user;

const getUser = async () => {
  if (!hasToken) return 
  
  const response = await fetch(API_URL + "/auth/user", {
    headers: {
      Authorization: sessionStorage.getItem(ACCESS_TOKEN_KEY),
    },
  });
  if (response.status === 403 || response.status === 401) {
    // Logout()
    return response.Error;
  } else {
     user = await response.json();
    return user;
  }
};

const Logout = () => {
  sessionStorage.removeItem(ACCESS_TOKEN_KEY);
  user = undefined
};

const setToken = () => {
  const token = sessionStorage.getItem(ACCESS_TOKEN_KEY);
  if (token) {
    hasToken =  true
    sessionStorage.setItem(ACCESS_TOKEN_KEY, token);
  }
};

const Login = async (email, password) => {
    const response = await fetch('/login', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ email, password }),
      });

      if (response.ok) {
        const data = await response.json()
        sessionStorage.setItem(ACCESS_TOKEN_KEY,  data.token)
        return true
      } else {
        return response.Error
      }
}

export { Login, Logout, getServices, getUser, setToken };
