import React, { useEffect, useRef, useState } from 'react';
import './Editor.css';

interface EditorProps {
  filePath: string;
}

export const Editor: React.FC<EditorProps> = ({ filePath }) => {
  const [content, setContent] = useState('');
  const [isLoading, setIsLoading] = useState(true);
  const editorRef = useRef<HTMLTextAreaElement>(null);
  const [lineCount, setLineCount] = useState(1);

  useEffect(() => {
    loadFile(filePath);
  }, [filePath]);

  const loadFile = async (path: string) => {
    setIsLoading(true);
    try {
      const response = await fetch(`/api/v1/editor/files/get?path=${encodeURIComponent(path)}`);
      if (response.ok) {
        const data = await response.json();
        setContent(data.content || '');
        setLineCount((data.content || '').split('\n').length);
      } else {
        // Fallback: mock content
        const mockContent = `// File: ${path}\n\n// Mock content for development\nfunction example() {\n  console.log('This is a mock file');\n}\n`;
        setContent(mockContent);
        setLineCount(mockContent.split('\n').length);
      }
    } catch (error) {
      console.error('Failed to load file:', error);
    } finally {
      setIsLoading(false);
    }
  };

  const handleContentChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    const newContent = e.target.value;
    setContent(newContent);
    setLineCount(newContent.split('\n').length);
  };

  const saveFile = async () => {
    try {
      await fetch(`/api/v1/editor/files/update`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ path: filePath, content }),
      });
      console.log('File saved');
    } catch (error) {
      console.error('Failed to save file:', error);
    }
  };

  const getLanguage = (path: string) => {
    const ext = path.split('.').pop()?.toLowerCase();
    const languages: { [key: string]: string } = {
      go: 'Go',
      ts: 'TypeScript',
      tsx: 'TypeScript React',
      js: 'JavaScript',
      jsx: 'JavaScript React',
      md: 'Markdown',
      json: 'JSON',
      yaml: 'YAML',
    };
    return languages[ext || ''] || 'Plain Text';
  };

  return (
    <div className="editor-container-new">
      <div className="editor-header-new">
        <div className="editor-info-new">
          <span className="file-name-new">{filePath.split('/').pop()}</span>
          <span className="file-lang-new">{getLanguage(filePath)}</span>
          <span className="file-stats-new">{lineCount} lines</span>
        </div>
        <div className="editor-actions-new">
          <button 
            className="editor-btn-new"
            onClick={saveFile}
            title="Save (Ctrl+S)"
          >
            💾 Save
          </button>
          <button 
            className="editor-btn-new"
            title="Format (Shift+Alt+F)"
          >
            🔧 Format
          </button>
        </div>
      </div>

      <div className="editor-wrapper-new">
        <div className="line-numbers-new">
          {Array.from({ length: lineCount }, (_, i) => (
            <div key={i + 1} className="line-number-new">
              {i + 1}
            </div>
          ))}
        </div>

        <textarea
          ref={editorRef}
          className="editor-textarea-new"
          value={content}
          onChange={handleContentChange}
          spellCheck={false}
          disabled={isLoading}
        />
      </div>

      <div className="editor-status-new">
        <div className="status-item-new">
          <span>Ln {content.split('\n').length}, Col 1</span>
        </div>
        <div className="status-item-new">
          <span>{getLanguage(filePath)}</span>
        </div>
        <div className="status-item-new">
          <span>UTF-8</span>
        </div>
      </div>
    </div>
  );
};
