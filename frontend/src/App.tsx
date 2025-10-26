import React from 'react';
import { BrowserRouter as Router, Routes, Route } from 'react-router-dom';
import DashboardPage from './pages/DashboardPage.tsx';
import DNSMappingsPage from './pages/DNSMappingsPage.tsx';
import RecentRequestsPage from './pages/RecentRequestsPage.tsx';
import ActiveClientsPage from './pages/ActiveClientsPage.tsx';
import NotFoundPage from './pages/NotFoundPage.tsx';
import './App.css';

const App: React.FC = () => {
  return (
    <div className="App">
      <Router>
        <Routes>
          <Route path="/" element={<DashboardPage />} />
          <Route path="/dns-mappings" element={<DNSMappingsPage />} />
          <Route path="/recent-requests" element={<RecentRequestsPage />} />
          <Route path="/clients" element={<ActiveClientsPage />} />
          <Route path="*" element={<NotFoundPage />} />
        </Routes>
      </Router>
    </div>
  );
};

export default App;
