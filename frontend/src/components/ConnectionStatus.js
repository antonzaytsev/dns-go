import React, { useState, useEffect } from 'react';
import { Wifi, WifiOff, AlertCircle } from 'lucide-react';

const ConnectionStatus = ({ isOnline, lastUpdated, error }) => {
  const [showDetails, setShowDetails] = useState(false);

  // Auto-hide error details after 5 seconds
  useEffect(() => {
    if (error && showDetails) {
      const timer = setTimeout(() => setShowDetails(false), 5000);
      return () => clearTimeout(timer);
    }
  }, [error, showDetails]);

  const getStatusColor = () => {
    if (error) return 'text-red-500';
    return isOnline ? 'text-green-500' : 'text-gray-400';
  };

  const getStatusText = () => {
    if (error) return 'Error';
    return isOnline ? 'Connected' : 'Connecting...';
  };

  const StatusIcon = error ? AlertCircle : (isOnline ? Wifi : WifiOff);

  return (
    <div className="flex items-center space-x-2">
      <StatusIcon className={`h-4 w-4 ${getStatusColor()}`} />
      <span className={`text-sm ${getStatusColor()}`}>
        {getStatusText()}
      </span>
      {lastUpdated && (
        <span className="text-xs text-gray-500">
          • Updated {lastUpdated.toLocaleTimeString()}
        </span>
      )}
      {error && (
        <button
          onClick={() => setShowDetails(!showDetails)}
          className="text-xs text-red-600 hover:text-red-800 underline"
        >
          Details
        </button>
      )}
      {error && showDetails && (
        <div className="absolute top-full left-0 mt-2 p-2 bg-red-50 border border-red-200 rounded text-xs text-red-700 max-w-md z-10">
          {error}
        </div>
      )}
    </div>
  );
};

export default ConnectionStatus;
