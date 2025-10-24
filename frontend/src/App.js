import React from 'react';
import { BrowserRouter as Router, Routes, Route } from 'react-router-dom';
import Dashboard from './components/Dashboard';
import DNSMappingsPage from './components/DNSMappingsPage';
import RecentRequestsPage from './components/RecentRequestsPage';
import NotFound from './components/NotFound';
import './App.css';

function App() {
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
}

export default App;
