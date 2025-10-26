import React from 'react';
import { Activity, Clock, Database, Users, Zap, TrendingUp, LucideIcon } from 'lucide-react';
import { OverviewMetrics } from '../../types';

type ColorType = 'blue' | 'green' | 'purple' | 'orange' | 'indigo' | 'pink';

interface OverviewCardProps {
  title: string;
  value: string;
  subtitle?: string;
  icon: LucideIcon;
  color?: ColorType;
}

interface OverviewCardsProps {
  overview: OverviewMetrics | null;
}

interface CardData {
  title: string;
  value: string;
  subtitle: string;
  icon: LucideIcon;
  color: ColorType;
}

const OverviewCard: React.FC<OverviewCardProps> = ({ title, value, subtitle, icon: Icon, color = 'blue' }) => {
  const colorClasses: Record<ColorType, string> = {
    blue: 'border-blue-200 bg-blue-50 text-blue-600',
    green: 'border-green-200 bg-green-50 text-green-600',
    purple: 'border-purple-200 bg-purple-50 text-purple-600',
    orange: 'border-orange-200 bg-orange-50 text-orange-600',
    indigo: 'border-indigo-200 bg-indigo-50 text-indigo-600',
    pink: 'border-pink-200 bg-pink-50 text-pink-600',
  };

  return (
    <div className="bg-white rounded-lg shadow-md p-6 border-l-4 border-blue-500 hover:shadow-lg transition-shadow">
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-sm font-medium text-gray-500 uppercase tracking-wide">
            {title}
          </h3>
          <div className="mt-2 flex items-baseline">
            <p className="text-2xl font-semibold text-gray-900">{value}</p>
          </div>
          {subtitle && (
            <p className="mt-1 text-sm text-gray-600">{subtitle}</p>
          )}
        </div>
        <div className={`p-3 rounded-full ${colorClasses[color]}`}>
          <Icon className="h-6 w-6" />
        </div>
      </div>
    </div>
  );
};

const OverviewCards: React.FC<OverviewCardsProps> = ({ overview }) => {
  if (!overview) {
    return (
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {[...Array(6)].map((_, i) => (
          <div key={i} className="bg-white rounded-lg shadow-md p-6 animate-pulse">
            <div className="flex items-center justify-between">
              <div className="flex-1">
                <div className="h-4 bg-gray-200 rounded w-24 mb-2"></div>
                <div className="h-8 bg-gray-200 rounded w-16 mb-1"></div>
                <div className="h-3 bg-gray-200 rounded w-20"></div>
              </div>
              <div className="w-12 h-12 bg-gray-200 rounded-full"></div>
            </div>
          </div>
        ))}
      </div>
    );
  }

  const formatNumber = (num: number | undefined): string => {
    if (!num) return '0';
    if (num >= 1000000) {
      return (num / 1000000).toFixed(1) + 'M';
    } else if (num >= 1000) {
      return (num / 1000).toFixed(1) + 'K';
    }
    return num.toString();
  };


  const cards: CardData[] = [
    {
      title: 'Total Requests',
      value: formatNumber(overview.total_requests),
      subtitle: `${overview.requests_per_second.toFixed(2)} req/sec`,
      icon: Activity,
      color: 'blue',
    },
    {
      title: 'Cache Hit Rate',
      value: `${overview.cache_hit_rate.toFixed(1)}%`,
      subtitle: 'Cache Performance',
      icon: Database,
      color: 'green',
    },
    {
      title: 'Success Rate',
      value: `${overview.success_rate.toFixed(1)}%`,
      subtitle: 'Query Success',
      icon: TrendingUp,
      color: 'purple',
    },
    {
      title: 'Avg Response Time',
      value: `${overview.average_response_time_ms.toFixed(1)} ms`,
      subtitle: 'Performance',
      icon: Zap,
      color: 'orange',
    },
    {
      title: 'Clients',
      value: formatNumber(overview.clients),
      subtitle: 'Connected',
      icon: Users,
      color: 'indigo',
    },
    {
      title: 'Uptime',
      value: overview.uptime,
      subtitle: 'System Uptime',
      icon: Clock,
      color: 'pink',
    },
  ];

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
      {cards.map((card, index) => (
        <OverviewCard key={index} {...card} />
      ))}
    </div>
  );
};

export default OverviewCards;
