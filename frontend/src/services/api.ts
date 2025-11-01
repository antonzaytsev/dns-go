import axios, { AxiosInstance, AxiosResponse, AxiosError, InternalAxiosRequestConfig } from 'axios';
import type {
  Metrics,
  HealthStatus,
  VersionInfo,
  SearchResponse,
  DNSMappingsResponse,
  ClientsResponse,
  APIResponse,
  LogCounts,
} from '../types';

const port: string = process.env.REACT_APP_API_PORT || '8080';

// Create axios instance with base configuration
const api: AxiosInstance = axios.create({
  baseURL: `http://${window.location.hostname}:${port}`,
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Request interceptor for logging
api.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    return config;
  },
  (error: AxiosError) => {
    console.error('API Request Error:', error);
    return Promise.reject(error);
  }
);

// Response interceptor for error handling
api.interceptors.response.use(
  (response: AxiosResponse) => {
    return response;
  },
  (error: AxiosError) => {
    console.error('API Response Error:', error.response?.status, error.message);
    return Promise.reject(error);
  }
);

// API service functions
export const dnsApi = {
  // Get DNS server metrics
  getMetrics: async (): Promise<Metrics> => {
    try {
      const response: AxiosResponse<Metrics> = await api.get('/api/metrics');
      return response.data;
    } catch (error) {
      console.error('Failed to fetch metrics:', error);
      throw error;
    }
  },

  // Get all DNS clients
  getClients: async (): Promise<ClientsResponse> => {
    try {
      const response: AxiosResponse<ClientsResponse> = await api.get('/api/clients');
      return response.data;
    } catch (error) {
      console.error('Failed to fetch clients:', error);
      throw error;
    }
  },

  // Get health status
  getHealth: async (): Promise<HealthStatus> => {
    try {
      const response: AxiosResponse<HealthStatus> = await api.get('/api/health');
      return response.data;
    } catch (error) {
      console.error('Failed to fetch health status:', error);
      throw error;
    }
  },

  // Get version information
  getVersion: async (): Promise<VersionInfo> => {
    try {
      const response: AxiosResponse<VersionInfo> = await api.get('/api/version');
      return response.data;
    } catch (error) {
      console.error('Failed to fetch version:', error);
      throw error;
    }
  },

  // Get log counts from Elasticsearch and PostgreSQL
  getLogCounts: async (): Promise<LogCounts> => {
    try {
      const response: AxiosResponse<LogCounts> = await api.get('/api/log-counts');
      return response.data;
    } catch (error) {
      console.error('Failed to fetch log counts:', error);
      throw error;
    }
  },

  // Search DNS logs
  searchLogs: async (
    searchTerm: string,
    limit: number = 100,
    offset: number = 0,
    since: Date | string | null = null
  ): Promise<SearchResponse> => {
    try {
      const params = new URLSearchParams();
      if (searchTerm) params.append('q', searchTerm);
      params.append('limit', limit.toString());
      params.append('offset', offset.toString());

      if (since !== null) {
        // Accept Date objects or ISO strings (server expects format: 2024-01-02T15:04:05Z)
        let sinceStr: string;
        if (since instanceof Date) {
          // Convert to required format: 2024-01-02T15:04:05Z
          sinceStr = since.toISOString().replace(/\.\d{3}Z$/, 'Z');
        } else {
          sinceStr = since;
        }
        params.append('since', sinceStr);
      }

      const response: AxiosResponse<SearchResponse> = await api.get(`/api/search?${params.toString()}`);
      return response.data;
    } catch (error) {
      console.error('Failed to search logs:', error);
      throw error;
    }
  },

  // DNS mappings management
  getDNSMappings: async (): Promise<DNSMappingsResponse> => {
    try {
      const response: AxiosResponse<DNSMappingsResponse> = await api.get('/api/dns-mappings');
      return response.data;
    } catch (error) {
      console.error('Failed to fetch DNS mappings:', error);
      throw error;
    }
  },

  addDNSMapping: async (domain: string, ip: string): Promise<APIResponse> => {
    try {
      const response: AxiosResponse<APIResponse> = await api.post('/api/dns-mappings', { domain, ip });
      return response.data;
    } catch (error) {
      console.error('Failed to add DNS mapping:', error);
      throw error;
    }
  },

  deleteDNSMapping: async (domain: string): Promise<APIResponse> => {
    try {
      const response: AxiosResponse<APIResponse> = await api.delete(`/api/dns-mappings?domain=${encodeURIComponent(domain)}`);
      return response.data;
    } catch (error) {
      console.error('Failed to delete DNS mapping:', error);
      throw error;
    }
  },
};

export default api;
