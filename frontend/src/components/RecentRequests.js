import React from 'react';
import { format } from 'date-fns';
import { CheckCircle, XCircle, Database, Clock, ExternalLink } from 'lucide-react';

const RecentRequests = ({ requests }) => {
  if (!requests || requests.length === 0) {
    return (
      <div className="bg-white rounded-lg shadow-md p-6">
        <h3 className="text-lg font-semibold text-gray-900 mb-4">Recent Requests</h3>
        <div className="text-center text-gray-500 py-8">
          No recent requests
        </div>
      </div>
    );
  }

  const getStatusIcon = (status) => {
    switch (status) {
      case 'success':
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

  const getStatusText = (status) => {
    switch (status) {
      case 'success':
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

  const getStatusColor = (status) => {
    switch (status) {
      case 'success':
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
    <div className="bg-white rounded-lg shadow-md p-6">
      <h3 className="text-lg font-semibold text-gray-900 mb-4">Recent Requests</h3>
      <div className="overflow-x-auto">
        <div className="max-h-96 overflow-y-auto">
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
              {requests.slice(0, 50).map((request, index) => (
                <tr
                  key={request.uuid || index}
                  className="hover:bg-gray-50 transition-colors"
                >
                  <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-900">
                    {format(new Date(request.timestamp), 'HH:mm:ss')}
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
              ))}
            </tbody>
          </table>
        </div>
      </div>
      {requests.length > 50 && (
        <div className="mt-4 text-center text-sm text-gray-500">
          Showing latest 50 of {requests.length} requests
        </div>
      )}
    </div>
  );
};

export default RecentRequests;
