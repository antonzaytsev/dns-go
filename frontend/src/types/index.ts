// Shared TypeScript interfaces for the DNS Dashboard

// ===== BASE/UTILITY TYPES =====

// Common callback function types
export type VoidCallback = () => void;
export type StringCallback = (value: string) => void;
export type DomainCallback = (domain: string) => void;

// Common state patterns
export interface LoadingState {
  loading: boolean;
}

export interface ErrorState {
  error: string | null;
}

export interface SuccessState {
  success: string | null;
}

export interface MessageState extends ErrorState, SuccessState {}

export interface AsyncOperationState extends LoadingState, ErrorState {
  lastUpdated: Date | null;
  refresh: VoidCallback;
}

// ===== CORE DATA INTERFACES =====

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

// DNS Mapping - unified interface (removed duplicate DNSMappingEntry)
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

// ===== COMPONENT PROPS INTERFACES =====

// Using base types for consistency
export interface TopClientsProps {
  clients: Client[];
}

export interface StatusMessagesProps extends MessageState {}

export interface RecentRequestsProps extends Partial<LoadingState> {
  requests: DnsRequest[];
  fullHeight?: boolean;
}

export interface RecentRequestsFullHeightProps extends LoadingState {
  requests: DnsRequest[];
}

// ===== HOOK RETURN TYPES =====

// Base hook return type that most hooks share
interface BaseHookReturn extends AsyncOperationState {
  // Common fields: loading, error, lastUpdated, refresh
}

export interface UseMetricsReturn extends BaseHookReturn {
  metrics: Metrics | null;
}

export interface UseHealthReturn extends LoadingState, ErrorState {
  health: HealthStatus | null;
  isHealthy: boolean;
}

export interface UseRecentRequestsReturn extends BaseHookReturn {
  recentRequests: DnsRequest[];
}

// ===== TIME SERIES & CHARTS =====

export interface TimeSeriesDataPoint {
  timestamp: string;
  value: number;
}

export interface TimeSeriesData {
  requests_last_hour?: TimeSeriesDataPoint[];
  requests_last_day?: TimeSeriesDataPoint[];
  requests_last_week?: TimeSeriesDataPoint[];
  requests_last_month?: TimeSeriesDataPoint[];
}

export interface ChartsProps {
  timeSeriesData?: TimeSeriesData | null;
}

export interface ConnectionStatusProps {
  isOnline: boolean;
  lastUpdated?: Date | null;
  error?: string | null;
}

// ===== DNS MAPPINGS =====

// DNS mapping state (Record is more efficient than individual interface)
export type DNSMappingsState = Record<string, string>;

// Modal confirmation state
export interface ModalState {
  show: boolean;
  domain: string;
}

// DNS mapping form/row operations - reusable callback types
export type MappingChangeCallback = (field: keyof DNSMapping, value: string) => void;
export type MappingSaveCallback = (originalDomain: string, newDomain: string, newIp: string) => void;

// DNS Mappings Props with base type composition
export interface AddMappingFormProps extends LoadingState {
  newMapping: DNSMapping;
  onMappingChange: MappingChangeCallback;
  onSubmit: VoidCallback;
  onCancel: VoidCallback;
}

export interface DNSMappingRowProps extends LoadingState {
  domain: string;
  ip: string;
  isEditing: boolean;
  onEdit: VoidCallback;
  onSave: MappingSaveCallback;
  onCancel: VoidCallback;
  onDelete: VoidCallback;
}

export interface DeleteConfirmationModalProps extends LoadingState {
  isOpen: boolean;
  domain: string;
  onConfirm: VoidCallback;
  onCancel: VoidCallback;
}

export interface DNSMappingsListProps extends LoadingState {
  mappings: DNSMappingsState;
  editingDomain: string | null;
  onEdit: DomainCallback;
  onSave: MappingSaveCallback;
  onCancelEdit: VoidCallback;
  onDelete: DomainCallback;
}

export interface DNSMappingsHeaderProps extends LoadingState {
  onRefresh: VoidCallback;
  onAddMapping: VoidCallback;
  mappingsCount: number;
}
