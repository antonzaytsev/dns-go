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
  Globe,
  Network
} from 'lucide-react';
import { dnsApi } from '../services/api';

const DNSMappings = () => {
  const [mappings, setMappings] = useState({});
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const [success, setSuccess] = useState(null);
  const [editingDomain, setEditingDomain] = useState(null);
  const [newMapping, setNewMapping] = useState({ domain: '', ip: '' });
  const [showAddForm, setShowAddForm] = useState(false);
  const [deleteConfirmation, setDeleteConfirmation] = useState({ show: false, domain: '' });


  // Load DNS mappings from API
  const loadMappings = async () => {
    setLoading(true);
    setError(null);

    try {
      const data = await dnsApi.getDNSMappings();
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

  const addMapping = async () => {
    setLoading(true);
    setError(null);

    try {
      await dnsApi.addDNSMapping(newMapping.domain.trim(), newMapping.ip.trim());
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

  const showDeleteConfirmation = (domain) => {
    setDeleteConfirmation({ show: true, domain });
  };

  const handleDeleteConfirm = async () => {
    const domain = deleteConfirmation.domain;
    setDeleteConfirmation({ show: false, domain: '' });

    setLoading(true);
    setError(null);

    try {
      await dnsApi.deleteDNSMapping(domain);
      setSuccess('DNS mapping deleted successfully');
      await loadMappings();
    } catch (err) {
      setError(`Failed to delete DNS mapping: ${err.message}`);
    } finally {
      setLoading(false);
    }
  };


  const handleDeleteCancel = () => {
    setDeleteConfirmation({ show: false, domain: '' });
  };

  const handleBackdropClick = (e) => {
    if (e.target === e.currentTarget) {
      handleDeleteCancel();
    }
  };

  const updateMapping = async (oldDomain, newDomain, newIp) => {
    setLoading(true);
    setError(null);

    try {
      await dnsApi.deleteDNSMapping(oldDomain);

      await dnsApi.addDNSMapping(newDomain.trim(), newIp.trim());

      setSuccess('DNS mapping updated successfully');
      setEditingDomain(null);
      await loadMappings();
    } catch (err) {
      setError(`Failed to update DNS mapping: ${err.message}`);
    } finally {
      setLoading(false);
    }
  };


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

  useEffect(() => {
    if (deleteConfirmation.show) {
      const modalElement = document.querySelector('[data-modal="delete-confirmation"]');
      if (modalElement) {
        modalElement.focus();
      }

      const handleKeyDown = (e) => {
        if (e.key === 'Escape') {
          handleDeleteCancel();
        }
      };

      document.addEventListener('keydown', handleKeyDown);
      return () => document.removeEventListener('keydown', handleKeyDown);
    }
  }, [deleteConfirmation.show]);

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
              className="inline-flex items-center px-3 py-2 border border-transparent text-sm leading-4 font-medium rounded-md shadow-sm text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
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
            <h4 className="text-sm font-medium text-gray-900 mb-4">Add New DNS Mapping</h4>
            <form onSubmit={(e) => { e.preventDefault(); addMapping(); }}>
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
                      onChange={(e) => setNewMapping({ ...newMapping, domain: e.target.value })}
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
                      onChange={(e) => setNewMapping({ ...newMapping, ip: e.target.value })}
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
                  onClick={() => {
                    setShowAddForm(false);
                    setNewMapping({ domain: '', ip: '' });
                    setError(null);
                  }}
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
                    onDelete={showDeleteConfirmation}
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

      {/* Delete Confirmation Modal */}
      {deleteConfirmation.show && (
        <div
          className="fixed inset-0 bg-gray-600 bg-opacity-50 overflow-y-auto h-full w-full z-50"
          onClick={handleBackdropClick}
          tabIndex={-1}
          data-modal="delete-confirmation"
        >
          <div className="relative top-20 mx-auto p-5 border w-96 shadow-lg rounded-md bg-white"
               onClick={(e) => e.stopPropagation()}>
            <div className="mt-3">
              <div className="mx-auto flex items-center justify-center h-12 w-12 rounded-full bg-red-100">
                <AlertCircle className="h-6 w-6 text-red-600" />
              </div>
              <div className="mt-3 text-center">
                <h3 className="text-lg leading-6 font-medium text-gray-900">
                  Delete DNS Mapping
                </h3>
                <div className="mt-2 px-7 py-3">
                  <p className="text-sm text-gray-500">
                    Are you sure you want to delete the mapping for{' '}
                    <span className="font-semibold text-gray-900">
                      {deleteConfirmation.domain}
                    </span>
                    ? This action cannot be undone.
                  </p>
                </div>
                <div className="flex items-center justify-center space-x-4 px-4 py-3">
                  <button
                    type="button"
                    onClick={handleDeleteCancel}
                    disabled={loading}
                    className="inline-flex items-center px-4 py-2 border border-gray-300 shadow-sm text-sm font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 disabled:opacity-50"
                  >
                    Cancel
                  </button>
                  <button
                    type="button"
                    onClick={handleDeleteConfirm}
                    disabled={loading}
                    className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md shadow-sm text-white bg-red-600 hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-red-500 disabled:opacity-50"
                  >
                    {loading ? (
                      <>
                        <RefreshCw className="h-4 w-4 mr-2 animate-spin" />
                        Deleting...
                      </>
                    ) : (
                      <>
                        <Trash2 className="h-4 w-4 mr-2" />
                        Delete
                      </>
                    )}
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      )}
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

  const handleKeyDown = (e) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSave();
    } else if (e.key === 'Escape') {
      handleCancel();
    }
  };

  if (isEditing) {
    return (
      <tr className="bg-blue-50">
        <td className="px-6 py-4 whitespace-nowrap">
          <div className="relative">
            <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
              <Globe className="h-4 w-4 text-gray-400" />
            </div>
            <input
              type="text"
              value={editDomain}
              onChange={(e) => setEditDomain(e.target.value)}
              onKeyDown={handleKeyDown}
              className="block w-full pl-10 py-2 border border-gray-300 rounded-md shadow-sm focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm"
              autoFocus
            />
          </div>
        </td>
        <td className="px-6 py-4 whitespace-nowrap">
          <div className="relative">
            <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
              <Network className="h-4 w-4 text-gray-400" />
            </div>
            <input
              type="text"
              value={editIp}
              onChange={(e) => setEditIp(e.target.value)}
              onKeyDown={handleKeyDown}
              className="block w-full pl-10 py-2 border border-gray-300 rounded-md shadow-sm focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm font-mono"
            />
          </div>
        </td>
        <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
          <div className="flex items-center justify-end space-x-2">
            <button
              onClick={handleSave}
              disabled={loading || !editDomain.trim() || !editIp.trim()}
              className="text-green-600 hover:text-green-900 disabled:opacity-50 disabled:cursor-not-allowed p-1 rounded focus:outline-none focus:ring-2 focus:ring-green-500"
              title="Save changes (Enter)"
            >
              <Save className="h-4 w-4" />
            </button>
            <button
              onClick={handleCancel}
              disabled={loading}
              className="text-gray-600 hover:text-gray-900 disabled:opacity-50 p-1 rounded focus:outline-none focus:ring-2 focus:ring-gray-500"
              title="Cancel editing (Escape)"
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
