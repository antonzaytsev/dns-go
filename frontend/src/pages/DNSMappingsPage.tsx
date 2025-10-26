import React, { useState, useEffect } from 'react';
import { RefreshCw, Plus } from 'lucide-react';
import { useHealth } from '../hooks/useMetrics.ts';
import DNSMappings from '../components/dns-mappings/DNSMappings.tsx';
import Navigation from '../components/shared/Navigation.tsx';
import ConnectionStatus from '../components/shared/ConnectionStatus.tsx';
import PageHeader from '../components/shared/PageHeader.tsx';
import { dnsApi } from '../services/api.ts';
import type { DNSMappingsState, DNSMappingsResponse } from '../types/index.ts';

const DNSMappingsPage: React.FC = () => {
  const [mappings, setMappings] = useState<DNSMappingsState>({});
  const [loading, setLoading] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);
  const [searchTerm, setSearchTerm] = useState<string>('');
  const [showAddForm, setShowAddForm] = useState<boolean>(false);
  const { isHealthy } = useHealth(30000);

  // Load DNS mappings from API
  const loadMappings = async (): Promise<void> => {
    setLoading(true);
    setError(null);

    try {
      const data: DNSMappingsResponse = await dnsApi.getDNSMappings();
      // Remove trailing dots for display
      const displayMappings: DNSMappingsState = {};
      Object.entries(data.mappings || {}).forEach(([domain, ip]: [string, string]) => {
        const displayDomain = domain.endsWith('.') ? domain.slice(0, -1) : domain;
        displayMappings[displayDomain] = ip;
      });
      setMappings(displayMappings);
      setLastUpdated(new Date());
    } catch (err: any) {
      setError(`Failed to load DNS mappings: ${err.message}`);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadMappings();
  }, []);

  const handleRefresh = (): void => {
    loadMappings();
  };

  const handleSearchChange = (value: string): void => {
    setSearchTerm(value);
  };

  const clearSearch = (): void => {
    setSearchTerm('');
  };

  // Filter mappings based on search term
  const filteredMappings = React.useMemo(() => {
    if (!searchTerm) return mappings;

    const filtered: DNSMappingsState = {};
    Object.entries(mappings).forEach(([domain, ip]) => {
      if (domain.toLowerCase().includes(searchTerm.toLowerCase()) ||
          ip.toLowerCase().includes(searchTerm.toLowerCase())) {
        filtered[domain] = ip;
      }
    });
    return filtered;
  }, [mappings, searchTerm]);

  const mappingsCount = Object.keys(filteredMappings).length;
  const totalMappingsCount = Object.keys(mappings).length;

  const getSubtitle = (): string => {
    if (loading) return 'Loading...';
    if (searchTerm) {
      return `${mappingsCount} of ${totalMappingsCount} mappings found`;
    }
    return `${totalMappingsCount} ${totalMappingsCount === 1 ? 'mapping' : 'mappings'}`;
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
        <PageHeader
          title="DNS Mappings"
          subtitle={getSubtitle()}
          searchValue={searchTerm}
          onSearchChange={handleSearchChange}
          searchPlaceholder="Search domain or IP address..."
          onClearSearch={clearSearch}
          actionButton={
            <button
              onClick={() => setShowAddForm(true)}
              className="inline-flex items-center px-3 py-2 border border-transparent text-sm leading-4 font-medium rounded-md shadow-sm text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
            >
              <Plus className="h-4 w-4 mr-2" />
              Add Mapping
            </button>
          }
        />

        <section>
          <DNSMappings
            mappings={filteredMappings}
            loading={loading}
            error={error}
            onRefresh={loadMappings}
            onMappingsChange={setMappings}
            showAddForm={showAddForm}
            onShowAddFormChange={setShowAddForm}
          />
        </section>
      </main>
    </div>
  );
};

export default DNSMappingsPage;
