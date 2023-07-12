import React from 'react';
import { Outlet } from 'react-router-dom';
import NavBar from '../components/navbar';
import { API_URL } from '../constants';
import { useUser } from '../context/user';

async function getUser(){
  const res = await fetch(API_URL+'/auth/user',{
    credentials: "same-origin"
  })
  if (!res.ok){
    return {}
  }

  return await res.json()
}

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
