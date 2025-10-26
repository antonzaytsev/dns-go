import React, { useState, useEffect } from 'react';
import { dnsApi } from '../../services/api.ts';
import StatusMessages from './StatusMessages.tsx';
import AddMappingForm from './AddMappingForm.tsx';
import DNSMappingsList from './DNSMappingsList.tsx';
import DeleteConfirmationModal from './DeleteConfirmationModal.tsx';
import type { DNSMappingsState, DNSMapping, ModalState, DNSMappingsResponse, DNSMappingsProps } from '../../types';

const DNSMappings: React.FC<DNSMappingsProps> = ({ 
  mappings, 
  loading: externalLoading = false, 
  error: externalError = null,
  onRefresh,
  onMappingsChange,
  showAddForm,
  onShowAddFormChange
}) => {
  const [operationLoading, setOperationLoading] = useState<boolean>(false);
  const [success, setSuccess] = useState<string | null>(null);
  const [editingDomain, setEditingDomain] = useState<string | null>(null);
  const [newMapping, setNewMapping] = useState<DNSMapping>({ domain: '', ip: '' });
  const [deleteConfirmation, setDeleteConfirmation] = useState<ModalState>({ show: false, domain: '' });

  // Use external error or local error for operations
  const error = externalError;
  const isLoading = externalLoading || operationLoading;

  const addMapping = async (): Promise<void> => {
    setOperationLoading(true);

    try {
      await dnsApi.addDNSMapping(newMapping.domain.trim(), newMapping.ip.trim());
      setSuccess('DNS mapping added successfully');
      setNewMapping({ domain: '', ip: '' });
      onShowAddFormChange(false);
      onRefresh(); // Refresh the mappings after successful addition
    } catch (err: any) {
      console.error('Failed to add DNS mapping:', err.message);
    } finally {
      setOperationLoading(false);
    }
  };

  const showDeleteConfirmation = (domain: string): void => {
    setDeleteConfirmation({ show: true, domain });
  };

  const handleDeleteConfirm = async (): Promise<void> => {
    const domain = deleteConfirmation.domain;
    setDeleteConfirmation({ show: false, domain: '' });

    setOperationLoading(true);

    try {
      await dnsApi.deleteDNSMapping(domain);
      setSuccess('DNS mapping deleted successfully');
      onRefresh(); // Refresh the mappings after successful deletion
    } catch (err: any) {
      console.error('Failed to delete DNS mapping:', err.message);
    } finally {
      setOperationLoading(false);
    }
  };


  const handleDeleteCancel = (): void => {
    setDeleteConfirmation({ show: false, domain: '' });
  };


  const updateMapping = async (oldDomain: string, newDomain: string, newIp: string): Promise<void> => {
    setOperationLoading(true);

    try {
      await dnsApi.deleteDNSMapping(oldDomain);
      await dnsApi.addDNSMapping(newDomain.trim(), newIp.trim());
      setSuccess('DNS mapping updated successfully');
      setEditingDomain(null);
      onRefresh(); // Refresh the mappings after successful update
    } catch (err: any) {
      console.error('Failed to update DNS mapping:', err.message);
    } finally {
      setOperationLoading(false);
    }
  };


  // Auto-clear messages after 5 seconds
  useEffect(() => {
    if (success) {
      const timer: NodeJS.Timeout = setTimeout(() => {
        setSuccess(null);
      }, 5000);
      return () => clearTimeout(timer);
    }
  }, [success]);

  const mappingEntries: [string, string][] = Object.entries(mappings);

  const handleFormCancel = (): void => {
    onShowAddFormChange(false);
    setNewMapping({ domain: '', ip: '' });
  };

  const handleMappingChange = (field: keyof DNSMapping, value: string): void => {
    setNewMapping(prev => ({
      ...prev,
      [field]: value
    }));
  };

  return (
    <>
      <StatusMessages error={error} success={success} />

      {showAddForm && (
        <div className="mb-6">
          <AddMappingForm
            newMapping={newMapping}
            onMappingChange={handleMappingChange}
            onSubmit={addMapping}
            onCancel={handleFormCancel}
            loading={isLoading}
          />
        </div>
      )}

      <DNSMappingsList
        mappings={mappings}
        editingDomain={editingDomain}
        onEdit={setEditingDomain}
        onSave={updateMapping}
        onCancelEdit={() => setEditingDomain(null)}
        onDelete={showDeleteConfirmation}
        loading={isLoading}
      />

      <DeleteConfirmationModal
        isOpen={deleteConfirmation.show}
        domain={deleteConfirmation.domain}
        onConfirm={handleDeleteConfirm}
        onCancel={handleDeleteCancel}
        loading={isLoading}
      />
    </>
  );
};

export default DNSMappings;
