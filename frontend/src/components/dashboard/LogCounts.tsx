import React, { useState, useEffect } from 'react';
import { Database, AlertCircle } from 'lucide-react';
import { dnsApi } from '../../services/api.ts';
import type { LogCounts as LogCountsType } from '../../types/index.ts';

const LogCounts: React.FC = () => {
  const [logCounts, setLogCounts] = useState<LogCountsType | null>(null);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);

  const fetchLogCounts = async () => {
    try {
      setError(null);
      const data = await dnsApi.getLogCounts();
      setLogCounts(data);
      setLoading(false);
    } catch (err: any) {
      setError(err.message || 'Failed to fetch log counts');
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchLogCounts();
    const interval = setInterval(fetchLogCounts, 30000); // Refresh every 30 seconds
    return () => clearInterval(interval);
  }, []);

  const formatNumber = (num: number | null | undefined): string => {
    if (num === null || num === undefined) return 'N/A';
    return num.toLocaleString();
  };

  if (loading) {
    return (
      <div className="bg-white rounded-lg shadow-md p-6">
        <h3 className="text-lg font-semibold text-gray-900 mb-4">Log Storage Statistics</h3>
        <div className="grid grid-cols-1 gap-4">
          <div className="animate-pulse">
            <div className="h-20 bg-gray-200 rounded"></div>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="bg-white rounded-lg shadow-md p-6">
      <h3 className="text-lg font-semibold text-gray-900 mb-4">Log Storage Statistics</h3>
      <div className="grid grid-cols-1 gap-4">
        {/* PostgreSQL Count */}
        <div className="border border-gray-200 rounded-lg p-4">
          <div className="flex items-center justify-between mb-2">
            <div className="flex items-center space-x-2">
              <Database className="h-5 w-5 text-blue-500" />
              <h4 className="text-sm font-medium text-gray-700">PostgreSQL</h4>
            </div>
          </div>
          {logCounts?.postgres?.error ? (
            <div className="flex items-center space-x-2 text-red-600">
              <AlertCircle className="h-4 w-4" />
              <span className="text-sm">{logCounts.postgres.error}</span>
            </div>
          ) : (
            <div>
              <p className="text-2xl font-bold text-gray-900">
                {formatNumber(logCounts?.postgres?.count)}
              </p>
              <p className="text-xs text-gray-500 mt-1">Total log records</p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default LogCounts;

