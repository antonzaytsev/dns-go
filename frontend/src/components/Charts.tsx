import React, { useState } from 'react';
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  BarElement,
  Title,
  Tooltip,
  Legend,
  ChartOptions,
} from 'chart.js';
import { Bar } from 'react-chartjs-2';
import { format, endOfWeek } from 'date-fns';
import { Clock, BarChart3, Calendar, CalendarDays } from 'lucide-react';
import type { ChartsProps, TimeSeriesDataPoint } from '../types';

ChartJS.register(
  CategoryScale,
  LinearScale,
  BarElement,
  Title,
  Tooltip,
  Legend
);

const chartOptions: ChartOptions<'bar'> = {
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: {
      display: false,
    },
    tooltip: {
      mode: 'index',
      intersect: false,
    },
  },
  scales: {
    y: {
      beginAtZero: true,
      ticks: {
        precision: 0,
      },
    },
    x: {
      ticks: {
        maxTicksLimit: 10,
      },
    },
  },
  interaction: {
    mode: 'nearest',
    axis: 'x',
    intersect: false,
  },
};

type ChartView = 'minute' | 'hour' | 'day' | 'week';

const Charts: React.FC<ChartsProps> = ({ timeSeriesData }) => {
  const [activeView, setActiveView] = useState<ChartView>('hour');

  if (!timeSeriesData) {
    return (
      <div className="bg-white rounded-lg shadow-md p-6">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold text-gray-900">
            Request Activity
          </h3>
          <div className="flex space-x-1">
            <div className="w-16 h-8 bg-gray-200 rounded animate-pulse"></div>
            <div className="w-16 h-8 bg-gray-200 rounded animate-pulse"></div>
            <div className="w-16 h-8 bg-gray-200 rounded animate-pulse"></div>
            <div className="w-16 h-8 bg-gray-200 rounded animate-pulse"></div>
          </div>
        </div>
        <div className="h-64 flex items-center justify-center">
          <div className="animate-pulse text-gray-400">Loading chart...</div>
        </div>
      </div>
    );
  }


  // Use pre-aggregated minute data (75 minute slots)
  const minuteData: TimeSeriesDataPoint[] = timeSeriesData.requests_last_hour || [];
  const minuteChartData = {
    labels: minuteData.map((point: TimeSeriesDataPoint): string => {
      const date = new Date(point.timestamp);
      return format(date, 'HH:mm');
    }),
    datasets: [
      {
        label: 'Requests per Minute',
        data: minuteData.map((point: TimeSeriesDataPoint): number => point.value),
        backgroundColor: 'rgba(59, 130, 246, 0.8)',
        borderColor: 'rgb(59, 130, 246)',
        borderWidth: 1,
      },
    ],
  };

  // Use pre-aggregated hour data (75 hour slots)
  const hourData: TimeSeriesDataPoint[] = timeSeriesData.requests_last_day || [];
  const hourChartData = {
    labels: hourData.map((point: TimeSeriesDataPoint): string => {
      const date = new Date(point.timestamp);
      return format(date, 'MMM dd HH:mm');
    }),
    datasets: [
      {
        label: 'Requests per Hour',
        data: hourData.map((point: TimeSeriesDataPoint): number => point.value),
        backgroundColor: 'rgba(99, 102, 241, 0.8)',
        borderColor: 'rgb(99, 102, 241)',
        borderWidth: 1,
      },
    ],
  };

  // Use pre-aggregated day data (75 day slots)
  const dayData: TimeSeriesDataPoint[] = timeSeriesData.requests_last_week || [];
  const dayChartData = {
    labels: dayData.map((point: TimeSeriesDataPoint): string => {
      const date = new Date(point.timestamp);
      return format(date, 'MMM dd');
    }),
    datasets: [
      {
        label: 'Requests per Day',
        data: dayData.map((point: TimeSeriesDataPoint): number => point.value),
        backgroundColor: 'rgba(16, 185, 129, 0.8)',
        borderColor: 'rgb(16, 185, 129)',
        borderWidth: 1,
      },
    ],
  };

  // Use pre-aggregated week data (75 week slots)
  const weekData: TimeSeriesDataPoint[] = timeSeriesData.requests_last_month || [];
  const weekChartData = {
    labels: weekData.map((point: TimeSeriesDataPoint): string => {
      const weekStart = new Date(point.timestamp);
      const weekEnd = endOfWeek(weekStart, { weekStartsOn: 1 });
      return `${format(weekStart, 'MMM dd')} - ${format(weekEnd, 'MMM dd')}`;
    }),
    datasets: [
      {
        label: 'Requests per Week',
        data: weekData.map((point: TimeSeriesDataPoint): number => point.value),
        backgroundColor: 'rgba(168, 85, 247, 0.8)',
        borderColor: 'rgb(168, 85, 247)',
        borderWidth: 1,
      },
    ],
  };

  // Determine current data, component, and title based on active view
  const getChartConfig = () => {
    switch (activeView) {
      case 'minute':
        return {
          data: minuteChartData,
          component: Bar,
          title: 'Requests per Minute (Last 75 Minutes)'
        };
      case 'hour':
        return {
          data: hourChartData,
          component: Bar,
          title: 'Requests per Hour (Last 75 Hours)'
        };
      case 'day':
        return {
          data: dayChartData,
          component: Bar,
          title: 'Requests per Day (Last 75 Days)'
        };
      case 'week':
        return {
          data: weekChartData,
          component: Bar,
          title: 'Requests per Week (Last 75 Weeks)'
        };
      default:
        return {
          data: hourChartData,
          component: Bar,
          title: 'Requests per Hour (Last 75 Hours)'
        };
    }
  };

  const { data: currentData, component: ChartComponent, title } = getChartConfig();

  return (
    <div className="bg-white rounded-lg shadow-md p-6">
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-lg font-semibold text-gray-900">
          {title}
        </h3>
        <div className="flex space-x-1">
          <button
            onClick={() => setActiveView('minute')}
            className={`inline-flex items-center px-2.5 py-1.5 text-sm font-medium rounded-md transition-colors ${
              activeView === 'minute'
                ? 'bg-blue-100 text-blue-700 border border-blue-200'
                : 'bg-gray-100 text-gray-600 border border-gray-200 hover:bg-gray-200'
            }`}
          >
            <Clock className="h-4 w-4 mr-1" />
            Minute
          </button>
          <button
            onClick={() => setActiveView('hour')}
            className={`inline-flex items-center px-2.5 py-1.5 text-sm font-medium rounded-md transition-colors ${
              activeView === 'hour'
                ? 'bg-blue-100 text-blue-700 border border-blue-200'
                : 'bg-gray-100 text-gray-600 border border-gray-200 hover:bg-gray-200'
            }`}
          >
            <BarChart3 className="h-4 w-4 mr-1" />
            Hour
          </button>
          <button
            onClick={() => setActiveView('day')}
            className={`inline-flex items-center px-2.5 py-1.5 text-sm font-medium rounded-md transition-colors ${
              activeView === 'day'
                ? 'bg-blue-100 text-blue-700 border border-blue-200'
                : 'bg-gray-100 text-gray-600 border border-gray-200 hover:bg-gray-200'
            }`}
          >
            <Calendar className="h-4 w-4 mr-1" />
            Day
          </button>
          <button
            onClick={() => setActiveView('week')}
            className={`inline-flex items-center px-2.5 py-1.5 text-sm font-medium rounded-md transition-colors ${
              activeView === 'week'
                ? 'bg-blue-100 text-blue-700 border border-blue-200'
                : 'bg-gray-100 text-gray-600 border border-gray-200 hover:bg-gray-200'
            }`}
          >
            <CalendarDays className="h-4 w-4 mr-1" />
            Week
          </button>
        </div>
      </div>
      <div className="h-64">
        <ChartComponent data={currentData} options={chartOptions} />
      </div>
    </div>
  );
};

export default Charts;
