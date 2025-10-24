import React from 'react';
import { Save, X, RefreshCw, Globe, Network } from 'lucide-react';
import type { AddMappingFormProps, DNSMapping } from '../../types';

const AddMappingForm: React.FC<AddMappingFormProps> = ({ 
  newMapping, 
  onMappingChange, 
  onSubmit, 
  onCancel, 
  loading 
}) => {
  const handleSubmit = (e: React.FormEvent<HTMLFormElement>): void => {
    e.preventDefault();
    onSubmit();
  };

  return (
    <div className="mb-6 bg-gray-50 border border-gray-200 rounded-md p-4">
      <h4 className="text-sm font-medium text-gray-900 mb-4">Add New DNS Mapping</h4>
      <form onSubmit={handleSubmit}>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div>
            <label htmlFor="new-domain" className="block text-sm font-medium text-gray-700 mb-1">
              Domain Name
            </label>
            <div className="relative rounded-md shadow-sm">
              <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                <Globe className="h-4 w-4 text-gray-400" />
              </div>
              <input
                type="text"
                id="new-domain"
                value={newMapping.domain}
                onChange={(e: React.ChangeEvent<HTMLInputElement>) => onMappingChange('domain', e.target.value)}
                placeholder="example.local"
                aria-describedby="domain-help"
                className="block w-full pl-10 py-3 border border-gray-300 rounded-md shadow-sm focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm"
              />
            </div>
            <p id="domain-help" className="mt-1 text-xs text-gray-500">
              Enter a domain name (e.g., example.local, server.internal)
            </p>
          </div>

          <div>
            <label htmlFor="new-ip" className="block text-sm font-medium text-gray-700 mb-1">
              IP Address
            </label>
            <div className="relative rounded-md shadow-sm">
              <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                <Network className="h-4 w-4 text-gray-400" />
              </div>
              <input
                type="text"
                id="new-ip"
                value={newMapping.ip}
                onChange={(e: React.ChangeEvent<HTMLInputElement>) => onMappingChange('ip', e.target.value)}
                placeholder="192.168.1.100"
                aria-describedby="ip-help"
                className="block w-full pl-10 py-3 border border-gray-300 rounded-md shadow-sm focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm font-mono"
              />
            </div>
            <p id="ip-help" className="mt-1 text-xs text-gray-500">
              Enter an IPv4 address (e.g., 192.168.1.100)
            </p>
          </div>
        </div>

        <div className="mt-4 flex justify-end space-x-2">
          <button
            type="button"
            onClick={onCancel}
            className="inline-flex items-center px-3 py-2 border border-gray-300 shadow-sm text-sm leading-4 font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
          >
            <X className="h-4 w-4 mr-1" />
            Cancel
          </button>
          <button
            type="submit"
            disabled={loading || !newMapping.domain.trim() || !newMapping.ip.trim()}
            className="inline-flex items-center px-3 py-2 border border-transparent text-sm leading-4 font-medium rounded-md shadow-sm text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {loading ? (
              <RefreshCw className="h-4 w-4 mr-1 animate-spin" />
            ) : (
              <Save className="h-4 w-4 mr-1" />
            )}
            Add Mapping
          </button>
        </div>
      </form>
    </div>
  );
};

export default AddMappingForm;
