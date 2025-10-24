import React, { useEffect } from 'react';
import { AlertCircle, RefreshCw } from 'lucide-react';

const DeleteConfirmationModal = ({ 
  isOpen, 
  domain, 
  onConfirm, 
  onCancel, 
  loading 
}) => {
  const handleBackdropClick = (e) => {
    if (e.target === e.currentTarget) {
      onCancel();
    }
  };

  useEffect(() => {
    if (isOpen) {
      const modalElement = document.querySelector('[data-modal="delete-confirmation"]');
      if (modalElement) {
        modalElement.focus();
      }

      const handleKeyDown = (e) => {
        if (e.key === 'Escape') {
          onCancel();
        }
      };

      document.addEventListener('keydown', handleKeyDown);
      return () => document.removeEventListener('keydown', handleKeyDown);
    }
  }, [isOpen, onCancel]);

  if (!isOpen) return null;

  return (
    <div
      className="fixed inset-0 bg-gray-600 bg-opacity-50 overflow-y-auto h-full w-full z-50"
      onClick={handleBackdropClick}
    >
      <div className="relative top-20 mx-auto p-5 border w-96 shadow-lg rounded-md bg-white">
        <div
          data-modal="delete-confirmation"
          tabIndex={-1}
          className="outline-none"
        >
          <div className="flex items-start">
            <div className="flex-shrink-0">
              <AlertCircle className="h-6 w-6 text-red-600" />
            </div>
            <div className="ml-4 w-full">
              <h3 className="text-lg font-medium text-gray-900 mb-2">
                Delete DNS Mapping
              </h3>
              <p className="text-sm text-gray-500 mb-4">
                Are you sure you want to delete the DNS mapping for{' '}
                <span className="font-medium text-gray-900">"{domain}"</span>?
                This action cannot be undone.
              </p>
              <div className="flex justify-end space-x-3">
                <button
                  onClick={onCancel}
                  disabled={loading}
                  className="px-4 py-2 text-sm font-medium text-gray-700 bg-gray-200 border border-gray-300 rounded-md hover:bg-gray-300 focus:outline-none focus:ring-2 focus:ring-gray-500 disabled:opacity-50"
                >
                  Cancel
                </button>
                <button
                  onClick={onConfirm}
                  disabled={loading}
                  className="px-4 py-2 text-sm font-medium text-white bg-red-600 border border-transparent rounded-md hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-red-500 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {loading ? (
                    <>
                      <RefreshCw className="h-4 w-4 mr-2 animate-spin inline" />
                      Deleting...
                    </>
                  ) : (
                    'Delete'
                  )}
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default DeleteConfirmationModal;
