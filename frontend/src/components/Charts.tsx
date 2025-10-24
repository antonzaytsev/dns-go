import React, { useState } from 'react';
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  BarElement,
  Title,
  Tooltip,
  Legend,
  ChartOptions,
} from 'chart.js';
import { Line, Bar } from 'react-chartjs-2';
import { format } from 'date-fns';
import { Clock, BarChart3, Calendar, CalendarDays } from 'lucide-react';
import type { ChartsProps, TimeSeriesDataPoint } from '../types';

ChartJS.register(
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  BarElement,
  Title,
  Tooltip,
  Legend
);

const chartOptions: ChartOptions<'line' | 'bar'> = {
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


  // Process minute data (per minute for last hour)
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
        borderColor: 'rgb(59, 130, 246)',
        backgroundColor: 'rgba(59, 130, 246, 0.1)',
        fill: true,
        tension: 0.4,
      },
    ],
  };

  // Process hour data (per hour for last day)
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

  // Process day data (per day for last week)
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

  // Process week data (per day for last month)
  const weekData: TimeSeriesDataPoint[] = timeSeriesData.requests_last_month || [];
  const weekChartData = {
    labels: weekData.map((point: TimeSeriesDataPoint): string => {
      const date = new Date(point.timestamp);
      return format(date, 'MMM dd');
    }),
    datasets: [
      {
        label: 'Requests per Day',
        data: weekData.map((point: TimeSeriesDataPoint): number => point.value),
        borderColor: 'rgb(168, 85, 247)',
        backgroundColor: 'rgba(168, 85, 247, 0.1)',
        fill: true,
        tension: 0.4,
      },
    ],
  };

  // Determine current data, component, and title based on active view
  const getChartConfig = () => {
    switch (activeView) {
      case 'minute':
        return {
          data: minuteChartData,
          component: Line,
          title: 'Requests per Minute (Last Hour)'
        };
      case 'hour':
        return {
          data: hourChartData,
          component: Bar,
          title: 'Requests per Hour (Last Day)'
        };
      case 'day':
        return {
          data: dayChartData,
          component: Bar,
          title: 'Requests per Day (Last Week)'
        };
      case 'week':
        return {
          data: weekChartData,
          component: Line,
          title: 'Requests per Day (Last Month)'
        };
      default:
        return {
          data: hourChartData,
          component: Bar,
          title: 'Requests per Hour (Last Day)'
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
