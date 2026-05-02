import React, { useState, useEffect } from 'react';
import './ProviderSelector.css';

interface ProviderSelectorProps {
  provider: 'local' | 'cloud';
  onProviderChange: (provider: 'local' | 'cloud') => void;
  model: string;
  onModelChange: (model: string) => void;
}

interface Model {
  name: string;
  description: string;
  params?: string;
}

const LOCAL_MODELS: Model[] = [
  { name: 'llama2', description: 'Llama 2 7B', params: '7B' },
  { name: 'gemma', description: 'Google Gemma 7B', params: '7B' },
  { name: 'mistral', description: 'Mistral 7B', params: '7B' },
  { name: 'neural-chat', description: 'Neural Chat 7B', params: '7B' },
  { name: 'llama2-13b', description: 'Llama 2 13B', params: '13B' },
];

const CLOUD_PROVIDERS: Model[] = [
  { name: 'anthropic-claude', description: 'Anthropic Claude 3', params: 'Cloud' },
  { name: 'openai-gpt4', description: 'OpenAI GPT-4', params: 'Cloud' },
  { name: 'openai-gpt35', description: 'OpenAI GPT-3.5', params: 'Cloud' },
];

export const ProviderSelector: React.FC<ProviderSelectorProps> = ({
  provider,
  onProviderChange,
  model,
  onModelChange,
}) => {
  const [availableModels, setAvailableModels] = useState<Model[]>([]);
  const [isLoading, setIsLoading] = useState(false);

  useEffect(() => {
    setAvailableModels(provider === 'local' ? LOCAL_MODELS : CLOUD_PROVIDERS);
    if (!model && availableModels.length > 0) {
      onModelChange(availableModels[0].name);
    }
  }, [provider]);

  const handleProviderChange = (newProvider: 'local' | 'cloud') => {
    onProviderChange(newProvider);
    // Auto-select first available model
    const models = newProvider === 'local' ? LOCAL_MODELS : CLOUD_PROVIDERS;
    onModelChange(models[0].name);
  };

  return (
    <div className="provider-selector">
      <div className="provider-buttons">
        <button
          className={`provider-btn ${provider === 'local' ? 'active' : ''}`}
          onClick={() => handleProviderChange('local')}
          title="Запуск моделей локально (быстро, приватно)"
        >
          <span className="icon">🖥️</span>
          <span>Local</span>
        </button>
        <button
          className={`provider-btn ${provider === 'cloud' ? 'active' : ''}`}
          onClick={() => handleProviderChange('cloud')}
          title="Облачные модели (мощнее, требует API ключ)"
        >
          <span className="icon">☁️</span>
          <span>Cloud</span>
        </button>
      </div>

      <div className="model-selector">
        <label>Model:</label>
        <select 
          value={model} 
          onChange={(e) => onModelChange(e.target.value)}
          className="model-select"
        >
          {availableModels.map(m => (
            <option key={m.name} value={m.name}>
              {m.description} {m.params ? `(${m.params})` : ''}
            </option>
          ))}
        </select>
        
        <div className="model-status">
          {provider === 'local' ? (
            <span className="status-badge local">
              <span className="dot"></span>
              Local
            </span>
          ) : (
            <span className="status-badge cloud">
              <span className="dot"></span>
              Cloud
            </span>
          )}
        </div>
      </div>

      <div className="provider-info">
        {provider === 'local' ? (
          <p>
            💡 Модели запускаются <strong>локально</strong> на вашем ПК.
            <br />
            Быстрое выполнение, полная приватность, требует GPU/много RAM.
          </p>
        ) : (
          <p>
            ☁️ Используются <strong>облачные API</strong> (Anthropic, OpenAI).
            <br />
            Мощные модели, требует API ключ и интернет соединение.
          </p>
        )}
      </div>
    </div>
  );
};
