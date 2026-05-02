import React, { useState, useEffect } from 'react';
import './FileExplorer.css';

interface FileExplorerProps {
  selectedFile: string | null;
  onSelectFile: (filePath: string) => void;
}

interface FileNode {
  name: string;
  path: string;
  type: 'file' | 'folder';
  children?: FileNode[];
  expanded?: boolean;
}

const DEFAULT_FILES: FileNode[] = [
  {
    name: 'src',
    path: './src',
    type: 'folder',
    expanded: true,
    children: [
      { name: 'main.go', path: './src/main.go', type: 'file' },
      { name: 'server.go', path: './src/server.go', type: 'file' },
      { name: 'handler.go', path: './src/handler.go', type: 'file' },
    ],
  },
  {
    name: 'tests',
    path: './tests',
    type: 'folder',
    children: [
      { name: 'main_test.go', path: './tests/main_test.go', type: 'file' },
    ],
  },
  {
    name: 'go.mod',
    path: './go.mod',
    type: 'file',
  },
  {
    name: 'README.md',
    path: './README.md',
    type: 'file',
  },
];

export const FileExplorer: React.FC<FileExplorerProps> = ({
  selectedFile,
  onSelectFile,
}) => {
  const [files, setFiles] = useState<FileNode[]>(DEFAULT_FILES);

  const toggleFolder = (path: string) => {
    const updateFiles = (items: FileNode[]): FileNode[] => {
      return items.map(item => {
        if (item.path === path && item.type === 'folder') {
          return { ...item, expanded: !item.expanded };
        }
        if (item.children) {
          return { ...item, children: updateFiles(item.children) };
        }
        return item;
      });
    };
    setFiles(updateFiles(files));
  };

  const getFileIcon = (name: string, type: string) => {
    if (type === 'folder') return '📁';
    const ext = name.split('.').pop()?.toLowerCase();
    switch (ext) {
      case 'go':
        return '🐹';
      case 'ts':
      case 'tsx':
        return '📘';
      case 'js':
      case 'jsx':
        return '📙';
      case 'md':
        return '📝';
      case 'json':
        return '⚙️';
      case 'yaml':
      case 'yml':
        return '📋';
      default:
        return '📄';
    }
  };

  const renderTree = (items: FileNode[], depth = 0) => {
    return items.map(item => (
      <div key={item.path}>
        <div
          className={`file-tree-item ${selectedFile === item.path ? 'selected' : ''}`}
          style={{ paddingLeft: `${depth * 16}px` }}
          onClick={() => {
            if (item.type === 'folder') {
              toggleFolder(item.path);
            } else {
              onSelectFile(item.path);
            }
          }}
        >
          <span className="file-tree-item-icon">
            {item.type === 'folder' ? (
              item.expanded ? '▼' : '▶'
            ) : (
              getFileIcon(item.name, item.type)
            )}
          </span>
          <span className="file-tree-item-name">{item.name}</span>
        </div>

        {item.children && item.expanded && (
          renderTree(item.children, depth + 1)
        )}
      </div>
    ));
  };

  return (
    <div className="file-explorer">
      {renderTree(files)}
    </div>
  );
};
