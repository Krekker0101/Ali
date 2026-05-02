import React, { useState, useRef, useEffect } from 'react';
import './ChatComponent.css';

interface ChatMessage {
  id: string;
  type: 'user' | 'assistant' | 'system';
  content: string;
  timestamp: number;
  loading?: boolean;
}

interface ChatComponentProps {
  onSendMessage: (message: string) => Promise<void>;
  isLoading: boolean;
  provider: 'local' | 'cloud';
  model: string;
}

export const ChatComponent: React.FC<ChatComponentProps> = ({
  onSendMessage,
  isLoading,
  provider,
  model,
}) => {
  const [messages, setMessages] = useState<ChatMessage[]>([
    {
      id: '0',
      type: 'system',
      content: `👋 Добро пожаловать! Вы используете ${provider === 'local' ? '🖥️ локальные' : '☁️ облачные'} модели. Отправьте запрос ИИ агенту.`,
      timestamp: Date.now(),
    },
  ]);
  const [inputValue, setInputValue] = useState('');
  const messagesEndRef = useRef<HTMLDivElement>(null);

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  };

  useEffect(() => {
    scrollToBottom();
  }, [messages]);

  const handleSend = async () => {
    if (!inputValue.trim() || isLoading) return;

    const userMessage: ChatMessage = {
      id: Date.now().toString(),
      type: 'user',
      content: inputValue,
      timestamp: Date.now(),
    };

    setMessages(prev => [...prev, userMessage]);
    const inputToSend = inputValue;
    setInputValue('');

    // Loading message
    const loadingId = (Date.now() + 1).toString();
    setMessages(prev => [
      ...prev,
      {
        id: loadingId,
        type: 'assistant',
        content: 'Обработка...',
        timestamp: Date.now(),
        loading: true,
      },
    ]);

    try {
      await onSendMessage(inputToSend);

      // Replace loading message with actual response
      setMessages(prev =>
        prev.map(msg =>
          msg.id === loadingId
            ? {
                ...msg,
                content: 'Ответ обработан успешно',
                loading: false,
              }
            : msg
        )
      );
    } catch (error) {
      setMessages(prev =>
        prev.map(msg =>
          msg.id === loadingId
            ? {
                ...msg,
                content: `❌ Ошибка: ${error instanceof Error ? error.message : 'Unknown error'}`,
                loading: false,
              }
            : msg
        )
      );
    }
  };

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && e.ctrlKey) {
      handleSend();
    }
  };

  return (
    <div className="chat-container-new">
      <div className="chat-messages-new">
        {messages.map(msg => (
          <div key={msg.id} className={`message-new message-${msg.type}`}>
            <div className="message-content-new">
              {msg.loading && <span className="loading-spinner-new"></span>}
              {msg.content}
            </div>
            <div className="message-time-new">
              {new Date(msg.timestamp).toLocaleTimeString()}
            </div>
          </div>
        ))}
        <div ref={messagesEndRef} />
      </div>

      <div className="chat-input-new">
        <input
          type="text"
          value={inputValue}
          onChange={e => setInputValue(e.target.value)}
          onKeyPress={handleKeyPress}
          placeholder="Отправьте задачу ИИ (Ctrl+Enter)..."
          disabled={isLoading || !model}
          className="input-field-new"
        />
        <button
          onClick={handleSend}
          disabled={isLoading || !model || !inputValue.trim()}
          className="send-btn-new"
          title="Отправить (Ctrl+Enter)"
        >
          {isLoading ? '⏳' : '➤'}
        </button>
      </div>

      <div className="chat-footer-new">
        <div className="chat-tips-new">
          <strong>💡 Примеры:</strong>
          <ul>
            <li>"Отрефакторь этот файл"</li>
            <li>"Найди баги в коде"</li>
            <li>"Напиши документацию"</li>
            <li>"Оптимизируй производительность"</li>
          </ul>
        </div>
      </div>
    </div>
  );
};
