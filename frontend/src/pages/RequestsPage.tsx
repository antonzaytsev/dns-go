import React from 'react';
import { AlertCircle } from 'lucide-react';
import { useRequests } from '../hooks/useMetrics.ts';
import Requests from '../components/requests/Requests.tsx';
import Navigation from '../components/shared/Navigation.tsx';
import type { RequestsFullHeightProps } from '../types/index.ts';

const RequestsPage: React.FC = () => {
  const { requests, loading, error } = useRequests(5000);

  return (
    <div className="min-h-screen bg-gray-100">
      {/* Header */}
      <header className="bg-white shadow-sm border-b border-gray-200">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between h-16">
            <div className="flex items-center space-x-8">
              <h1 className="text-2xl font-bold text-gray-900">
                DNS Server
              </h1>
              <Navigation />
            </div>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="h-[calc(100vh-4rem)] max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="h-full">
          {error && (
            <div className="mb-4 bg-red-50 border border-red-200 rounded-md p-4">
              <div className="flex">
                <AlertCircle className="h-5 w-5 text-red-400" />
                <div className="ml-3">
                  <h3 className="text-sm font-medium text-red-800">
                    Error loading requests
                  </h3>
                  <div className="mt-2 text-sm text-red-700">
                    {error}
                  </div>
                </div>
              </div>
            </div>
          )}
          <div className="h-full">
            <RequestsFullHeight requests={requests} loading={loading} />
          </div>
        </div>
      </main>
    </div>
  );
};

// Create a full-height version that wraps the existing Requests
const RequestsFullHeight: React.FC<RequestsFullHeightProps> = ({ requests, loading }) => {
  return (
    <div className="h-full">
      <Requests requests={requests} loading={loading} fullHeight={true} />
    </div>
  );
};

export default RequestsPage;
