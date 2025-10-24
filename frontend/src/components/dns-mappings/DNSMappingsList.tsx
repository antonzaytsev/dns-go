import React from 'react';
import DNSMappingRow from './DNSMappingRow.tsx';
import type { DNSMappingsListProps } from '../../types';

const DNSMappingsList: React.FC<DNSMappingsListProps> = ({ 
  mappings, 
  editingDomain, 
  onEdit, 
  onSave, 
  onCancelEdit, 
  onDelete, 
  loading 
}) => {
  const mappingEntries: [string, string][] = Object.entries(mappings);

  if (mappingEntries.length === 0) {
    return (
      <div className="text-center py-12">
        <div className="text-gray-500">
          <p className="text-lg font-medium">No DNS mappings configured</p>
          <p className="mt-2">Add your first DNS mapping to get started.</p>
        </div>
      </div>
    );
  }

  return (
    <div className="bg-white shadow overflow-hidden sm:rounded-md">
      <table className="min-w-full divide-y divide-gray-200">
        <thead className="bg-gray-50">
          <tr>
            <th 
              scope="col" 
              className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
            >
              Domain
            </th>
            <th 
              scope="col" 
              className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
            >
              IP Address
            </th>
            <th 
              scope="col" 
              className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider"
            >
              Actions
            </th>
          </tr>
        </thead>
        <tbody className="bg-white divide-y divide-gray-200">
          {mappingEntries.map(([domain, ip]) => (
            <DNSMappingRow
              key={domain}
              domain={domain}
              ip={ip}
              isEditing={editingDomain === domain}
              onEdit={() => onEdit(domain)}
              onSave={onSave}
              onCancel={() => onCancelEdit()}
              onDelete={onDelete}
              loading={loading}
            />
          ))}
        </tbody>
      </table>
    </div>
  );
};

export default DNSMappingsList;
