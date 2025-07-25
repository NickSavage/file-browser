import React, { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';
import { 
  FolderOpen, 
  File, 
  Download, 
  Upload, 
  Plus, 
  Trash2, 
  Edit3,
  ChevronRight,
  Home,
  Search,
  Grid,
  List
} from 'lucide-react';
import FileUpload from './FileUpload';
import FileList from './FileList';
import Breadcrumb from './Breadcrumb';
import SearchBar from './SearchBar';

const FileBrowser = () => {
  const { '*': currentPath } = useParams();
  const navigate = useNavigate();
  const { apiRequest } = useAuth();
  const [files, setFiles] = useState([]);
  const [loading, setLoading] = useState(true);
  const [viewMode, setViewMode] = useState('list');
  const [showUpload, setShowUpload] = useState(false);
  const [searchTerm, setSearchTerm] = useState('');

  const path = currentPath || '';

  useEffect(() => {
    fetchFiles();
  }, [path]);

  const fetchFiles = async () => {
    setLoading(true);
    try {
      const response = await apiRequest(`/api/browse/${path}`);
      const data = await response.json();
      setFiles(data.files || []);
    } catch (error) {
      console.error('Failed to fetch files:', error);
      setFiles([]);
    } finally {
      setLoading(false);
    }
  };

  const handleNavigate = (newPath) => {
    navigate(`/browse/${newPath}`);
  };

  const handleFileAction = async (action, filePath, data = {}) => {
    try {
      let response;
      
      switch (action) {
        case 'delete':
          response = await apiRequest(`/api/delete/${filePath}`, { method: 'DELETE' });
          break;
        case 'rename':
          response = await apiRequest(`/api/rename/${filePath}`, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ newName: data.newName })
          });
          break;
        case 'mkdir':
          response = await apiRequest(`/api/mkdir/${path}`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ name: data.name })
          });
          break;
        default:
          return;
      }

      if (response.ok) {
        await fetchFiles();
      } else {
        const error = await response.json();
        alert(`Error: ${error.error}`);
      }
    } catch (error) {
      console.error(`Failed to ${action}:`, error);
      alert(`Failed to ${action}`);
    }
  };

  const handleCreateFolder = () => {
    const name = prompt('Enter folder name:');
    if (name && name.trim()) {
      handleFileAction('mkdir', '', { name: name.trim() });
    }
  };

  const filteredFiles = files.filter(file =>
    file.name.toLowerCase().includes(searchTerm.toLowerCase())
  );

  if (loading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="bg-white rounded-lg shadow-sm p-6">
        <div className="flex items-center justify-between mb-4">
          <Breadcrumb path={path} onNavigate={handleNavigate} />
          
          <div className="flex items-center space-x-2">
            <SearchBar searchTerm={searchTerm} onSearchChange={setSearchTerm} />
            
            <div className="flex items-center border rounded-lg">
              <button
                onClick={() => setViewMode('list')}
                className={`p-2 ${viewMode === 'list' ? 'bg-blue-100 text-blue-600' : 'text-gray-400'}`}
              >
                <List className="w-4 h-4" />
              </button>
              <button
                onClick={() => setViewMode('grid')}
                className={`p-2 ${viewMode === 'grid' ? 'bg-blue-100 text-blue-600' : 'text-gray-400'}`}
              >
                <Grid className="w-4 h-4" />
              </button>
            </div>

            <button
              onClick={() => setShowUpload(true)}
              className="flex items-center space-x-2 px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 transition-colors"
            >
              <Upload className="w-4 h-4" />
              <span>Upload</span>
            </button>

            <button
              onClick={handleCreateFolder}
              className="flex items-center space-x-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
            >
              <Plus className="w-4 h-4" />
              <span>New Folder</span>
            </button>
          </div>
        </div>

        <FileList
          files={filteredFiles}
          viewMode={viewMode}
          onNavigate={handleNavigate}
          onFileAction={handleFileAction}
          currentPath={path}
        />
      </div>

      {showUpload && (
        <FileUpload
          path={path}
          onClose={() => setShowUpload(false)}
          onUploadComplete={fetchFiles}
        />
      )}
    </div>
  );
};

export default FileBrowser;