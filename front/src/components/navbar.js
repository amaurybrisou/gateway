'user client'

import { Link } from 'react-router-dom';
import { useUser } from '../context/user';

export default function NavBar(){
    const { logout, user } = useUser()

    return(
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
              <div className="hidden md:block">
                <div className="ml-10 flex items-baseline space-x-4">
                  {!user && (
                    <Link
                      to="/login"
                      className="text-gray-300 hover:bg-gray-700 hover:text-white px-3 py-2 rounded-md text-sm font-medium"
                    >
                      Login
                    </Link>
                  )}
                  {user && (
                    <div className="text-gray-300 px-3 py-2 rounded-md text-sm font-medium grid grid-cols-2 gap-4">
                      <span>
                      Welcome, {user.firstname}
                      </span>
                      <Link to="" onClick={() => logout() }>Logout</Link>
                    </div>
                  )}
                </div>
              </div>
            </div>
          </div>
        </nav>
    )
}