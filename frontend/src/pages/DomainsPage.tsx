import React, { useState, useEffect } from 'react';
import { RefreshCw, AlertCircle } from 'lucide-react';
import { dnsApi } from '../services/api.ts';
import { useHealth } from '../hooks/useMetrics.ts';
import Navigation from '../components/shared/Navigation.tsx';
import ConnectionStatus from '../components/shared/ConnectionStatus.tsx';
import PageHeader from '../components/shared/PageHeader.tsx';
import type { DomainCount } from '../types/index.ts';

type TimeFilter = 'hour' | 'day' | 'week' | 'month' | null;

const DomainsPage: React.FC = () => {
  const [domains, setDomains] = useState<DomainCount[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);
  const [domainFilter, setDomainFilter] = useState<string>('');
  const [timeFilter, setTimeFilter] = useState<TimeFilter>('day');

  const { isHealthy } = useHealth(30000);

  const getTimeFilterDate = (filter: TimeFilter): Date | null => {
    if (!filter) return null;
    const now = new Date();
    switch (filter) {
      case 'hour':
        return new Date(now.getTime() - 60 * 60 * 1000);
      case 'day':
        return new Date(now.getTime() - 24 * 60 * 60 * 1000);
      case 'week':
        return new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000);
      case 'month':
        return new Date(now.getTime() - 30 * 24 * 60 * 60 * 1000);
      default:
        return null;
    }
  };

  const fetchDomains = async () => {
    try {
      setLoading(true);
      setError(null);
      const since = getTimeFilterDate(timeFilter);
      const response = await dnsApi.getDomainCounts(since, domainFilter);
      setDomains(response.domains || []);
      setLastUpdated(new Date());
    } catch (err) {
      setError('Failed to fetch domain counts');
      console.error('Error fetching domains:', err);
      setDomains([]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    const timeoutId = setTimeout(() => {
      fetchDomains();
    }, domainFilter ? 300 : 0);
    return () => clearTimeout(timeoutId);
  }, [timeFilter, domainFilter]);

  const formatNumber = (num: number): string => {
    if (num >= 1000000) {
      return (num / 1000000).toFixed(1) + 'M';
    } else if (num >= 1000) {
      return (num / 1000).toFixed(1) + 'K';
    }
    return num.toString();
  };

  const clearSearch = () => {
    setDomainFilter('');
  };

  const handleSearchChange = (value: string) => {
    setDomainFilter(value);
  };

  const getSubtitle = (): string => {
    if (loading) return 'Loading...';
    const count = domains?.length || 0;
    return `${count} ${count === 1 ? 'domain' : 'domains'} found`;
  };

  const getTimeFilterLabel = (filter: TimeFilter): string => {
    switch (filter) {
      case 'hour':
        return 'Last Hour';
      case 'day':
        return 'Last Day';
      case 'week':
        return 'Last Week';
      case 'month':
        return 'Last Month';
      default:
        return 'All Time';
    }
  };

  return (
    <div className="min-h-screen bg-gray-100">
      <header className="bg-white shadow-sm border-b border-gray-200">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between h-16">
            <div className="flex items-center space-x-8">
              <h1 className="text-2xl font-bold text-gray-900">
                DNS Server
              </h1>
              <Navigation />
            </div>
            <div className="flex items-center space-x-4">
              <ConnectionStatus
                isOnline={isHealthy}
                lastUpdated={lastUpdated}
                error={error}
              />
              <button
                onClick={fetchDomains}
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

      <main className="max-w-7xl mx-auto py-6 sm:px-6 lg:px-8">
        <PageHeader
          title="Domains"
          subtitle={getSubtitle()}
          searchValue={domainFilter}
          onSearchChange={handleSearchChange}
          searchPlaceholder="Filter by domain name..."
          onClearSearch={clearSearch}
        />

        <div className="mb-6 flex flex-wrap gap-2">
          <button
            onClick={() => setTimeFilter(null)}
            className={`px-4 py-2 text-sm font-medium rounded-md ${
              timeFilter === null
                ? 'bg-indigo-600 text-white'
                : 'bg-white text-gray-700 border border-gray-300 hover:bg-gray-50'
            }`}
          >
            All Time
          </button>
          <button
            onClick={() => setTimeFilter('hour')}
            className={`px-4 py-2 text-sm font-medium rounded-md ${
              timeFilter === 'hour'
                ? 'bg-indigo-600 text-white'
                : 'bg-white text-gray-700 border border-gray-300 hover:bg-gray-50'
            }`}
          >
            Last Hour
          </button>
          <button
            onClick={() => setTimeFilter('day')}
            className={`px-4 py-2 text-sm font-medium rounded-md ${
              timeFilter === 'day'
                ? 'bg-indigo-600 text-white'
                : 'bg-white text-gray-700 border border-gray-300 hover:bg-gray-50'
            }`}
          >
            Last Day
          </button>
          <button
            onClick={() => setTimeFilter('week')}
            className={`px-4 py-2 text-sm font-medium rounded-md ${
              timeFilter === 'week'
                ? 'bg-indigo-600 text-white'
                : 'bg-white text-gray-700 border border-gray-300 hover:bg-gray-50'
            }`}
          >
            Last Week
          </button>
          <button
            onClick={() => setTimeFilter('month')}
            className={`px-4 py-2 text-sm font-medium rounded-md ${
              timeFilter === 'month'
                ? 'bg-indigo-600 text-white'
                : 'bg-white text-gray-700 border border-gray-300 hover:bg-gray-50'
            }`}
          >
            Last Month
          </button>
        </div>

        {error && (
          <div className="mb-6 bg-red-50 border border-red-200 rounded-md p-4">
            <div className="flex">
              <AlertCircle className="h-5 w-5 text-red-400" />
              <div className="ml-3">
                <h3 className="text-sm font-medium text-red-800">Error</h3>
                <div className="mt-2 text-sm text-red-700">{error}</div>
              </div>
            </div>
          </div>
        )}

        <div className="bg-white shadow overflow-hidden sm:rounded-md">
          {loading ? (
            <div className="text-center py-12">
              <RefreshCw className="h-8 w-8 animate-spin mx-auto text-gray-400" />
              <p className="mt-2 text-sm text-gray-500">Loading domains...</p>
            </div>
          ) : domains.length === 0 ? (
            <div className="text-center py-12">
              <h3 className="mt-2 text-sm font-medium text-gray-900">No domains found</h3>
              <p className="mt-1 text-sm text-gray-500">
                {domainFilter || timeFilter
                  ? 'Try adjusting your filters.'
                  : 'No domains have been requested yet.'}
              </p>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="min-w-full divide-y divide-gray-200">
                <thead className="bg-gray-50">
                  <tr>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Domain
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Request Count
                    </th>
                  </tr>
                </thead>
                <tbody className="bg-white divide-y divide-gray-200">
                  {domains.map((domain, index) => (
                    <tr key={`${domain.domain}-${index}`} className="hover:bg-gray-50">
                      <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
                        {domain.domain}
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                        <span className="font-semibold">{formatNumber(domain.count)}</span>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      </main>
    </div>
  );
};

export default DomainsPage;

