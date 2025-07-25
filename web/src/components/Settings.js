import React, { useState } from 'react';
import { Settings as SettingsIcon, Users, RefreshCw, Shield, ArrowLeft, Home } from 'lucide-react';
import { Link } from 'react-router-dom';
import UserManagement from './UserManagement';
import { useAuth } from '../contexts/AuthContext';

const Settings = () => {
  const [activeTab, setActiveTab] = useState('users');
  const [rebuildLoading, setRebuildLoading] = useState(false);
  const { apiRequest } = useAuth();

  const rebuildIndex = async () => {
    setRebuildLoading(true);
    try {
      await apiRequest('/api/index/rebuild', { method: 'POST' });
      alert('Index rebuilt successfully!');
    } catch (error) {
      console.error('Failed to rebuild index:', error);
      alert('Failed to rebuild index');
    } finally {
      setRebuildLoading(false);
    }
  };

  const tabs = [
    { id: 'users', label: 'User Management', icon: Users },
    { id: 'system', label: 'System', icon: SettingsIcon },
  ];

  return (
    <div className="max-w-6xl mx-auto">
      {/* Breadcrumb Navigation */}
      <nav className="flex items-center space-x-2 text-sm text-gray-500 mb-4">
        <Link to="/browse/" className="flex items-center hover:text-gray-700">
          <Home className="h-4 w-4 mr-1" />
          File Browser
        </Link>
        <span>/</span>
        <span className="text-gray-900 font-medium">Settings</span>
      </nav>

      <div className="bg-white shadow rounded-lg">
        <div className="px-4 py-5 sm:p-6">
          <div className="flex items-center justify-between mb-6">
            <div className="flex items-center">
              <Shield className="h-6 w-6 text-purple-600 mr-2" />
              <h1 className="text-2xl font-bold text-gray-900">Admin Settings</h1>
            </div>
            <Link
              to="/browse/"
              className="flex items-center space-x-2 px-3 py-2 text-gray-600 hover:text-gray-900 border border-gray-300 rounded-md hover:border-gray-400 transition-colors"
            >
              <ArrowLeft className="h-4 w-4" />
              <span>Back to Files</span>
            </Link>
          </div>

          {/* Tab Navigation */}
          <div className="border-b border-gray-200 mb-6">
            <nav className="-mb-px flex space-x-8">
              {tabs.map((tab) => {
                const Icon = tab.icon;
                return (
                  <button
                    key={tab.id}
                    onClick={() => setActiveTab(tab.id)}
                    className={`flex items-center py-2 px-1 border-b-2 font-medium text-sm ${
                      activeTab === tab.id
                        ? 'border-purple-500 text-purple-600'
                        : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
                    }`}
                  >
                    <Icon className="h-4 w-4 mr-2" />
                    {tab.label}
                  </button>
                );
              })}
            </nav>
          </div>

          {/* Tab Content */}
          <div className="mt-6">
            {activeTab === 'users' && (
              <div>
                <UserManagement />
              </div>
            )}

            {activeTab === 'system' && (
              <div className="space-y-6">
                <div className="bg-gray-50 rounded-lg p-6">
                  <h3 className="text-lg font-medium text-gray-900 mb-2">File Index</h3>
                  <p className="text-sm text-gray-600 mb-4">
                    Rebuild the file index to update the directory structure and file information.
                    This may take some time for large directories.
                  </p>
                  <button
                    onClick={rebuildIndex}
                    disabled={rebuildLoading}
                    className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    {rebuildLoading ? (
                      <>
                        <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white mr-2"></div>
                        Rebuilding...
                      </>
                    ) : (
                      <>
                        <RefreshCw className="h-4 w-4 mr-2" />
                        Rebuild Index
                      </>
                    )}
                  </button>
                </div>

                <div className="bg-gray-50 rounded-lg p-6">
                  <h3 className="text-lg font-medium text-gray-900 mb-2">Authentication</h3>
                  <p className="text-sm text-gray-600 mb-4">
                    Authentication settings and security configuration.
                  </p>
                  <div className="space-y-3">
                    <div className="flex justify-between items-center py-2">
                      <span className="text-sm font-medium text-gray-900">JWT Secret</span>
                      <span className="text-sm text-gray-500">
                        {process.env.REACT_APP_JWT_SECRET ? 'Custom' : 'Default (change in production)'}
                      </span>
                    </div>
                    <div className="flex justify-between items-center py-2">
                      <span className="text-sm font-medium text-gray-900">Database</span>
                      <span className="text-sm text-gray-500">SQLite</span>
                    </div>
                  </div>
                </div>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};

export default Settings;