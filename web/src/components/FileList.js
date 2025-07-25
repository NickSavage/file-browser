import React, { useState } from 'react';
import { 
  FolderOpen, 
  File, 
  Download, 
  Trash2, 
  Edit3,
  Image,
  FileText,
  Archive,
  Video,
  Music,
  Code,
  Link
} from 'lucide-react';

const FileList = ({ files, viewMode, onNavigate, onFileAction, currentPath }) => {
  const [editingFile, setEditingFile] = useState(null);
  const [newName, setNewName] = useState('');

  const getFileIcon = (file) => {
    let baseIcon;
    
    if (file.isDir) {
      baseIcon = <FolderOpen className="w-5 h-5 text-blue-500" />;
    } else {
      const ext = file.extension.toLowerCase();
      if (['.jpg', '.jpeg', '.png', '.gif', '.bmp', '.svg'].includes(ext)) {
        baseIcon = <Image className="w-5 h-5 text-green-500" />;
      } else if (['.txt', '.md', '.doc', '.docx', '.pdf'].includes(ext)) {
        baseIcon = <FileText className="w-5 h-5 text-gray-500" />;
      } else if (['.zip', '.rar', '.7z', '.tar', '.gz'].includes(ext)) {
        baseIcon = <Archive className="w-5 h-5 text-orange-500" />;
      } else if (['.mp4', '.avi', '.mkv', '.mov', '.wmv'].includes(ext)) {
        baseIcon = <Video className="w-5 h-5 text-purple-500" />;
      } else if (['.mp3', '.wav', '.flac', '.aac', '.ogg'].includes(ext)) {
        baseIcon = <Music className="w-5 h-5 text-pink-500" />;
      } else if (['.js', '.ts', '.jsx', '.tsx', '.py', '.go', '.java', '.cpp', '.c', '.html', '.css'].includes(ext)) {
        baseIcon = <Code className="w-5 h-5 text-indigo-500" />;
      } else {
        baseIcon = <File className="w-5 h-5 text-gray-400" />;
      }
    }

    if (file.isSymlink) {
      return (
        <div className="relative inline-block">
          {baseIcon}
          <Link className="absolute -bottom-1 -right-1 w-3 h-3 text-blue-600 bg-white rounded-full p-0.5" />
        </div>
      );
    }

    return baseIcon;
  };

  const formatFileSize = (bytes) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  const formatDate = (date) => {
    return new Date(date).toLocaleDateString() + ' ' + new Date(date).toLocaleTimeString();
  };

  const handleDownload = (file) => {
    const downloadUrl = `/api/download/${file.relativePath}`;
    const link = document.createElement('a');
    link.href = downloadUrl;
    link.download = file.name;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
  };

  const handleRename = (file) => {
    setEditingFile(file.relativePath);
    setNewName(file.name);
  };

  const handleRenameSubmit = (file) => {
    if (newName && newName !== file.name) {
      onFileAction('rename', file.relativePath, { newName });
    }
    setEditingFile(null);
    setNewName('');
  };

  const handleDelete = (file) => {
    if (window.confirm(`Are you sure you want to delete "${file.name}"?`)) {
      onFileAction('delete', file.relativePath);
    }
  };

  const handleClick = (file) => {
    if (file.isDir) {
      onNavigate(file.relativePath);
    }
  };

  if (files.length === 0) {
    return (
      <div className="text-center py-12 text-gray-500">
        <FolderOpen className="w-12 h-12 mx-auto mb-4 opacity-50" />
        <p className="text-lg font-medium">This folder is empty</p>
        <p className="text-sm">Upload files or create a new folder to get started</p>
      </div>
    );
  }

  if (viewMode === 'grid') {
    return (
      <div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-6 gap-4">
        {files.map((file) => (
          <div
            key={file.name}
            className="bg-gray-50 rounded-lg p-4 hover:bg-gray-100 transition-colors group"
          >
            <div 
              className={`text-center ${file.isDir ? 'cursor-pointer' : ''}`}
              onClick={() => handleClick(file)}
            >
              <div className="flex justify-center mb-2">
                {getFileIcon(file)}
              </div>
              {editingFile === file.relativePath ? (
                <input
                  type="text"
                  value={newName}
                  onChange={(e) => setNewName(e.target.value)}
                  onBlur={() => handleRenameSubmit(file)}
                  onKeyPress={(e) => e.key === 'Enter' && handleRenameSubmit(file)}
                  className="text-sm text-center w-full bg-white border rounded px-1"
                  autoFocus
                />
              ) : (
                <div className="text-center">
                  <p className="text-sm font-medium text-gray-900 truncate" title={file.name}>
                    {file.name}
                  </p>
                  {file.isSymlink && file.linkTarget && (
                    <p className="text-xs text-blue-600 truncate mt-1" title={`→ ${file.linkTarget}`}>
                      → {file.linkTarget}
                    </p>
                  )}
                </div>
              )}
              {!file.isDir && (
                <p className="text-xs text-gray-500 mt-1">
                  {formatFileSize(file.size)}
                </p>
              )}
            </div>
            
            <div className="flex justify-center space-x-1 mt-2 opacity-0 group-hover:opacity-100 transition-opacity">
              {!file.isDir && (
                <button
                  onClick={() => handleDownload(file)}
                  className="p-1 text-gray-400 hover:text-blue-600 transition-colors"
                  title="Download"
                >
                  <Download className="w-4 h-4" />
                </button>
              )}
              <button
                onClick={() => handleRename(file)}
                className="p-1 text-gray-400 hover:text-yellow-600 transition-colors"
                title="Rename"
              >
                <Edit3 className="w-4 h-4" />
              </button>
              <button
                onClick={() => handleDelete(file)}
                className="p-1 text-gray-400 hover:text-red-600 transition-colors"
                title="Delete"
              >
                <Trash2 className="w-4 h-4" />
              </button>
            </div>
          </div>
        ))}
      </div>
    );
  }

  return (
    <div className="overflow-x-auto">
      <table className="min-w-full divide-y divide-gray-200">
        <thead className="bg-gray-50">
          <tr>
            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Name
            </th>
            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Size
            </th>
            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Modified
            </th>
            <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
              Actions
            </th>
          </tr>
        </thead>
        <tbody className="bg-white divide-y divide-gray-200">
          {files.map((file) => (
            <tr key={file.name} className="hover:bg-gray-50">
              <td className="px-6 py-4 whitespace-nowrap">
                <div className="flex items-center">
                  {getFileIcon(file)}
                  <div className="ml-3">
                    {editingFile === file.relativePath ? (
                      <input
                        type="text"
                        value={newName}
                        onChange={(e) => setNewName(e.target.value)}
                        onBlur={() => handleRenameSubmit(file)}
                        onKeyPress={(e) => e.key === 'Enter' && handleRenameSubmit(file)}
                        className="text-sm font-medium text-gray-900 bg-white border rounded px-2 py-1"
                        autoFocus
                      />
                    ) : (
                      <div>
                        <span
                          className={`text-sm font-medium text-gray-900 ${
                            file.isDir ? 'cursor-pointer hover:text-blue-600' : ''
                          }`}
                          onClick={() => handleClick(file)}
                        >
                          {file.name}
                        </span>
                        {file.isSymlink && file.linkTarget && (
                          <div className="text-xs text-blue-600 mt-1" title={`Symlink target: ${file.linkTarget}`}>
                            → {file.linkTarget}
                          </div>
                        )}
                      </div>
                    )}
                  </div>
                </div>
              </td>
              <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                {file.isDir ? '-' : formatFileSize(file.size)}
              </td>
              <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                {formatDate(file.modTime)}
              </td>
              <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                <div className="flex items-center justify-end space-x-2">
                  {!file.isDir && (
                    <button
                      onClick={() => handleDownload(file)}
                      className="text-blue-600 hover:text-blue-900 transition-colors"
                      title="Download"
                    >
                      <Download className="w-4 h-4" />
                    </button>
                  )}
                  <button
                    onClick={() => handleRename(file)}
                    className="text-yellow-600 hover:text-yellow-900 transition-colors"
                    title="Rename"
                  >
                    <Edit3 className="w-4 h-4" />
                  </button>
                  <button
                    onClick={() => handleDelete(file)}
                    className="text-red-600 hover:text-red-900 transition-colors"
                    title="Delete"
                  >
                    <Trash2 className="w-4 h-4" />
                  </button>
                </div>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
};

export default FileList;