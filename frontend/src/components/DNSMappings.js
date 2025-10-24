import React, { useState, useEffect } from 'react';
import { 
  Plus, 
  Trash2, 
  Edit3, 
  Save, 
  X, 
  AlertCircle,
  CheckCircle,
  RefreshCw,
  Globe
} from 'lucide-react';

const DNSMappings = () => {
  const [mappings, setMappings] = useState({});
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const [success, setSuccess] = useState(null);
  const [editingDomain, setEditingDomain] = useState(null);
  const [newMapping, setNewMapping] = useState({ domain: '', ip: '' });
  const [showAddForm, setShowAddForm] = useState(false);

  // Load DNS mappings from API
  const loadMappings = async () => {
    setLoading(true);
    setError(null);
    
    try {
      const response = await fetch('/api/dns-mappings');
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }
      
      const data = await response.json();
      // Remove trailing dots for display
      const displayMappings = {};
      Object.entries(data.mappings || {}).forEach(([domain, ip]) => {
        const displayDomain = domain.endsWith('.') ? domain.slice(0, -1) : domain;
        displayMappings[displayDomain] = ip;
      });
      setMappings(displayMappings);
    } catch (err) {
      setError(`Failed to load DNS mappings: ${err.message}`);
    } finally {
      setLoading(false);
    }
  };

  // Add a new DNS mapping
  const addMapping = async () => {
    if (!newMapping.domain.trim() || !newMapping.ip.trim()) {
      setError('Both domain and IP address are required');
      return;
    }

    setLoading(true);
    setError(null);
    
    try {
      const response = await fetch('/api/dns-mappings', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          domain: newMapping.domain.trim(),
          ip: newMapping.ip.trim(),
        }),
      });

      if (!response.ok) {
        const errorData = await response.text();
        throw new Error(errorData || `HTTP ${response.status}`);
      }

      setSuccess('DNS mapping added successfully');
      setNewMapping({ domain: '', ip: '' });
      setShowAddForm(false);
      await loadMappings();
    } catch (err) {
      setError(`Failed to add DNS mapping: ${err.message}`);
    } finally {
      setLoading(false);
    }
  };

  // Delete a DNS mapping
  const deleteMapping = async (domain) => {
    if (!window.confirm(`Are you sure you want to delete the mapping for "${domain}"?`)) {
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const response = await fetch(`/api/dns-mappings?domain=${encodeURIComponent(domain)}`, {
        method: 'DELETE',
      });

      if (!response.ok) {
        const errorData = await response.text();
        throw new Error(errorData || `HTTP ${response.status}`);
      }

      setSuccess('DNS mapping deleted successfully');
      await loadMappings();
    } catch (err) {
      setError(`Failed to delete DNS mapping: ${err.message}`);
    } finally {
      setLoading(false);
    }
  };

  // Update a DNS mapping
  const updateMapping = async (oldDomain, newDomain, newIp) => {
    if (!newDomain.trim() || !newIp.trim()) {
      setError('Both domain and IP address are required');
      return;
    }

    setLoading(true);
    setError(null);

    try {
      // If domain changed, delete old and add new; otherwise just update IP
      if (oldDomain !== newDomain.trim()) {
        // Delete old mapping
        await fetch(`/api/dns-mappings?domain=${encodeURIComponent(oldDomain)}`, {
          method: 'DELETE',
        });
      }

      // Add/update new mapping
      const response = await fetch('/api/dns-mappings', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          domain: newDomain.trim(),
          ip: newIp.trim(),
        }),
      });

      if (!response.ok) {
        const errorData = await response.text();
        throw new Error(errorData || `HTTP ${response.status}`);
      }

      setSuccess('DNS mapping updated successfully');
      setEditingDomain(null);
      await loadMappings();
    } catch (err) {
      setError(`Failed to update DNS mapping: ${err.message}`);
    } finally {
      setLoading(false);
    }
  };

  // Load mappings on component mount
  useEffect(() => {
    loadMappings();
  }, []);

  // Auto-clear messages after 5 seconds
  useEffect(() => {
    if (error || success) {
      const timer = setTimeout(() => {
        setError(null);
        setSuccess(null);
      }, 5000);
      return () => clearTimeout(timer);
    }
  }, [error, success]);

  const mappingEntries = Object.entries(mappings);

  return (
    <div className="bg-white shadow rounded-lg">
      <div className="px-4 py-5 sm:p-6">
        <div className="flex items-center justify-between mb-6">
          <div className="flex items-center">
            <Globe className="h-6 w-6 text-indigo-600 mr-2" />
            <h3 className="text-lg font-medium text-gray-900">
              DNS Mappings
            </h3>
          </div>
          <div className="flex items-center space-x-2">
            <button
              onClick={loadMappings}
              disabled={loading}
              className="inline-flex items-center px-3 py-2 border border-gray-300 shadow-sm text-sm leading-4 font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 disabled:opacity-50"
            >
              <RefreshCw className={`h-4 w-4 mr-2 ${loading ? 'animate-spin' : ''}`} />
              Refresh
            </button>
            <button
              onClick={() => setShowAddForm(true)}
              className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md shadow-sm text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
            >
              <Plus className="h-4 w-4 mr-2" />
              Add Mapping
            </button>
          </div>
        </div>

        {/* Status Messages */}
        {error && (
          <div className="mb-4 bg-red-50 border border-red-200 rounded-md p-4">
            <div className="flex">
              <AlertCircle className="h-5 w-5 text-red-400" />
              <div className="ml-3">
                <p className="text-sm text-red-700">{error}</p>
              </div>
            </div>
          </div>
        )}

        {success && (
          <div className="mb-4 bg-green-50 border border-green-200 rounded-md p-4">
            <div className="flex">
              <CheckCircle className="h-5 w-5 text-green-400" />
              <div className="ml-3">
                <p className="text-sm text-green-700">{success}</p>
              </div>
            </div>
          </div>
        )}

        {/* Add New Mapping Form */}
        {showAddForm && (
          <div className="mb-6 bg-gray-50 border border-gray-200 rounded-md p-4">
            <h4 className="text-sm font-medium text-gray-900 mb-3">Add New DNS Mapping</h4>
            <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
              <div>
                <label htmlFor="new-domain" className="block text-sm font-medium text-gray-700">
                  Domain
                </label>
                <input
                  type="text"
                  id="new-domain"
                  value={newMapping.domain}
                  onChange={(e) => setNewMapping({ ...newMapping, domain: e.target.value })}
                  placeholder="example.local"
                  className="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm"
                />
              </div>
              <div>
                <label htmlFor="new-ip" className="block text-sm font-medium text-gray-700">
                  IP Address
                </label>
                <input
                  type="text"
                  id="new-ip"
                  value={newMapping.ip}
                  onChange={(e) => setNewMapping({ ...newMapping, ip: e.target.value })}
                  placeholder="192.168.1.100"
                  className="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm"
                />
              </div>
            </div>
            <div className="mt-3 flex justify-end space-x-2">
              <button
                onClick={() => {
                  setShowAddForm(false);
                  setNewMapping({ domain: '', ip: '' });
                  setError(null);
                }}
                className="inline-flex items-center px-3 py-2 border border-gray-300 shadow-sm text-sm leading-4 font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50"
              >
                <X className="h-4 w-4 mr-1" />
                Cancel
              </button>
              <button
                onClick={addMapping}
                disabled={loading}
                className="inline-flex items-center px-3 py-2 border border-transparent text-sm leading-4 font-medium rounded-md shadow-sm text-white bg-indigo-600 hover:bg-indigo-700 disabled:opacity-50"
              >
                <Save className="h-4 w-4 mr-1" />
                Add
              </button>
            </div>
          </div>
        )}

        {/* DNS Mappings Table */}
        <div className="overflow-hidden shadow ring-1 ring-black ring-opacity-5 md:rounded-lg">
          <table className="min-w-full divide-y divide-gray-300">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Domain
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  IP Address
                </th>
                <th className="relative px-6 py-3">
                  <span className="sr-only">Actions</span>
                </th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {mappingEntries.length === 0 ? (
                <tr>
                  <td colSpan="3" className="px-6 py-12 text-center text-sm text-gray-500">
                    {loading ? (
                      <div className="flex items-center justify-center">
                        <RefreshCw className="h-5 w-5 animate-spin mr-2" />
                        Loading DNS mappings...
                      </div>
                    ) : (
                      <div>
                        <Globe className="mx-auto h-12 w-12 text-gray-400" />
                        <p className="mt-2">No DNS mappings configured</p>
                        <p className="text-gray-400">Click "Add Mapping" to get started</p>
                      </div>
                    )}
                  </td>
                </tr>
              ) : (
                mappingEntries.map(([domain, ip]) => (
                  <MappingRow
                    key={domain}
                    domain={domain}
                    ip={ip}
                    isEditing={editingDomain === domain}
                    onEdit={() => setEditingDomain(domain)}
                    onSave={updateMapping}
                    onCancel={() => setEditingDomain(null)}
                    onDelete={deleteMapping}
                    loading={loading}
                  />
                ))
              )}
            </tbody>
          </table>
        </div>

        {mappingEntries.length > 0 && (
          <div className="mt-4 text-sm text-gray-500">
            Total mappings: {mappingEntries.length}
          </div>
        )}
      </div>
    </div>
  );
};

// Individual mapping row component
const MappingRow = ({ domain, ip, isEditing, onEdit, onSave, onCancel, onDelete, loading }) => {
  const [editDomain, setEditDomain] = useState(domain);
  const [editIp, setEditIp] = useState(ip);

  useEffect(() => {
    if (isEditing) {
      setEditDomain(domain);
      setEditIp(ip);
    }
  }, [isEditing, domain, ip]);

  const handleSave = () => {
    onSave(domain, editDomain, editIp);
  };

  const handleCancel = () => {
    setEditDomain(domain);
    setEditIp(ip);
    onCancel();
  };

  if (isEditing) {
    return (
      <tr className="bg-blue-50">
        <td className="px-6 py-4 whitespace-nowrap">
          <input
            type="text"
            value={editDomain}
            onChange={(e) => setEditDomain(e.target.value)}
            className="block w-full border-gray-300 rounded-md shadow-sm focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm"
          />
        </td>
        <td className="px-6 py-4 whitespace-nowrap">
          <input
            type="text"
            value={editIp}
            onChange={(e) => setEditIp(e.target.value)}
            className="block w-full border-gray-300 rounded-md shadow-sm focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm"
          />
        </td>
        <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
          <div className="flex items-center justify-end space-x-2">
            <button
              onClick={handleSave}
              disabled={loading}
              className="text-green-600 hover:text-green-900 disabled:opacity-50"
            >
              <Save className="h-4 w-4" />
            </button>
            <button
              onClick={handleCancel}
              disabled={loading}
              className="text-gray-600 hover:text-gray-900 disabled:opacity-50"
            >
              <X className="h-4 w-4" />
            </button>
          </div>
        </td>
      </tr>
    );
  }

  return (
    <tr className="hover:bg-gray-50">
      <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
        {domain}
      </td>
      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
        <span className="font-mono">{ip}</span>
      </td>
      <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
        <div className="flex items-center justify-end space-x-2">
          <button
            onClick={onEdit}
            disabled={loading}
            className="text-indigo-600 hover:text-indigo-900 disabled:opacity-50"
          >
            <Edit3 className="h-4 w-4" />
          </button>
          <button
            onClick={() => onDelete(domain)}
            disabled={loading}
            className="text-red-600 hover:text-red-900 disabled:opacity-50"
          >
            <Trash2 className="h-4 w-4" />
          </button>
        </div>
      </td>
    </tr>
  );
};

export default DNSMappings;
