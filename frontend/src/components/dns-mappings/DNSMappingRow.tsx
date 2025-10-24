import React, { useState, useEffect } from 'react';
import { Edit3, Save, X, Trash2, Globe, Network } from 'lucide-react';
import type { DNSMappingRowProps } from '../../types';

const DNSMappingRow: React.FC<DNSMappingRowProps> = ({ domain, ip, isEditing, onEdit, onSave, onCancel, onDelete, loading }) => {
  const [editDomain, setEditDomain] = useState<string>(domain);
  const [editIp, setEditIp] = useState<string>(ip);

  useEffect(() => {
    if (isEditing) {
      setEditDomain(domain);
      setEditIp(ip);
    }
  }, [isEditing, domain, ip]);

  const handleSave = (): void => {
    onSave(domain, editDomain, editIp);
  };

  const handleCancel = (): void => {
    setEditDomain(domain);
    setEditIp(ip);
    onCancel();
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>): void => {
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
              onChange={(e: React.ChangeEvent<HTMLInputElement>) => setEditDomain(e.target.value)}
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
              onChange={(e: React.ChangeEvent<HTMLInputElement>) => setEditIp(e.target.value)}
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

export default DNSMappingRow;
