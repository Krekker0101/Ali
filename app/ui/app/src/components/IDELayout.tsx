import React, { useState, useCallback } from 'react';
import './IDELayout.css';
import { FileExplorer } from './FileExplorer';
import { Editor } from './Editor';
import { Chat } from './Chat';
import { ProviderSelector } from './ProviderSelector';
import { TaskMonitor } from './TaskMonitor';

/**
 * Интегрированный IDE компонент
 * Объединяет: файловый браузер + редактор + AI чат + провайдер выбор
 * Макет: слева - файлы, центр - редактор, справа - чат
 */
export const IDELayout: React.FC = () => {
  const [selectedFile, setSelectedFile] = useState<string | null>(null);
  const [openFiles, setOpenFiles] = useState<string[]>([]);
  const [provider, setProvider] = useState<'local' | 'cloud'>('local');
  const [model, setModel] = useState<string>('');
  const [isAgentRunning, setIsAgentRunning] = useState(false);
  const [taskHistory, setTaskHistory] = useState<any[]>([]);

  const handleFileSelect = useCallback((filePath: string) => {
    setSelectedFile(filePath);
    if (!openFiles.includes(filePath)) {
      setOpenFiles([...openFiles, filePath]);
    }
  }, [openFiles]);

  const handleFileClose = useCallback((filePath: string) => {
    setOpenFiles(openFiles.filter(f => f !== filePath));
    if (selectedFile === filePath) {
      setSelectedFile(openFiles[0] || null);
    }
  }, [openFiles, selectedFile]);

  const handleAgentTask = useCallback(async (prompt: string) => {
    if (!model) {
      alert('Выберите модель');
      return;
    }

    setIsAgentRunning(true);
    try {
      const response = await fetch('/api/v1/agents/task', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          provider,
          model,
          prompt,
          context: {
            selectedFile,
            projectRoot: '.',
          },
        }),
      });

      const data = await response.json();
      setTaskHistory([...taskHistory, data]);
    } catch (error) {
      console.error('Agent error:', error);
    } finally {
      setIsAgentRunning(false);
    }
  }, [provider, model, selectedFile, taskHistory]);

  return (
    <div className="ide-layout">
      {/* Header с провайдером и выбором модели */}
      <div className="ide-header">
        <h1>Ali IDE - AI Code Assistant</h1>
        <ProviderSelector
          provider={provider}
          onProviderChange={setProvider}
          model={model}
          onModelChange={setModel}
        />
      </div>

      <div className="ide-container">
        {/* Левая панель - файловый браузер */}
        <div className="ide-sidebar-left">
          <div className="sidebar-header">
            <h3>📁 Files</h3>
          </div>
          <FileExplorer 
            selectedFile={selectedFile}
            onSelectFile={handleFileSelect}
          />
        </div>

        {/* Центральная часть - редактор */}
        <div className="ide-center">
          <div className="editor-tabs">
            {openFiles.map(file => (
              <div 
                key={file}
                className={`tab ${selectedFile === file ? 'active' : ''}`}
                onClick={() => setSelectedFile(file)}
              >
                <span>{file.split('/').pop()}</span>
                <button 
                  className="tab-close"
                  onClick={(e) => {
                    e.stopPropagation();
                    handleFileClose(file);
                  }}
                >
                  ×
                </button>
              </div>
            ))}
          </div>
          
          {selectedFile ? (
            <Editor filePath={selectedFile} />
          ) : (
            <div className="editor-empty">
              <p>Выберите файл для редактирования</p>
            </div>
          )}
        </div>

        {/* Правая панель - чат и задачи */}
        <div className="ide-sidebar-right">
          <div className="sidebar-tabs">
            <button className="sidebar-tab active">💬 Chat</button>
            <button className="sidebar-tab">📊 Tasks</button>
          </div>

          <Chat 
            onSendMessage={handleAgentTask}
            isLoading={isAgentRunning}
            provider={provider}
            model={model}
          />

          {taskHistory.length > 0 && (
            <TaskMonitor tasks={taskHistory} />
          )}
        </div>
      </div>
    </div>
  );
};
