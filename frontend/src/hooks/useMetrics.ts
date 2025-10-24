import { useState, useEffect, useCallback } from 'react';
import { dnsApi } from '../services/api.ts';
import type { 
  Metrics, 
  HealthStatus, 
  DnsRequest, 
  SearchResponse, 
  UseMetricsReturn,
  UseHealthReturn,
  UseRecentRequestsReturn 
} from '../types';

export const useMetrics = (refreshInterval: number = 5000): UseMetricsReturn => {
  const [metrics, setMetrics] = useState<Metrics | null>(null);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);

  const fetchMetrics = useCallback(async (): Promise<void> => {
    try {
      setError(null);
      const data: Metrics = await dnsApi.getMetrics();
      setMetrics(data);
      setLastUpdated(new Date());
      setLoading(false);
    } catch (err: any) {
      setError(err.message || 'Failed to fetch metrics');
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    // Initial fetch
    fetchMetrics();

    // Set up interval for periodic updates
    const interval: NodeJS.Timeout = setInterval(fetchMetrics, refreshInterval);

    // Cleanup interval on unmount
    return () => clearInterval(interval);
  }, [fetchMetrics, refreshInterval]);

  // Manual refresh function
  const refresh = useCallback((): void => {
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

export const useHealth = (refreshInterval: number = 30000): UseHealthReturn => {
  const [health, setHealth] = useState<HealthStatus | null>(null);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);

  const fetchHealth = useCallback(async (): Promise<void> => {
    try {
      setError(null);
      const data: HealthStatus = await dnsApi.getHealth();
      setHealth(data);
      setLoading(false);
    } catch (err: any) {
      setError(err.message || 'Failed to fetch health status');
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchHealth();
    const interval: NodeJS.Timeout = setInterval(fetchHealth, refreshInterval);
    return () => clearInterval(interval);
  }, [fetchHealth, refreshInterval]);

  return {
    health,
    loading,
    error,
    isHealthy: health?.status === 'healthy',
  };
};

export const useRecentRequests = (refreshInterval: number = 5000): UseRecentRequestsReturn => {
  const [recentRequests, setRecentRequests] = useState<DnsRequest[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);

  const fetchRecentRequests = useCallback(async (): Promise<void> => {
    try {
      setError(null);
      // Load last 50 requests using empty search query
      const data: SearchResponse = await dnsApi.searchLogs('', 50, 0);
      setRecentRequests(data.results || []);
      setLastUpdated(new Date());
      setLoading(false);
    } catch (err: any) {
      setError(err.message || 'Failed to fetch recent requests');
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    // Initial fetch
    fetchRecentRequests();

    // Set up interval for periodic updates
    const interval: NodeJS.Timeout = setInterval(fetchRecentRequests, refreshInterval);

    // Cleanup interval on unmount
    return () => clearInterval(interval);
  }, [fetchRecentRequests, refreshInterval]);

  // Manual refresh function
  const refresh = useCallback((): void => {
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
