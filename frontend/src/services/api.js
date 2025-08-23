import axios from 'axios';

const port = process.env.API_PORT || '8080'

// Create axios instance with base configuration
const api = axios.create({
  baseURL: `http://${window.location.hostname}:${port}`,
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Request interceptor for logging
api.interceptors.request.use(
  (config) => {
    return config;
  },
  (error) => {
    console.error('API Request Error:', error);
    return Promise.reject(error);
  }
);

// Response interceptor for error handling
api.interceptors.response.use(
  (response) => {
    return response;
  },
  (error) => {
    console.error('API Response Error:', error.response?.status, error.message);
    return Promise.reject(error);
  }
);

// API service functions
export const dnsApi = {
  // Get DNS server metrics
  getMetrics: async () => {
    try {
      const response = await api.get('/api/metrics');
      return response.data;
    } catch (error) {
      console.error('Failed to fetch metrics:', error);
      throw error;
    }
  },

  // Get health status
  getHealth: async () => {
    try {
      const response = await api.get('api//health');
      return response.data;
    } catch (error) {
      console.error('Failed to fetch health status:', error);
      throw error;
    }
  },

  // Get version information
  getVersion: async () => {
    try {
      const response = await api.get('api//version');
      return response.data;
    } catch (error) {
      console.error('Failed to fetch version:', error);
      throw error;
    }
  },

  // Search DNS logs
  searchLogs: async (searchTerm, limit = 100, offset = 0) => {
    try {
      const params = new URLSearchParams();
      if (searchTerm) params.append('q', searchTerm);
      params.append('limit', limit.toString());
      params.append('offset', offset.toString());

      const response = await api.get(`api//search?${params.toString()}`);
      return response.data;
    } catch (error) {
      console.error('Failed to search logs:', error);
      throw error;
    }
  },
};

export default api;
