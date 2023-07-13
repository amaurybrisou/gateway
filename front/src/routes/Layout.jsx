import React from 'react';
import { Outlet } from 'react-router-dom';
import NavBar from '../components/navbar';
import { useUser } from '../context/user';

function Layout(props) {
  const {user} = useUser()
  // const navigate = useNavigate();
  return (
    <div>
      <NavBar user={user}/>
      <Outlet />
    </div>
  );
}

export default Layout;
