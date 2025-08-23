import React from 'react';
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
} from 'chart.js';
import { Line, Bar } from 'react-chartjs-2';
import { format } from 'date-fns';

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

const chartOptions = {
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

const Charts = ({ timeSeriesData }) => {
  if (!timeSeriesData) {
    return (
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="bg-white rounded-lg shadow-md p-6">
          <h3 className="text-lg font-semibold text-gray-900 mb-4">
            Requests per Minute (Last Hour)
          </h3>
          <div className="h-64 flex items-center justify-center">
            <div className="animate-pulse text-gray-400">Loading chart...</div>
          </div>
        </div>
        <div className="bg-white rounded-lg shadow-md p-6">
          <h3 className="text-lg font-semibold text-gray-900 mb-4">
            Requests per Hour (Last Day)
          </h3>
          <div className="h-64 flex items-center justify-center">
            <div className="animate-pulse text-gray-400">Loading chart...</div>
          </div>
        </div>
      </div>
    );
  }

  // Process hourly data
  const hourlyData = timeSeriesData.requests_last_hour || [];
  const hourlyChartData = {
    labels: hourlyData.map(point => {
      const date = new Date(point.timestamp);
      return format(date, 'HH:mm');
    }),
    datasets: [
      {
        label: 'Requests per Minute',
        data: hourlyData.map(point => point.value),
        borderColor: 'rgb(59, 130, 246)',
        backgroundColor: 'rgba(59, 130, 246, 0.1)',
        fill: true,
        tension: 0.4,
      },
    ],
  };

  // Process daily data
  const dailyData = timeSeriesData.requests_last_day || [];
  const dailyChartData = {
    labels: dailyData.map(point => {
      const date = new Date(point.timestamp);
      return format(date, 'MMM dd HH:mm');
    }),
    datasets: [
      {
        label: 'Requests per Hour',
        data: dailyData.map(point => point.value),
        backgroundColor: 'rgba(99, 102, 241, 0.8)',
        borderColor: 'rgb(99, 102, 241)',
        borderWidth: 1,
      },
    ],
  };

  return (
    <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
      <div className="bg-white rounded-lg shadow-md p-6">
        <h3 className="text-lg font-semibold text-gray-900 mb-4">
          Requests per Minute (Last Hour)
        </h3>
        <div className="h-64">
          <Line data={hourlyChartData} options={chartOptions} />
        </div>
      </div>
      
      <div className="bg-white rounded-lg shadow-md p-6">
        <h3 className="text-lg font-semibold text-gray-900 mb-4">
          Requests per Hour (Last Day)
        </h3>
        <div className="h-64">
          <Bar data={dailyChartData} options={chartOptions} />
        </div>
      </div>
    </div>
  );
};

export default Charts;
