import { useState, useEffect, useCallback } from 'react';
import { dnsApi } from '../services/api.ts';

export const useMetrics = (refreshInterval = 5000) => {
  const [metrics, setMetrics] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [lastUpdated, setLastUpdated] = useState(null);

  const fetchMetrics = useCallback(async () => {
    try {
      setError(null);
      const data = await dnsApi.getMetrics();
      setMetrics(data);
      setLastUpdated(new Date());
      setLoading(false);
    } catch (err) {
      setError(err.message || 'Failed to fetch metrics');
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    // Initial fetch
    fetchMetrics();

    // Set up interval for periodic updates
    const interval = setInterval(fetchMetrics, refreshInterval);

    // Cleanup interval on unmount
    return () => clearInterval(interval);
  }, [fetchMetrics, refreshInterval]);

  // Manual refresh function
  const refresh = useCallback(() => {
    setLoading(true);
    fetchMetrics();
  }, [fetchMetrics]);

  return {
    metrics,
    loading,
    error,
    lastUpdated,
    refresh,
  };
};

export const useHealth = (refreshInterval = 30000) => {
  const [health, setHealth] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  const fetchHealth = useCallback(async () => {
    try {
      setError(null);
      const data = await dnsApi.getHealth();
      setHealth(data);
      setLoading(false);
    } catch (err) {
      setError(err.message || 'Failed to fetch health status');
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchHealth();
    const interval = setInterval(fetchHealth, refreshInterval);
    return () => clearInterval(interval);
  }, [fetchHealth, refreshInterval]);

  return {
    health,
    loading,
    error,
    isHealthy: health?.status === 'healthy',
  };
};

export const useRecentRequests = (refreshInterval = 5000) => {
  const [recentRequests, setRecentRequests] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [lastUpdated, setLastUpdated] = useState(null);

  const fetchRecentRequests = useCallback(async () => {
    try {
      setError(null);
      // Load last 50 requests using empty search query
      const data = await dnsApi.searchLogs('', 50, 0);
      setRecentRequests(data.results || []);
      setLastUpdated(new Date());
      setLoading(false);
    } catch (err) {
      setError(err.message || 'Failed to fetch recent requests');
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    // Initial fetch
    fetchRecentRequests();

    // Set up interval for periodic updates
    const interval = setInterval(fetchRecentRequests, refreshInterval);

    // Cleanup interval on unmount
    return () => clearInterval(interval);
  }, [fetchRecentRequests, refreshInterval]);

  // Manual refresh function
  const refresh = useCallback(() => {
    setLoading(true);
    fetchRecentRequests();
  }, [fetchRecentRequests]);

  return {
    recentRequests,
    loading,
    error,
    lastUpdated,
    refresh,
  };
};
