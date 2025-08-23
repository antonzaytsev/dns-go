import React from 'react';

const QueryTypes = ({ queryTypes }) => {
  if (!queryTypes || Object.keys(queryTypes).length === 0) {
    return (
      <div className="bg-white rounded-lg shadow-md p-6">
        <h3 className="text-lg font-semibold text-gray-900 mb-4">Query Types</h3>
        <div className="text-center text-gray-500 py-8">
          No query data available
        </div>
      </div>
    );
  }

  const formatNumber = (num) => {
    if (num >= 1000000) {
      return (num / 1000000).toFixed(1) + 'M';
    } else if (num >= 1000) {
      return (num / 1000).toFixed(1) + 'K';
    }
    return num.toString();
  };

  const sortedTypes = Object.entries(queryTypes)
    .sort(([, a], [, b]) => b - a)
    .slice(0, 8); // Show top 8 query types

  const colors = [
    'bg-blue-100 border-blue-300 text-blue-800',
    'bg-green-100 border-green-300 text-green-800',
    'bg-purple-100 border-purple-300 text-purple-800',
    'bg-orange-100 border-orange-300 text-orange-800',
    'bg-pink-100 border-pink-300 text-pink-800',
    'bg-indigo-100 border-indigo-300 text-indigo-800',
    'bg-yellow-100 border-yellow-300 text-yellow-800',
    'bg-red-100 border-red-300 text-red-800',
  ];

  return (
    <div className="bg-white rounded-lg shadow-md p-6">
      <h3 className="text-lg font-semibold text-gray-900 mb-4">Query Types</h3>
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        {sortedTypes.map(([type, count], index) => (
          <div
            key={type}
            className={`p-4 rounded-lg border-2 text-center transition-transform hover:scale-105 ${
              colors[index % colors.length]
            }`}
          >
            <div className="font-semibold text-sm mb-1">{type}</div>
            <div className="text-xl font-bold">{formatNumber(count)}</div>
          </div>
        ))}
      </div>
    </div>
  );
};

export default QueryTypes;
