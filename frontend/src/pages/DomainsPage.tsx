import React, { useState, useEffect, useRef } from 'react';
import { useSearchParams } from 'react-router-dom';
import { RefreshCw, AlertCircle, Filter, Globe, X, Search, RotateCcw } from 'lucide-react';
import { dnsApi } from '../services/api.ts';
import { useHealth } from '../hooks/useMetrics.ts';
import Navigation from '../components/shared/Navigation.tsx';
import ConnectionStatus from '../components/shared/ConnectionStatus.tsx';
import type { DomainCount } from '../types/index.ts';

type TimeFilter = 'hour' | 'day' | 'week' | 'month' | 'all';

const DomainsPage: React.FC = () => {
  const { isHealthy } = useHealth(30000);
  const [searchParams, setSearchParams] = useSearchParams();

  // State
  const [domains, setDomains] = useState<DomainCount[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);
  const [domainFilter, setDomainFilter] = useState<string>('');
  const [clientIPFilter, setClientIPFilter] = useState<string>('');
  const [timeFilter, setTimeFilter] = useState<TimeFilter>('day');
  const initializedRef = useRef<boolean>(false);

  const getTimeFilterDate = (filter: TimeFilter): Date | null => {
    if (filter === 'all') return null;
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

  // Update URL when filters change
  const updateURL = (domain: string, client: string, time: TimeFilter): void => {
    if (!initializedRef.current) return;

    const params = new URLSearchParams();
    if (domain) params.set('domain', domain);
    if (client) params.set('client', client);
    if (time !== 'day') params.set('period', time);

    setSearchParams(params, { replace: true });
  };

  // Fetch domains
  const fetchDomains = async (
    domain: string = domainFilter,
    client: string = clientIPFilter,
    time: TimeFilter = timeFilter,
    updateURLParams: boolean = true
  ): Promise<void> => {
    setLoading(true);
    setError(null);

    try {
      const since = getTimeFilterDate(time);
      const response = await dnsApi.getDomainCounts(domain, client, since);
      setDomains(response.domains || []);
      setLastUpdated(new Date());

      if (updateURLParams) {
        updateURL(domain, client, time);
      }
    } catch (err) {
      setError('Failed to fetch domain counts');
      console.error('Error fetching domains:', err);
      setDomains([]);
    } finally {
      setLoading(false);
    }
  };

  // Load filters from URL on mount
  useEffect(() => {
    if (initializedRef.current) return;

    const urlDomain = searchParams.get('domain') || '';
    const urlClient = searchParams.get('client') || '';
    const urlPeriod = (searchParams.get('period') as TimeFilter) || 'day';

    setDomainFilter(urlDomain);
    setClientIPFilter(urlClient);
    setTimeFilter(urlPeriod);
    initializedRef.current = true;

    fetchDomains(urlDomain, urlClient, urlPeriod, false);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const handleFiltersSubmit = (): void => {
    fetchDomains(domainFilter, clientIPFilter, timeFilter, true);
  };

  const clearFilters = (): void => {
    setDomainFilter('');
    setClientIPFilter('');
    setTimeFilter('day');
    fetchDomains('', '', 'day', true);
  };

  const hasActiveFilters = (): boolean => {
    return domainFilter !== '' || clientIPFilter !== '' || timeFilter !== 'day';
  };

  const handleTimeFilterChange = (newTime: TimeFilter): void => {
    setTimeFilter(newTime);
    fetchDomains(domainFilter, clientIPFilter, newTime, true);
  };

  const formatNumber = (num: number): string => {
    if (num >= 1000000) {
      return (num / 1000000).toFixed(1) + 'M';
    } else if (num >= 1000) {
      return (num / 1000).toFixed(1) + 'K';
    }
    return num.toString();
  };

  const getSubtitle = (): string => {
    if (loading) return 'Loading...';
    const count = domains?.length || 0;
    return `${count.toLocaleString()} ${count === 1 ? 'domain' : 'domains'}`;
  };

  const getTimeFilterLabel = (filter: TimeFilter): string => {
    switch (filter) {
      case 'hour': return 'Last Hour';
      case 'day': return 'Last Day';
      case 'week': return 'Last Week';
      case 'month': return 'Last Month';
      case 'all': return 'All Time';
      default: return 'Last Day';
    }
  };

  return (
    <div className="min-h-screen bg-gray-100">
      <header className="bg-white shadow-sm border-b border-gray-200">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between h-16">
            <div className="flex items-center space-x-8">
              <h1 className="text-2xl font-bold text-gray-900">DNS Server</h1>
              <Navigation />
            </div>
            <div className="flex items-center space-x-4">
              <ConnectionStatus
                isOnline={isHealthy}
                lastUpdated={lastUpdated}
                error={error}
              />
              <button
                onClick={() => fetchDomains(domainFilter, clientIPFilter, timeFilter, false)}
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
        {/* Page Header */}
        <div className="mb-6">
          <h1 className="text-3xl font-bold text-gray-900 mb-2">Domains</h1>
          <p className="text-sm text-gray-600">{getSubtitle()}</p>
        </div>

        {/* Filters Block */}
        <div className="mb-6 bg-white rounded-lg shadow-md p-4">
          <div className="flex items-center mb-4">
            <Filter className="h-5 w-5 text-gray-500 mr-2" />
            <h2 className="text-lg font-semibold text-gray-900">Filters</h2>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4">
            {/* Domain Filter */}
            <div>
              <label htmlFor="domain-filter" className="block text-sm font-medium text-gray-700 mb-2">
                Domain
              </label>
              <div className="relative">
                <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                  <Globe className="h-5 w-5 text-gray-400" />
                </div>
                <input
                  id="domain-filter"
                  type="text"
                  placeholder="e.g., google.com"
                  value={domainFilter}
                  onChange={(e) => setDomainFilter(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') {
                      e.preventDefault();
                      handleFiltersSubmit();
                    }
                  }}
                  className="block w-full pl-10 pr-10 py-2 border border-gray-300 rounded-md shadow-sm placeholder-gray-400 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 text-sm"
                />
                {domainFilter && (
                  <button
                    type="button"
                    onClick={() => setDomainFilter('')}
                    className="absolute inset-y-0 right-0 pr-3 flex items-center text-gray-400 hover:text-gray-600"
                  >
                    <X className="h-4 w-4" />
                  </button>
                )}
              </div>
            </div>

            {/* Client IP Filter */}
            <div>
              <label htmlFor="client-ip-filter" className="block text-sm font-medium text-gray-700 mb-2">
                Client IP
              </label>
              <div className="relative">
                <input
                  id="client-ip-filter"
                  type="text"
                  placeholder="e.g., 192.168.0.163"
                  value={clientIPFilter}
                  onChange={(e) => setClientIPFilter(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') {
                      e.preventDefault();
                      handleFiltersSubmit();
                    }
                  }}
                  className="block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm placeholder-gray-400 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 text-sm"
                />
                {clientIPFilter && (
                  <button
                    type="button"
                    onClick={() => setClientIPFilter('')}
                    className="absolute inset-y-0 right-0 pr-3 flex items-center text-gray-400 hover:text-gray-600"
                  >
                    <X className="h-4 w-4" />
                  </button>
                )}
              </div>
            </div>
          </div>

          {/* Time Period Filter */}
          <div className="mb-4">
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Time Period
            </label>
            <div className="flex flex-wrap gap-2">
              {(['all', 'hour', 'day', 'week', 'month'] as TimeFilter[]).map((period) => (
                <button
                  key={period}
                  onClick={() => handleTimeFilterChange(period)}
                  className={`px-4 py-2 text-sm font-medium rounded-md ${
                    timeFilter === period
                      ? 'bg-indigo-600 text-white'
                      : 'bg-gray-100 text-gray-700 border border-gray-300 hover:bg-gray-200'
                  }`}
                >
                  {getTimeFilterLabel(period)}
                </button>
              ))}
            </div>
          </div>

          {/* Action Buttons */}
          <div className="flex items-center justify-end space-x-3">
            <button
              type="button"
              onClick={clearFilters}
              disabled={loading || !hasActiveFilters()}
              className="px-4 py-2 border border-gray-300 text-sm font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 disabled:opacity-50 inline-flex items-center"
            >
              <RotateCcw className="h-4 w-4 mr-2" />
              Reset
            </button>
            <button
              type="button"
              onClick={handleFiltersSubmit}
              disabled={loading}
              className="px-4 py-2 bg-indigo-600 text-white text-sm font-medium rounded-md hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 disabled:opacity-50 inline-flex items-center"
            >
              {loading ? (
                <>
                  <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white mr-2"></div>
                  Loading...
                </>
              ) : (
                <>
                  <Search className="h-4 w-4 mr-2" />
                  Apply Filters
                </>
              )}
            </button>
          </div>
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

        <div className="bg-white shadow rounded-lg">
          {loading ? (
            <div className="text-center py-12">
              <RefreshCw className="h-8 w-8 animate-spin mx-auto text-gray-400" />
              <p className="mt-2 text-sm text-gray-500">Loading domains...</p>
            </div>
          ) : domains.length === 0 ? (
            <div className="text-center py-12">
              <h3 className="mt-2 text-sm font-medium text-gray-900">No domains found</h3>
              <p className="mt-1 text-sm text-gray-500">
                {hasActiveFilters()
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
