import { useState, useEffect, useCallback } from 'react';
import { dnsApi } from '../services/api.ts';
import type { 
  Metrics, 
  HealthStatus, 
  DnsRequest, 
  SearchResponse, 
  UseMetricsReturn,
  UseHealthReturn,
  UseRequestsReturn 
} from '../types/index.ts';

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

export const useRequests = (refreshInterval: number = 5000): UseRequestsReturn => {
  const [requests, setRequests] = useState<DnsRequest[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);

  const fetchRequests = useCallback(async (): Promise<void> => {
    try {
      setError(null);
      // Load last 50 requests with no filters
      const data: SearchResponse = await dnsApi.searchLogs('', '', 50, 0);
      setRequests(data.results || []);
      setLastUpdated(new Date());
      setLoading(false);
    } catch (err: any) {
      setError(err.message || 'Failed to fetch requests');
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    // Initial fetch
    fetchRequests();

    // Set up interval for periodic updates
    const interval: NodeJS.Timeout = setInterval(fetchRequests, refreshInterval);

    // Cleanup interval on unmount
    return () => clearInterval(interval);
  }, [fetchRequests, refreshInterval]);

  // Manual refresh function
  const refresh = useCallback((): void => {
    setLoading(true);
    fetchRequests();
  }, [fetchRequests]);

  return {
    requests,
    loading,
    error,
    lastUpdated,
    refresh,
  };
};
