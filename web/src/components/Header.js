import React from 'react';
import { RefreshCw, HardDrive, FileText, FolderOpen } from 'lucide-react';

const Header = ({ index, onRebuildIndex }) => {
  const formatFileSize = (bytes) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  const formatDate = (date) => {
    return new Date(date).toLocaleString();
  };

  return (
    <header className="bg-white shadow-sm border-b">
      <div className="container mx-auto px-4 py-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-4">
            <HardDrive className="w-8 h-8 text-blue-600" />
            <div>
              <h1 className="text-2xl font-bold text-gray-900">File Browser</h1>
              {index && (
                <p className="text-sm text-gray-500">
                  Last indexed: {formatDate(index.lastIndexed)}
                </p>
              )}
            </div>
          </div>
          
          <div className="flex items-center space-x-6">
            {index && (
              <div className="flex items-center space-x-4 text-sm text-gray-600">
                <div className="flex items-center space-x-1">
                  <FileText className="w-4 h-4" />
                  <span>{index.totalFiles.toLocaleString()} files</span>
                </div>
                <div className="flex items-center space-x-1">
                  <FolderOpen className="w-4 h-4" />
                  <span>{index.directories.length.toLocaleString()} folders</span>
                </div>
                <div className="flex items-center space-x-1">
                  <HardDrive className="w-4 h-4" />
                  <span>{formatFileSize(index.totalSize)}</span>
                </div>
              </div>
            )}
            
            <button
              onClick={onRebuildIndex}
              className="flex items-center space-x-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
            >
              <RefreshCw className="w-4 h-4" />
              <span>Rebuild Index</span>
            </button>
          </div>
        </div>
      </div>
    </header>
  );
};

export default Header;