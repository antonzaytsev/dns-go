import React from 'react';
import { format } from 'date-fns';
import { CheckCircle, XCircle, Database, Clock } from 'lucide-react';

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
      <div className="max-h-96 overflow-y-auto space-y-3">
        {requests.slice(0, 20).map((request, index) => (
          <div
            key={request.uuid || index}
            className="border rounded-lg p-4 hover:bg-gray-50 transition-colors"
          >
            <div className="flex items-center justify-between mb-2">
              <div className="flex items-center space-x-2">
                {getStatusIcon(request.status)}
                <span className="font-medium text-gray-900">
                  {request.request?.query} ({request.request?.type})
                </span>
              </div>
              <span
                className={`px-2 py-1 text-xs font-medium rounded-full border ${getStatusColor(
                  request.status
                )}`}
              >
                {getStatusText(request.status)}
              </span>
            </div>
            <div className="flex items-center justify-between text-sm text-gray-600">
              <div className="flex items-center space-x-4">
                <span>Client: {request.request?.client}</span>
                <span>Duration: {request.total_duration_ms?.toFixed(1)}ms</span>
              </div>
              <span>
                {format(new Date(request.timestamp), 'HH:mm:ss')}
              </span>
            </div>
            {request.response && request.response.upstream !== 'cache' && (
              <div className="mt-2 text-xs text-gray-500">
                Upstream: {request.response.upstream} (RTT: {request.response.rtt_ms?.toFixed(1)}ms)
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  );
};

export default RecentRequests;
