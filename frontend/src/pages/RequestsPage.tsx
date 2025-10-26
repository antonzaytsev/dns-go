import React, { useState } from 'react';
import { AlertCircle, RefreshCw } from 'lucide-react';
import { useRequests } from '../hooks/useMetrics.ts';
import { useHealth } from '../hooks/useMetrics.ts';
import Requests from '../components/requests/Requests.tsx';
import Navigation from '../components/shared/Navigation.tsx';
import ConnectionStatus from '../components/shared/ConnectionStatus.tsx';
import PageHeader from '../components/shared/PageHeader.tsx';
import { dnsApi } from '../services/api.ts';
import type { RequestsFullHeightProps, DnsRequest, SearchResponse } from '../types/index.ts';

const RequestsPage: React.FC = () => {
  const { requests, loading, error, refresh } = useRequests(5000);
  const { isHealthy } = useHealth(30000);

  // Search state
  const [searchTerm, setSearchTerm] = useState<string>('');
  const [searchResults, setSearchResults] = useState<DnsRequest[]>([]);
  const [searchLoading, setSearchLoading] = useState<boolean>(false);
  const [searchError, setSearchError] = useState<string | null>(null);
  const [currentPage, setCurrentPage] = useState<number>(0);
  const [totalResults, setTotalResults] = useState<number>(0);
  const [searchPerformed, setSearchPerformed] = useState<boolean>(false);
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);

  const pageSize: number = 50;

  // Perform search
  const performSearch = async (term: string = searchTerm, page: number = 0): Promise<void> => {
    setSearchLoading(true);
    setSearchError(null);

    try {
      const result: SearchResponse = await dnsApi.searchLogs(term, pageSize, page * pageSize);
      setSearchResults(result.results || []);
      setTotalResults(result.total || 0);
      setCurrentPage(page);
      setSearchPerformed(true);
    } catch (error: any) {
      if (error.message && error.message.includes('503')) {
        setSearchError('Search service unavailable. Please check if Elasticsearch is running.');
      } else if (error.message && error.message.includes('500')) {
        setSearchError('Search failed due to database error. Please try again.');
      } else {
        setSearchError('Failed to search logs. Please check your connection.');
      }
      setSearchResults([]);
      setTotalResults(0);
    } finally {
      setSearchLoading(false);
    }
  };

  const handleSearchChange = (value: string): void => {
    setSearchTerm(value);
  };

  const handleSearchSubmit = (): void => {
    setCurrentPage(0);
    performSearch(searchTerm, 0);
  };

  const clearSearch = (): void => {
    setSearchTerm('');
    setSearchResults([]);
    setSearchPerformed(false);
    setCurrentPage(0);
    setTotalResults(0);
    setSearchError(null);
  };

  const handleRefresh = (): void => {
    if (searchPerformed) {
      performSearch(searchTerm, currentPage);
    } else {
      refresh();
      setLastUpdated(new Date());
    }
  };

  // Calculate display data
  const displayRequests = searchPerformed ? searchResults : (requests || []);
  const isLoading = searchLoading || (!searchPerformed && loading);

  const getSubtitle = (): string => {
    if (isLoading) return 'Loading...';
    if (searchPerformed) {
      return `${totalResults} ${totalResults === 1 ? 'result' : 'results'} found`;
    }
    return `${requests?.length || 0} requests`;
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
                error={error || searchError}
              />
              <button
                onClick={handleRefresh}
                disabled={isLoading}
                className="inline-flex items-center px-3 py-2 border border-gray-300 shadow-sm text-sm leading-4 font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 disabled:opacity-50"
              >
                <RefreshCw className={`h-4 w-4 mr-2 ${isLoading ? 'animate-spin' : ''}`} />
                Refresh
              </button>
            </div>
          </div>
        </div>
      </header>

      <main className="max-w-7xl mx-auto py-6 sm:px-6 lg:px-8">
        <PageHeader
          title="Requests"
          subtitle={getSubtitle()}
          searchValue={searchTerm}
          onSearchChange={handleSearchChange}
          onSearchSubmit={handleSearchSubmit}
          searchPlaceholder="Search domain, IP, client..."
          onClearSearch={clearSearch}
        />

        {(error || searchError) && (
          <div className="mb-6 bg-red-50 border border-red-200 rounded-md p-4">
            <div className="flex">
              <AlertCircle className="h-5 w-5 text-red-400" />
              <div className="ml-3">
                <h3 className="text-sm font-medium text-red-800">Error</h3>
                <div className="mt-2 text-sm text-red-700">{error || searchError}</div>
              </div>
            </div>
          </div>
        )}

        <div className="h-full">
          <RequestsFullHeight
            requests={displayRequests}
            loading={isLoading}
            searchPerformed={searchPerformed}
            currentPage={currentPage}
            totalResults={totalResults}
            pageSize={pageSize}
            onPageChange={performSearch}
            searchTerm={searchTerm}
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
