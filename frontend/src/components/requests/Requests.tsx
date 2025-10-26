import React from 'react';
import { format } from 'date-fns';
import { CheckCircle, XCircle, Database, Clock, ExternalLink, ChevronLeft, ChevronRight } from 'lucide-react';
import type { DnsRequest, RequestsProps } from '../../types';

const Requests: React.FC<RequestsProps> = ({ 
  requests, 
  loading = false, 
  fullHeight = false,
  searchPerformed = false,
  currentPage = 0,
  totalResults = 0,
  pageSize = 50,
  onPageChange,
  searchTerm = ''
}) => {

  // Pagination handlers
  const nextPage = (): void => {
    if (onPageChange && (currentPage + 1) * pageSize < totalResults) {
      onPageChange(searchTerm, currentPage + 1);
    }
  };

  const prevPage = (): void => {
    if (onPageChange && currentPage > 0) {
      onPageChange(searchTerm, currentPage - 1);
    }
  };

  const displayRequests = requests || [];
  const showPagination = searchPerformed && totalResults > pageSize;

  const getStatusIcon = (status: string): JSX.Element => {
    switch (status) {
      case 'success':
      case 'custom_resolution':
        return <CheckCircle className="h-4 w-4 text-green-500" />;
      case 'cache_hit':
        return <Database className="h-4 w-4 text-blue-500" />;
      case 'all_upstreams_failed':
      case 'malformed_query':
        return <XCircle className="h-4 w-4 text-red-500" />;
      default:
        return <Clock className="h-4 w-4 text-gray-500" />;
    }
  };

  const getStatusText = (status: string): string => {
    switch (status) {
      case 'success':
      case 'custom_resolution':
        return 'Success';
      case 'cache_hit':
        return 'Cache Hit';
      case 'all_upstreams_failed':
        return 'Failed';
      case 'malformed_query':
        return 'Malformed';
      default:
        return status;
    }
  };

  const getStatusColor = (status: string): string => {
    switch (status) {
      case 'success':
      case 'custom_resolution':
        return 'bg-green-100 text-green-800 border-green-200';
      case 'cache_hit':
        return 'bg-blue-100 text-blue-800 border-blue-200';
      case 'all_upstreams_failed':
      case 'malformed_query':
        return 'bg-red-100 text-red-800 border-red-200';
      default:
        return 'bg-gray-100 text-gray-800 border-gray-200';
    }
  };

  return (
    <div className={`bg-white rounded-lg shadow-md ${fullHeight ? 'h-full flex flex-col' : ''}`}>
      {showPagination && (
        <div className="flex items-center justify-between p-4 border-b border-gray-200">
          <div className="text-sm text-gray-600">
            {totalResults.toLocaleString()} total result{totalResults !== 1 ? 's' : ''}
          </div>
          <div className="flex items-center space-x-2">
            <button
              onClick={prevPage}
              disabled={currentPage === 0}
              className="p-1 rounded hover:bg-gray-100 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <ChevronLeft className="h-4 w-4" />
            </button>
            <span className="text-xs">
              Page {currentPage + 1} of {Math.ceil(totalResults / pageSize)}
            </span>
            <button
              onClick={nextPage}
              disabled={(currentPage + 1) * pageSize >= totalResults}
              className="p-1 rounded hover:bg-gray-100 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <ChevronRight className="h-4 w-4" />
            </button>
          </div>
        </div>
      )}

      {loading && (
        <div className="text-center py-8">
          <div className="inline-flex items-center text-gray-600">
            <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-indigo-600 mr-2"></div>
            Loading requests...
          </div>
        </div>
      )}
      <div className={`overflow-x-auto ${fullHeight ? 'flex-1' : ''}`}>
        <div className={`${fullHeight ? 'h-full' : 'max-h-96'} overflow-y-auto`}>
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50 sticky top-0">
              <tr>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Time
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Query
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Type
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Client
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Status
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Duration
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Response IPs
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Upstream
                </th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {!loading && displayRequests && displayRequests.length > 0 ? displayRequests.map((request, index) => (
                <tr
                  key={request.uuid || index}
                  className="hover:bg-gray-50 transition-colors"
                >
                  <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-900">
                    <div className="flex flex-col">
                      <span className="font-medium">{format(new Date(request.timestamp), 'MM/dd/yyyy')}</span>
                      <span className="text-xs text-gray-600">{format(new Date(request.timestamp), 'HH:mm:ss')}</span>
                    </div>
                  </td>
                  <td className="px-4 py-3 text-sm text-gray-900 max-w-xs truncate">
                    <div className="flex items-center space-x-1">
                      <span className="truncate" title={request.request?.query}>
                        {request.request?.query}
                      </span>
                      {request.request?.query && (
                        <ExternalLink className="h-3 w-3 text-gray-400 flex-shrink-0" />
                      )}
                    </div>
                  </td>
                  <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-900">
                    <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-gray-100 text-gray-800">
                      {request.request?.type || 'A'}
                    </span>
                  </td>
                  <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-600">
                    {request.request?.client?.split(':')[0] || 'Unknown'}
                  </td>
                  <td className="px-4 py-3 whitespace-nowrap">
                    <div className="flex items-center space-x-2">
                      {getStatusIcon(request.status)}
                      <span
                        className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${getStatusColor(
                          request.status
                        )}`}
                      >
                        {getStatusText(request.status)}
                      </span>
                    </div>
                  </td>
                  <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-900">
                    <span className={`font-mono ${
                      request.total_duration_ms > 100 ? 'text-red-600' : 
                      request.total_duration_ms > 50 ? 'text-yellow-600' : 
                      'text-green-600'
                    }`}>
                      {request.total_duration_ms?.toFixed(1) || '0.0'}ms
                    </span>
                  </td>
                  <td className="px-4 py-3 text-sm text-gray-600 max-w-xs">
                    {request.ip_addresses && request.ip_addresses.length > 0 ? (
                      <div className="space-y-1">
                        {request.ip_addresses.slice(0, 3).map((ip, ipIndex) => (
                          <div key={ipIndex} className="flex items-center space-x-1">
                            <span className="font-mono text-xs bg-gray-100 px-2 py-1 rounded">
                              {ip}
                            </span>
                          </div>
                        ))}
                        {request.ip_addresses.length > 3 && (
                          <div className="text-xs text-gray-400">
                            +{request.ip_addresses.length - 3} more
                          </div>
                        )}
                      </div>
                    ) : request.status === 'success' && request.response?.answer_count > 0 ? (
                      <span className="text-xs text-gray-400 italic">
                        {request.response.answer_count} answer{request.response.answer_count !== 1 ? 's' : ''}
                      </span>
                    ) : (
                      <span className="text-gray-400">-</span>
                    )}
                  </td>
                  <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-600">
                    {request.response?.upstream === 'cache' ? (
                      <div className="flex items-center space-x-1">
                        <Database className="h-3 w-3 text-blue-500" />
                        <span className="text-blue-600 font-medium">Cache</span>
                      </div>
                    ) : request.response?.upstream ? (
                      <div className="flex flex-col">
                        <span className="truncate" title={request.response.upstream}>
                          {request.response.upstream}
                        </span>
                        {request.response.rtt_ms && (
                          <span className="text-xs text-gray-400">
                            RTT: {request.response.rtt_ms.toFixed(1)}ms
                          </span>
                        )}
                      </div>
                    ) : (
                      <span className="text-gray-400">-</span>
                    )}
                  </td>
                </tr>
              )) : !loading ? (
                <tr>
                  <td colSpan="8" className="px-4 py-8 text-center text-gray-500">
                    {searchPerformed ? (
                      searchTerm ? (
                        <>No results found for "<span className="font-medium">{searchTerm}</span>"</>
                      ) : (
                        'No DNS logs found'
                      )
                    ) : (
                      'No requests'
                    )}
                  </td>
                </tr>
              ) : null}
            </tbody>
          </table>
        </div>
      </div>

      {!searchPerformed && requests && requests.length > 50 && (
        <div className="p-4 text-center text-sm text-gray-500 border-t border-gray-200">
          Showing latest 50 of {requests.length} requests
        </div>
      )}
    </div>
  );
};

export default Requests;
