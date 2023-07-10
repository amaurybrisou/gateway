import React, { useEffect, useState } from "react";
import { Route, Routes } from "react-router-dom";

import HomePage from "./routes/Home";
import Layout from "./routes/Layout";
import LoginPage from "./routes/LoginPage";

import { getUser, setToken } from './svc/svc';

setToken();


const App = () => {
  const [user, setUser] = useState(null)
  useEffect(() => {
    getUser().then(setUser)
  }, [])

  return (
    <Routes>
      <Route path="/" element={<Layout user={user}/>}>
        <Route index element={<HomePage />}/>
        <Route path="login" element={<LoginPage />}/>
      </Route>
    </Routes>
  );
};

export default App;