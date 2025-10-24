import React from 'react';
import { Plus, RefreshCw, Globe } from 'lucide-react';
import type { DNSMappingsHeaderProps } from '../../types';

const DNSMappingsHeader: React.FC<DNSMappingsHeaderProps> = ({ 
  onRefresh, 
  onAddMapping, 
  loading, 
  mappingsCount 
}) => {
  return (
    <div className="flex items-center justify-between mb-6">
      <div className="flex items-center">
        <Globe className="h-6 w-6 text-indigo-600 mr-2" />
        <h3 className="text-lg font-medium text-gray-900">
          DNS Mappings
        </h3>
        {mappingsCount > 0 && (
          <span className="ml-2 text-sm text-gray-500">
            ({mappingsCount} {mappingsCount === 1 ? 'mapping' : 'mappings'})
          </span>
        )}
      </div>
      <div className="flex items-center space-x-2">
        <button
          onClick={onRefresh}
          disabled={loading}
          className="inline-flex items-center px-3 py-2 border border-gray-300 shadow-sm text-sm leading-4 font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 disabled:opacity-50"
        >
          <RefreshCw className={`h-4 w-4 mr-2 ${loading ? 'animate-spin' : ''}`} />
          Refresh
        </button>
        <button
          onClick={onAddMapping}
          className="inline-flex items-center px-3 py-2 border border-transparent text-sm leading-4 font-medium rounded-md shadow-sm text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
        >
          <Plus className="h-4 w-4 mr-2" />
          Add Mapping
        </button>
      </div>
    </div>
  );
};

export default DNSMappingsHeader;
