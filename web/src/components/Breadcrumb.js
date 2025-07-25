import React from 'react';
import { ChevronRight, Home } from 'lucide-react';

const Breadcrumb = ({ path, onNavigate }) => {
  const pathParts = path ? path.split('/').filter(Boolean) : [];

  return (
    <nav className="flex items-center space-x-2 text-sm">
      <button
        onClick={() => onNavigate('')}
        className="flex items-center space-x-1 text-blue-600 hover:text-blue-800 transition-colors"
      >
        <Home className="w-4 h-4" />
        <span>Home</span>
      </button>
      
      {pathParts.map((part, index) => {
        const partPath = pathParts.slice(0, index + 1).join('/');
        const isLast = index === pathParts.length - 1;
        
        return (
          <React.Fragment key={index}>
            <ChevronRight className="w-4 h-4 text-gray-400" />
            {isLast ? (
              <span className="text-gray-900 font-medium">{part}</span>
            ) : (
              <button
                onClick={() => onNavigate(partPath)}
                className="text-blue-600 hover:text-blue-800 transition-colors"
              >
                {part}
              </button>
            )}
          </React.Fragment>
        );
      })}
    </nav>
  );
};

export default Breadcrumb;