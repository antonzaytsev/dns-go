import React from 'react';
import { BrowserRouter as Router, Routes, Route } from 'react-router-dom';
import Dashboard from './components/Dashboard.tsx';
import DNSMappingsPage from './components/DNSMappingsPage.tsx';
import RecentRequestsPage from './components/RecentRequestsPage.tsx';
import NotFound from './components/NotFound';
import './App.css';

const App: React.FC = () => {
  return (
    <div className="App">
      <Router>
        <Routes>
          <Route path="/" element={<Dashboard />} />
          <Route path="/dns-mappings" element={<DNSMappingsPage />} />
          <Route path="/recent-requests" element={<RecentRequestsPage />} />
          <Route path="*" element={<NotFound />} />
        </Routes>
      </Router>
    </div>
  );
};

export default App;
