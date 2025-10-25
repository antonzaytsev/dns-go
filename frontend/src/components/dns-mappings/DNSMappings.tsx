import React, { useState, useEffect } from 'react';
import { dnsApi } from '../../services/api.ts';
import DNSMappingsHeader from './DNSMappingsHeader.tsx';
import StatusMessages from '../StatusMessages.tsx';
import AddMappingForm from './AddMappingForm.tsx';
import DNSMappingsList from './DNSMappingsList.tsx';
import DeleteConfirmationModal from './DeleteConfirmationModal.tsx';
import type { DNSMappingsState, DNSMapping, ModalState, DNSMappingsResponse } from '../../types';

const DNSMappings: React.FC = () => {
  const [mappings, setMappings] = useState<DNSMappingsState>({});
  const [loading, setLoading] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [editingDomain, setEditingDomain] = useState<string | null>(null);
  const [newMapping, setNewMapping] = useState<DNSMapping>({ domain: '', ip: '' });
  const [showAddForm, setShowAddForm] = useState<boolean>(false);
  const [deleteConfirmation, setDeleteConfirmation] = useState<ModalState>({ show: false, domain: '' });


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
    } catch (err: any) {
      setError(`Failed to load DNS mappings: ${err.message}`);
    } finally {
      setLoading(false);
    }
  };

  const addMapping = async (): Promise<void> => {
    setLoading(true);
    setError(null);

    try {
      await dnsApi.addDNSMapping(newMapping.domain.trim(), newMapping.ip.trim());
      setSuccess('DNS mapping added successfully');
      setNewMapping({ domain: '', ip: '' });
      setShowAddForm(false);
      await loadMappings();
    } catch (err: any) {
      setError(`Failed to add DNS mapping: ${err.message}`);
    } finally {
      setLoading(false);
    }
  };

  const showDeleteConfirmation = (domain: string): void => {
    setDeleteConfirmation({ show: true, domain });
  };

  const handleDeleteConfirm = async (): Promise<void> => {
    const domain = deleteConfirmation.domain;
    setDeleteConfirmation({ show: false, domain: '' });

    setLoading(true);
    setError(null);

    try {
      await dnsApi.deleteDNSMapping(domain);
      setSuccess('DNS mapping deleted successfully');
      await loadMappings();
    } catch (err: any) {
      setError(`Failed to delete DNS mapping: ${err.message}`);
    } finally {
      setLoading(false);
    }
  };


  const handleDeleteCancel = (): void => {
    setDeleteConfirmation({ show: false, domain: '' });
  };


  const updateMapping = async (oldDomain: string, newDomain: string, newIp: string): Promise<void> => {
    setLoading(true);
    setError(null);

    try {
      await dnsApi.deleteDNSMapping(oldDomain);

      await dnsApi.addDNSMapping(newDomain.trim(), newIp.trim());

      setSuccess('DNS mapping updated successfully');
      setEditingDomain(null);
      await loadMappings();
    } catch (err: any) {
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
      const timer: NodeJS.Timeout = setTimeout(() => {
        setError(null);
        setSuccess(null);
      }, 5000);
      return () => clearTimeout(timer);
    }
  }, [error, success]);

  const mappingEntries: [string, string][] = Object.entries(mappings);

  const handleFormCancel = (): void => {
    setShowAddForm(false);
    setNewMapping({ domain: '', ip: '' });
    setError(null);
  };

  const handleMappingChange = (field: keyof DNSMapping, value: string): void => {
    setNewMapping(prev => ({
      ...prev,
      [field]: value
    }));
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
            onMappingChange={handleMappingChange}
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
