import React from 'react';
import { BrowserRouter as Router, Routes, Route } from 'react-router-dom';
import DashboardPage from './pages/DashboardPage.tsx';
import DNSMappingsPage from './pages/DNSMappingsPage.tsx';
import RequestsPage from './pages/RequestsPage.tsx';
import ClientsPage from './pages/ClientsPage.tsx';
import DomainsPage from './pages/DomainsPage.tsx';
import NotFoundPage from './pages/NotFoundPage.tsx';
import './App.css';

const App: React.FC = () => {
  return (
    <div className="App">
      <Router>
        <Routes>
          <Route path="/" element={<DashboardPage />} />
          <Route path="/dns-mappings" element={<DNSMappingsPage />} />
          <Route path="/requests" element={<RequestsPage />} />
          <Route path="/clients" element={<ClientsPage />} />
          <Route path="/domains" element={<DomainsPage />} />
          <Route path="*" element={<NotFoundPage />} />
        </Routes>
      </Router>
    </div>
  );
};

export default App;
