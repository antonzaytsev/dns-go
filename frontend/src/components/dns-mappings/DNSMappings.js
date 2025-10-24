import React, { useState, useEffect } from 'react';
import { dnsApi } from '../../services/api.ts';
import DNSMappingsHeader from './DNSMappingsHeader';
import StatusMessages from '../StatusMessages.tsx';
import AddMappingForm from './AddMappingForm';
import DNSMappingsList from './DNSMappingsList';
import DeleteConfirmationModal from './DeleteConfirmationModal';

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

  const mappingEntries = Object.entries(mappings);

  const handleFormCancel = () => {
    setShowAddForm(false);
    setNewMapping({ domain: '', ip: '' });
    setError(null);
  };

  return (
    <div className="bg-white shadow rounded-lg">
      <div className="px-4 py-5 sm:p-6">
        <DNSMappingsHeader
          onRefresh={loadMappings}
          onAddMapping={() => setShowAddForm(true)}
          loading={loading}
          mappingsCount={mappingEntries.length}
        />

        <StatusMessages error={error} success={success} />

        {showAddForm && (
          <AddMappingForm
            newMapping={newMapping}
            onMappingChange={setNewMapping}
            onSubmit={addMapping}
            onCancel={handleFormCancel}
            loading={loading}
          />
        )}

        <DNSMappingsList
          mappings={mappings}
          editingDomain={editingDomain}
          onEdit={setEditingDomain}
          onSave={updateMapping}
          onCancelEdit={() => setEditingDomain(null)}
          onDelete={showDeleteConfirmation}
          loading={loading}
        />

        {mappingEntries.length > 0 && (
          <div className="mt-4 text-sm text-gray-500">
            Total mappings: {mappingEntries.length}
          </div>
        )}
      </div>

      <DeleteConfirmationModal
        isOpen={deleteConfirmation.show}
        domain={deleteConfirmation.domain}
        onConfirm={handleDeleteConfirm}
        onCancel={handleDeleteCancel}
        loading={loading}
      />
    </div>
  );
};

export default DNSMappings;
