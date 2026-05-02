import React from 'react';
import './TaskMonitor.css';

interface Task {
  id: string;
  name: string;
  status: 'pending' | 'running' | 'completed' | 'failed';
  progress: number;
  output?: string;
  startTime: number;
  endTime?: number;
}

interface TaskMonitorProps {
  tasks: Task[];
}

export const TaskMonitor: React.FC<TaskMonitorProps> = ({ tasks }) => {
  const getStatusIcon = (status: Task['status']) => {
    switch (status) {
      case 'running':
        return '⏳';
      case 'completed':
        return '✅';
      case 'failed':
        return '❌';
      case 'pending':
        return '⏰';
    }
  };

  const getStatusColor = (status: Task['status']) => {
    switch (status) {
      case 'running':
        return '#fbbf24';
      case 'completed':
        return '#4ade80';
      case 'failed':
        return '#f87171';
      case 'pending':
        return '#60a5fa';
    }
  };

  const formatTime = (ms: number) => {
    const seconds = Math.floor(ms / 1000);
    if (seconds < 60) return `${seconds}s`;
    const minutes = Math.floor(seconds / 60);
    return `${minutes}m ${seconds % 60}s`;
  };

  if (tasks.length === 0) {
    return null;
  }

  return (
    <div className="task-monitor-container">
      <div className="task-monitor-header">
        <h4>📊 Task Monitor</h4>
        <span className="task-count">{tasks.length}</span>
      </div>

      <div className="task-list">
        {tasks.map(task => {
          const duration = task.endTime 
            ? task.endTime - task.startTime
            : Date.now() - task.startTime;

          return (
            <div key={task.id} className="task-item">
              <div className="task-header">
                <span className="task-icon" title={task.status}>
                  {getStatusIcon(task.status)}
                </span>
                <span className="task-name">{task.name}</span>
                <span className="task-time">{formatTime(duration)}</span>
              </div>

              <div className="task-progress">
                <div className="progress-bar">
                  <div
                    className="progress-fill"
                    style={{
                      width: `${task.progress}%`,
                      backgroundColor: getStatusColor(task.status),
                    }}
                  />
                </div>
                <span className="progress-text">{task.progress}%</span>
              </div>

              {task.output && (
                <div className="task-output">
                  <pre>{task.output}</pre>
                </div>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
};
