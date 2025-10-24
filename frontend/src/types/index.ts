// Shared TypeScript interfaces for the DNS Dashboard

export interface DnsRequest {
  uuid?: string;
  timestamp: string;
  request?: {
    query?: string;
    type?: string;
    client?: string;
  };
  status: string;
  duration_ms?: number;
  response?: {
    ips?: string[];
  };
  upstream?: string;
}

export interface Client {
  ip: string;
  requests: number;
  cache_hit_rate: number;
  success_rate: number;
  last_seen: string;
}

export interface Metrics {
  total_requests?: number;
  cache_hits?: number;
  cache_misses?: number;
  upstream_requests?: number;
  failed_requests?: number;
  avg_response_time?: number;
  clients?: Client[];
  query_types?: Record<string, number>;
  recent_requests?: DnsRequest[];
  uptime?: string;
  version?: string;
}

export interface HealthStatus {
  status: string;
  uptime?: string;
  version?: string;
  timestamp?: string;
}

export interface VersionInfo {
  version: string;
  git_commit?: string;
  build_date?: string;
  go_version?: string;
}

export interface SearchResponse {
  results: DnsRequest[];
  total: number;
  source?: string;
}

export interface DNSMapping {
  domain: string;
  ip: string;
}

export interface DNSMappingsResponse {
  mappings: Record<string, string>;
}

export interface APIResponse<T = any> {
  data: T;
  message?: string;
  success?: boolean;
}

// Component-specific prop interfaces
export interface TopClientsProps {
  clients: Client[];
}

export interface StatusMessagesProps {
  error?: string | null;
  success?: string | null;
}

export interface RecentRequestsProps {
  requests: DnsRequest[];
  loading?: boolean;
  fullHeight?: boolean;
}

export interface RecentRequestsFullHeightProps {
  requests: DnsRequest[];
  loading: boolean;
}
