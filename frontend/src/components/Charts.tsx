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
import { format, startOfWeek, endOfWeek } from 'date-fns';
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

  // Helper function to aggregate daily data into weekly buckets
  const aggregateByWeek = (data: TimeSeriesDataPoint[]): TimeSeriesDataPoint[] => {
    const weekMap = new Map<string, number>();
    
    data.forEach(point => {
      const date = new Date(point.timestamp);
      const weekStart = startOfWeek(date, { weekStartsOn: 1 }); // Monday as start of week
      const weekKey = format(weekStart, 'yyyy-MM-dd');
      weekMap.set(weekKey, (weekMap.get(weekKey) || 0) + point.value);
    });

    return Array.from(weekMap.entries()).map(([dateStr, value]) => ({
      timestamp: new Date(dateStr).toISOString(),
      value
    })).sort((a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime());
  };

  // Helper function to generate exactly 75 time slots and fill with data
  const generateTimeSlots = (
    data: TimeSeriesDataPoint[], 
    timeUnit: 'minute' | 'hour' | 'day' | 'week'
  ): TimeSeriesDataPoint[] => {
    const now = new Date();
    const slots: TimeSeriesDataPoint[] = [];
    const dataMap = new Map<string, number>();
    
    // Create a map of existing data
    data.forEach(point => {
      const date = new Date(point.timestamp);
      let key: string;
      
      switch (timeUnit) {
        case 'minute':
          key = format(date.getTime() - (date.getTime() % 60000), 'yyyy-MM-dd HH:mm');
          break;
        case 'hour':
          key = format(date.getTime() - (date.getTime() % 3600000), 'yyyy-MM-dd HH:00');
          break;
        case 'day':
          key = format(date, 'yyyy-MM-dd');
          break;
        case 'week':
          const weekStart = startOfWeek(date, { weekStartsOn: 1 });
          key = format(weekStart, 'yyyy-MM-dd');
          break;
        default:
          key = format(date, 'yyyy-MM-dd');
      }
      
      dataMap.set(key, (dataMap.get(key) || 0) + point.value);
    });
    
    // Generate 75 time slots going backwards from now
    for (let i = 74; i >= 0; i--) {
      let slotTime: Date;
      let key: string;
      
      switch (timeUnit) {
        case 'minute':
          slotTime = new Date(now.getTime() - (i * 60000));
          slotTime.setSeconds(0, 0);
          key = format(slotTime, 'yyyy-MM-dd HH:mm');
          break;
        case 'hour':
          slotTime = new Date(now.getTime() - (i * 3600000));
          slotTime.setMinutes(0, 0, 0);
          key = format(slotTime, 'yyyy-MM-dd HH:00');
          break;
        case 'day':
          slotTime = new Date(now.getTime() - (i * 24 * 3600000));
          slotTime.setHours(0, 0, 0, 0);
          key = format(slotTime, 'yyyy-MM-dd');
          break;
        case 'week':
          const weekAgo = new Date(now.getTime() - (i * 7 * 24 * 3600000));
          slotTime = startOfWeek(weekAgo, { weekStartsOn: 1 });
          key = format(slotTime, 'yyyy-MM-dd');
          break;
        default:
          slotTime = new Date(now.getTime() - (i * 24 * 3600000));
          slotTime.setHours(0, 0, 0, 0);
          key = format(slotTime, 'yyyy-MM-dd');
      }
      
      slots.push({
        timestamp: slotTime.toISOString(),
        value: dataMap.get(key) || 0
      });
    }
    
    return slots;
  };

  // Process minute data (per minute for last 75 minutes)
  const rawMinuteData: TimeSeriesDataPoint[] = timeSeriesData.requests_last_hour || [];
  const minuteData: TimeSeriesDataPoint[] = generateTimeSlots(rawMinuteData, 'minute');
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

  // Process hour data (per hour for last 75 hours)
  const rawHourData: TimeSeriesDataPoint[] = timeSeriesData.requests_last_day || [];
  const hourData: TimeSeriesDataPoint[] = generateTimeSlots(rawHourData, 'hour');
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

  // Process day data (per day for last 75 days)
  const rawDayData: TimeSeriesDataPoint[] = timeSeriesData.requests_last_week || [];
  const dayData: TimeSeriesDataPoint[] = generateTimeSlots(rawDayData, 'day');
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

  // Process week data (aggregate daily data into weekly buckets for last 75 weeks)
  const monthlyDailyData: TimeSeriesDataPoint[] = timeSeriesData.requests_last_month || [];
  const aggregatedWeekData: TimeSeriesDataPoint[] = aggregateByWeek(monthlyDailyData);
  const weekData: TimeSeriesDataPoint[] = generateTimeSlots(aggregatedWeekData, 'week');
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
