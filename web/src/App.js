import React, { useState, useEffect } from 'react';
import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import FileBrowser from './components/FileBrowser';
import Header from './components/Header';
import Login from './components/Login';
import Settings from './components/Settings';
import ProtectedRoute from './components/ProtectedRoute';
import { AuthProvider, useAuth } from './contexts/AuthContext';

const AuthenticatedApp = () => {
  const [index, setIndex] = useState(null);
  const [loading, setLoading] = useState(true);
  const { apiRequest } = useAuth();

  useEffect(() => {
    fetchIndex();
  }, []);

  const fetchIndex = async () => {
    try {
      const response = await apiRequest('/api/index');
      const data = await response.json();
      setIndex(data);
    } catch (error) {
      console.error('Failed to fetch index:', error);
    } finally {
      setLoading(false);
    }
  };


  if (loading) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto"></div>
          <p className="mt-4 text-gray-600">Loading file browser...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-50">
      <Header index={index} />
      <main className="container mx-auto px-4 py-6">
        <Routes>
          <Route 
            path="/browse/*" 
            element={
              <ProtectedRoute>
                <FileBrowser />
              </ProtectedRoute>
            } 
          />
          <Route 
            path="/settings" 
            element={
              <ProtectedRoute requireAdmin={true}>
                <Settings />
              </ProtectedRoute>
            } 
          />
          <Route path="/" element={<Navigate to="/browse/" replace />} />
        </Routes>
      </main>
    </div>
  );
};

function App() {
  return (
    <AuthProvider>
      <Router>
        <Routes>
          <Route path="/login" element={<Login />} />
          <Route path="/*" element={<AuthenticatedApp />} />
        </Routes>
      </Router>
    </AuthProvider>
  );
}

export default App;