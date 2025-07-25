import React, { useState, useEffect } from 'react';
import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import FileBrowser from './components/FileBrowser';
import Header from './components/Header';

function App() {
  const [index, setIndex] = useState(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchIndex();
  }, []);

  const fetchIndex = async () => {
    try {
      const response = await fetch('/api/index');
      const data = await response.json();
      setIndex(data);
    } catch (error) {
      console.error('Failed to fetch index:', error);
    } finally {
      setLoading(false);
    }
  };

  const rebuildIndex = async () => {
    try {
      await fetch('/api/index/rebuild', { method: 'POST' });
      await fetchIndex();
    } catch (error) {
      console.error('Failed to rebuild index:', error);
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
    <Router>
      <div className="min-h-screen bg-gray-50">
        <Header index={index} onRebuildIndex={rebuildIndex} />
        <main className="container mx-auto px-4 py-6">
          <Routes>
            <Route path="/browse/*" element={<FileBrowser />} />
            <Route path="/" element={<Navigate to="/browse/" replace />} />
          </Routes>
        </main>
      </div>
    </Router>
  );
}

export default App;