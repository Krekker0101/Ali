import { useEffect, useRef, useState, useCallback } from 'react';

interface WebSocketMessage {
  type: 'chat' | 'task-update' | 'task-complete' | 'error';
  data: any;
  timestamp: number;
}

export const useWebSocket = (url: string) => {
  const [connected, setConnected] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const messageHandlersRef = useRef<Map<string, (data: any) => void>>(new Map());

  useEffect(() => {
    const connectWebSocket = () => {
      try {
        // Replace http/https with ws/wss
        const wsUrl = url.replace(/^http/, 'ws');
        const ws = new WebSocket(wsUrl);

        ws.onopen = () => {
          setConnected(true);
          setError(null);
          console.log('WebSocket connected');
        };

        ws.onmessage = (event) => {
          try {
            const message: WebSocketMessage = JSON.parse(event.data);
            const handlers = messageHandlersRef.current.get(message.type);
            if (handlers) {
              handlers(message.data);
            }
          } catch (err) {
            console.error('Failed to parse WebSocket message:', err);
          }
        };

        ws.onerror = (event) => {
          setError('WebSocket error');
          console.error('WebSocket error:', event);
        };

        ws.onclose = () => {
          setConnected(false);
          // Attempt to reconnect after 3 seconds
          setTimeout(() => {
            console.log('Attempting to reconnect...');
            connectWebSocket();
          }, 3000);
        };

        wsRef.current = ws;
      } catch (err) {
        setError('Failed to connect to WebSocket');
        console.error('WebSocket connection error:', err);
      }
    };

    connectWebSocket();

    return () => {
      if (wsRef.current) {
        wsRef.current.close();
      }
    };
  }, [url]);

  const send = useCallback((type: string, data: any) => {
    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
      wsRef.current.send(
        JSON.stringify({
          type,
          data,
          timestamp: Date.now(),
        })
      );
    } else {
      console.warn('WebSocket is not connected');
    }
  }, []);

  const on = useCallback((type: string, handler: (data: any) => void) => {
    messageHandlersRef.current.set(type, handler);
    return () => {
      messageHandlersRef.current.delete(type);
    };
  }, []);

  return { connected, error, send, on };
};

/**
 * Hook для управления чатом с AI агентом через WebSocket
 */
export const useChatAgent = (provider: 'local' | 'cloud', model: string) => {
  const { connected, send, on } = useWebSocket('ws://localhost:11434/ws/agent');
  const [isLoading, setIsLoading] = useState(false);

  useEffect(() => {
    const unsubscribe = on('task-update', (data) => {
      console.log('Task update:', data);
      setIsLoading(data.status === 'running');
    });

    return unsubscribe;
  }, [on]);

  const sendAgentMessage = useCallback(async (prompt: string, context?: any) => {
    setIsLoading(true);
    send('agent-request', {
      provider,
      model,
      prompt,
      context,
    });
  }, [provider, model, send]);

  return { connected, isLoading, sendAgentMessage };
};

/**
 * Hook для мониторинга задач в реальном времени
 */
export const useTaskMonitor = () => {
  const { on } = useWebSocket('ws://localhost:11434/ws/tasks');
  const [tasks, setTasks] = useState<any[]>([]);

  useEffect(() => {
    const unsubscribe = on('task-update', (data) => {
      setTasks(prev => {
        const existingIndex = prev.findIndex(t => t.id === data.id);
        if (existingIndex >= 0) {
          const updated = [...prev];
          updated[existingIndex] = { ...updated[existingIndex], ...data };
          return updated;
        }
        return [...prev, data];
      });
    });

    return unsubscribe;
  }, [on]);

  return { tasks };
};
