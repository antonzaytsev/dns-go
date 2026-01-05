import React, { useState, useEffect, useRef } from 'react';
import { useSearchParams } from 'react-router-dom';
import { AlertCircle, RefreshCw, Search, X, Filter, Globe, RotateCcw } from 'lucide-react';
import { useHealth } from '../hooks/useMetrics.ts';
import Requests from '../components/requests/Requests.tsx';
import Navigation from '../components/shared/Navigation.tsx';
import ConnectionStatus from '../components/shared/ConnectionStatus.tsx';
import { dnsApi } from '../services/api.ts';
import type { RequestsFullHeightProps, DnsRequest, SearchResponse } from '../types/index.ts';

const RequestsPage: React.FC = () => {
  const { isHealthy } = useHealth(30000);
  const [searchParams, setSearchParams] = useSearchParams();

  // Filter state
  const [domainFilter, setDomainFilter] = useState<string>('');
  const [clientIPFilter, setClientIPFilter] = useState<string>('');
  const [results, setResults] = useState<DnsRequest[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);
  const [currentPage, setCurrentPage] = useState<number>(0);
  const [totalResults, setTotalResults] = useState<number>(0);
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);
  const initializedRef = useRef<boolean>(false);

  const pageSize: number = 50;

  // Update URL when filters change (but only after initialization)
  const updateURL = (domain: string, client: string, page: number): void => {
    if (!initializedRef.current) return;

    const params = new URLSearchParams();
    if (domain) params.set('domain', domain);
    if (client) params.set('client', client);
    if (page > 0) params.set('page', page.toString());

    setSearchParams(params, { replace: true });
  };

  // Fetch results
  const fetchResults = async (
    domain: string = domainFilter,
    clientIP: string = clientIPFilter,
    page: number = 0,
    updateURLParams: boolean = true
  ): Promise<void> => {
    setLoading(true);
    setError(null);

    try {
      const result: SearchResponse = await dnsApi.searchLogs(domain, clientIP, pageSize, page * pageSize);
      setResults(result.results || []);
      setTotalResults(result.total || 0);
      setCurrentPage(page);
      setLastUpdated(new Date());

      // Update URL with current filters (only if there are filters or non-zero page)
      if (updateURLParams && (domain || clientIP || page > 0)) {
        updateURL(domain, clientIP, page);
      } else if (updateURLParams && !domain && !clientIP && page === 0) {
        // Clear URL params when resetting to default view
        setSearchParams({}, { replace: true });
      }
    } catch (err: any) {
      if (err.message && err.message.includes('503')) {
        setError('Search service unavailable. Please check if Elasticsearch is running.');
      } else if (err.message && err.message.includes('500')) {
        setError('Search failed due to database error. Please try again.');
      } else {
        setError('Failed to fetch logs. Please check your connection.');
      }
      setResults([]);
      setTotalResults(0);
    } finally {
      setLoading(false);
    }
  };

  // Load filters from URL on mount and fetch initial data
  useEffect(() => {
    if (initializedRef.current) return;

    const urlDomain = searchParams.get('domain') || '';
    const urlClient = searchParams.get('client') || '';
    const urlPage = parseInt(searchParams.get('page') || '0', 10);

    setDomainFilter(urlDomain);
    setClientIPFilter(urlClient);
    setCurrentPage(urlPage);
    initializedRef.current = true;

    // Always fetch data on mount (with or without filters)
    fetchResults(urlDomain, urlClient, urlPage, false);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const handleFiltersSubmit = (): void => {
    setCurrentPage(0);
    fetchResults(domainFilter, clientIPFilter, 0, true);
  };

  const clearFilters = (): void => {
    setDomainFilter('');
    setClientIPFilter('');
    setCurrentPage(0);
    // Fetch all results without filters
    fetchResults('', '', 0, true);
  };

  const hasActiveFilters = (): boolean => {
    return domainFilter !== '' || clientIPFilter !== '';
  };

  const handleRefresh = (): void => {
    fetchResults(domainFilter, clientIPFilter, currentPage, false);
  };

  const handlePageChange = async (_: string, page: number): Promise<void> => {
    await fetchResults(domainFilter, clientIPFilter, page, true);
  };

  const getSubtitle = (): string => {
    if (loading) return 'Loading...';
    return `${totalResults.toLocaleString()} ${totalResults === 1 ? 'request' : 'requests'}`;
  };

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
            <div className="flex items-center space-x-4">
              <ConnectionStatus
                isOnline={isHealthy}
                lastUpdated={lastUpdated}
                error={error}
              />
              <button
                onClick={handleRefresh}
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
          <h1 className="text-3xl font-bold text-gray-900 mb-2">Requests</h1>
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
                  Searching...
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

        <div className="h-full">
          <RequestsFullHeight
            requests={results}
            loading={loading}
            searchPerformed={true}
            currentPage={currentPage}
            totalResults={totalResults}
            pageSize={pageSize}
            onPageChange={handlePageChange}
            searchTerm={domainFilter}
          />
        </div>
      </main>
    </div>
  );
};

// Create a full-height version that wraps the existing Requests
const RequestsFullHeight: React.FC<RequestsFullHeightProps> = ({
  requests,
  loading,
  searchPerformed,
  currentPage,
  totalResults,
  pageSize,
  onPageChange,
  searchTerm
}) => {
  return (
    <div className="h-full">
      <Requests
        requests={requests}
        loading={loading}
        fullHeight={true}
        searchPerformed={searchPerformed}
        currentPage={currentPage}
        totalResults={totalResults}
        pageSize={pageSize}
        onPageChange={onPageChange}
        searchTerm={searchTerm}
      />
    </div>
  );
};

export default RequestsPage;
