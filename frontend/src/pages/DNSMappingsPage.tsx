import React from 'react';
import DNSMappings from '../components/dns-mappings/DNSMappings.tsx';
import Navigation from '../components/shared/Navigation.tsx';

const DNSMappingsPage: React.FC = () => {
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
      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="space-y-8">
          {/* DNS Mappings Management */}
          <section>
            <DNSMappings />
          </section>
        </div>
      </main>
    </div>
  );
};

export default DNSMappingsPage;
