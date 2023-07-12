import React from 'react';
import { Link, Outlet } from 'react-router-dom';
import { Logout } from '../../svc/svc';

function Layout(props) {
  // const navigate = useNavigate();
  return (
    <div>
      <nav className="bg-gray-800">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between h-16">
            <div className="flex items-center">
              <div className="flex-shrink-0">
                <Link to="/" className="text-white">
                  Service Gateway
                </Link>
              </div>
            </div>
            <Consumer className="hidden md:block">
              <div className="ml-10 flex items-baseline space-x-4">
                {!props.user && (
                  <Link
                    to="/login"
                    className="text-gray-300 hover:bg-gray-700 hover:text-white px-3 py-2 rounded-md text-sm font-medium"
                  >
                    Login
                  </Link>
                )}
                {props.user && (
                  <div className="text-gray-300 px-3 py-2 rounded-md text-sm font-medium grid grid-cols-2 gap-4">
                    <span>
                    Welcome, {props.user.firstname}
                    </span>
                    <Link onClick={() => Logout() }>Logout</Link>
                  </div>
                )}
              </div>
            </Consumer>
          </div>
        </div>
      </nav>
      <Outlet />
    </div>
  );
}

export default Layout;
