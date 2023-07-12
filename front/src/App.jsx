import React from "react";
import { Route, Routes } from "react-router-dom";

import UserProvider from "./context/user";
import HomePage from "./routes/Home";
import Layout from "./routes/Layout";
import LoginPage from "./routes/LoginPage";

const App = () => {
  return (
    <UserProvider>
      <Routes>
        <Route path="/" element={<Layout />}>
          <Route index element={<HomePage />}/>
          <Route path="login" element={<LoginPage />}/>
        </Route>
      </Routes>
    </UserProvider>
  );
};

export default App;