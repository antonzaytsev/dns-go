import React from 'react';
import { Link, useLocation } from 'react-router-dom';
import { ExternalLink } from 'lucide-react';

const Navigation: React.FC = () => {
  const location = useLocation();

  const isActive = (path: string): boolean => {
    return location.pathname === path;
  };

  return (
    <nav className="flex space-x-8">
      <Link
        to="/"
        className={`inline-flex items-center px-1 pt-1 text-sm font-medium ${
          isActive('/')
            ? 'text-gray-900 border-b-2 border-indigo-500'
            : 'text-gray-500 hover:text-gray-700 hover:border-gray-300 border-b-2 border-transparent'
        }`}
      >
        Dashboard
      </Link>
      <Link
        to="/dns-mappings"
        className={`inline-flex items-center px-1 pt-1 text-sm font-medium ${
          isActive('/dns-mappings')
            ? 'text-gray-900 border-b-2 border-indigo-500'
            : 'text-gray-500 hover:text-gray-700 hover:border-gray-300 border-b-2 border-transparent'
        }`}
      >
        DNS Mappings
      </Link>
      <Link
        to="/requests"
        className={`inline-flex items-center px-1 pt-1 text-sm font-medium ${
          isActive('/requests')
            ? 'text-gray-900 border-b-2 border-indigo-500'
            : 'text-gray-500 hover:text-gray-700 hover:border-gray-300 border-b-2 border-transparent'
        }`}
      >
        Requests
      </Link>
      <Link
        to="/clients"
        className={`inline-flex items-center px-1 pt-1 text-sm font-medium ${
          isActive('/clients')
            ? 'text-gray-900 border-b-2 border-indigo-500'
            : 'text-gray-500 hover:text-gray-700 hover:border-gray-300 border-b-2 border-transparent'
        }`}
      >
        Clients
      </Link>
    </nav>
  );
};

export default Navigation;
