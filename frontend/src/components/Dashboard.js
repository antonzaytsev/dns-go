import React from 'react';
import { RefreshCw, AlertCircle } from 'lucide-react';
import { useMetrics, useHealth, useRecentRequests } from '../hooks/useMetrics';
import OverviewCards from './OverviewCards';
import Charts from './Charts';
import QueryTypes from './QueryTypes';
import TopClients from './TopClients';
import RecentRequests from './RecentRequests';
import ConnectionStatus from './ConnectionStatus';
import DNSMappings from './dns-mappings';

const Dashboard = () => {
  const { metrics, loading, error, lastUpdated, refresh } = useMetrics(5000);
  const { isHealthy } = useHealth(30000);
  const { recentRequests, loading: requestsLoading, error: requestsError } = useRecentRequests(5000);

  return (
    <div className="min-h-screen bg-gray-100">
      {/* Header */}
      <header className="bg-white shadow-sm border-b border-gray-200">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between h-16">
            <div className="flex items-center">
              <h1 className="text-2xl font-bold text-gray-900">
                DNS Server Dashboard
              </h1>
            </div>
            <div className="flex items-center space-x-4">
              <ConnectionStatus 
                isOnline={isHealthy}
                lastUpdated={lastUpdated}
                error={error}
              />
              <button
                onClick={refresh}
                disabled={loading}
                className="inline-flex items-center px-3 py-2 border border-gray-300 shadow-sm text-sm leading-4 font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 disabled:opacity-50"
              >
                <RefreshCw className={`h-4 w-4 mr-2 ${loading ? 'animate-spin' : ''}`} />
                Refresh
              </button>
            </div>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {error && (
          <div className="mb-6 bg-red-50 border border-red-200 rounded-md p-4">
            <div className="flex">
              <AlertCircle className="h-5 w-5 text-red-400" />
              <div className="ml-3">
                <h3 className="text-sm font-medium text-red-800">
                  Error loading metrics
                </h3>
                <div className="mt-2 text-sm text-red-700">
                  {error}
                </div>
              </div>
            </div>
          </div>
        )}

        <div className="space-y-8">
          {/* Overview Cards */}
          <section>
            <h2 className="text-lg font-medium text-gray-900 mb-4">Overview</h2>
            <OverviewCards overview={metrics?.overview} />
          </section>

          {/* Charts */}
          <section>
            <h2 className="text-lg font-medium text-gray-900 mb-4">Request Patterns</h2>
            <Charts timeSeriesData={metrics?.time_series} />
          </section>

          {/* Query Types and Top Clients */}
          <section className="grid grid-cols-1 lg:grid-cols-2 gap-8">
            <QueryTypes queryTypes={metrics?.query_types} />
            <TopClients clients={metrics?.top_clients} />
          </section>

          {/* DNS Mappings Management */}
          <section>
            <DNSMappings />
          </section>

          {/* Recent Requests */}
          <section>
            {requestsError && (
              <div className="mb-4 bg-red-50 border border-red-200 rounded-md p-4">
                <div className="flex">
                  <AlertCircle className="h-5 w-5 text-red-400" />
                  <div className="ml-3">
                    <h3 className="text-sm font-medium text-red-800">
                      Error loading recent requests
                    </h3>
                    <div className="mt-2 text-sm text-red-700">
                      {requestsError}
                    </div>
                  </div>
                </div>
              </div>
            )}
            <RecentRequests requests={recentRequests} loading={requestsLoading} />
          </section>
        </div>
      </main>
    </div>
  );
};

export default Dashboard;
