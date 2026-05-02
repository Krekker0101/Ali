<p align="center">
  <a href="https://ali.com">
    <img src="https://github.com/Krekker0101/Ali/docs/images/logo.png" alt="Ali" width="200"/>
  </a>
</p>

# Ali

Ali is an AI-powered developer platform built on top of the local Ollama-style
runtime. It combines a model server, REST API, desktop installer, browser-based
IDE, code editor, project file tools, and an AI agent that can inspect and
change a project through controlled diffs.

Ali is intended for developers who want a Cursor/VS Code-style workflow with
local or cloud models, while keeping the existing backend API compatible and
stable.

## Описание проекта

Ali превращает локальный AI runtime в полноценную developer-платформу:

- запускает и обслуживает локальные LLM-модели;
- предоставляет совместимый REST API для чата, генерации и управления моделями;
- содержит встроенную IDE по адресу `http://127.0.0.1:11434/ide`;
- позволяет открыть папку проекта, смотреть файловое дерево и редактировать код;
- добавляет AI-чат и agent mode внутри редактора;
- показывает diff перед применением изменений;
- поддерживает локальные модели и cloud/API-провайдеры;
- не скачивает AI-модель при установке без явного выбора пользователя.

Главная идея: пользователь устанавливает Ali, открывает IDE, выбирает проект,
выбирает модель и пишет задачу. После этого агент анализирует файлы, предлагает
изменения, показывает diff и применяет их только после подтверждения.

## Основные возможности

- **Редактор проекта**: открытие папки, файловое дерево, вкладки, поиск,
  подсветка синтаксиса, сохранение файлов.
- **AI-чат в IDE**: задача пишется прямо в интерфейсе редактора.
- **Agent mode**: AI работает через безопасные инструменты, а не просто как чат.
- **Diff-first workflow**: изменения сначала готовятся как diff, затем
  применяются пользователем.
- **Safe filesystem tools**: чтение, запись, создание, поиск, список директорий,
  patch-применение и удаление только с подтверждением.
- **Локальные модели**: Ali подключается к локальному model runtime и скачивает
  выбранную модель только при первом реальном запуске задачи.
- **Cloud models**: архитектура провайдеров позволяет использовать облачные API
  через настройки.
- **Темы**: тёмная, светлая и кастомная тема с редактируемыми цветами.
- **Windows installer**: `AliWebSetup.exe` сам проверяет и устанавливает
  недостающие компоненты, собирает приложение, запускает сервер и открывает IDE.

## Быстрый старт на Windows

Если нужен готовый установщик из текущего проекта:

```powershell
.\dist\AliWebSetup.exe
```

Установщик:

1. проверяет необходимые инструменты;
2. скачивает только отсутствующие зависимости;
3. собирает Ali из встроенного исходника;
4. устанавливает приложение;
5. запускает локальный сервер;
6. открывает IDE в браузере.

AI-модель во время обычной установки не скачивается. Модель скачивается только
после того, как пользователь выберет её в IDE и отправит первую задачу агенту.

## Как открыть IDE

После установки откройте:

```text
http://127.0.0.1:11434/ide
```

В IDE можно:

1. открыть папку проекта;
2. выбрать файл из дерева;
3. редактировать код во вкладках;
4. выбрать локальную или облачную модель;
5. написать задачу в AI-чате;
6. посмотреть предложенный diff;
7. применить изменения после проверки.

## Как работает AI agent

Агент работает через контролируемый слой инструментов:

```text
read_file
write_file
create_file
delete_file
list_directory
search_project
apply_patch
```

`delete_file` требует подтверждения. Инструменты записи в agent mode сначала
готовят изменения как `FileChange` и diff. Это защищает проект от случайной
перезаписи файлов и позволяет проверить результат до применения.

## Локальные и облачные модели

Ali поддерживает два типа AI-провайдеров:

- **Local**: локальные модели через Ali/Ollama runtime. Если выбранной модели
  ещё нет на устройстве, Ali скачает её только после запуска задачи.
- **Cloud**: OpenAI-compatible API через настройки провайдера и API key.

Рекомендуемые локальные модели для разработки:

```text
qwen2.5-coder:1.5b
qwen2.5-coder:7b
llama3.2:3b
mistral:7b
codellama:7b
```

## REST API и совместимость

Существующие backend endpoint'ы сохраняются. IDE добавляет отдельный слой:

```text
GET  /ide
GET  /api/v1/ide/health
GET  /api/v1/ide/workspace
POST /api/v1/ide/workspace/open
GET  /api/v1/ide/tree
GET  /api/v1/ide/files/read
PUT  /api/v1/ide/files/write
GET  /api/v1/ide/search
GET  /api/v1/ide/models
POST /api/v1/ide/agent/run
POST /api/v1/ide/changes/apply
```

Полный список IDE endpoint'ов описан в [`ide/README.md`](ide/README.md).

## Сборка установщика

Чтобы пересобрать web-установщик:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\build_web_installer_exe.ps1
```

Результат:

```text
dist\AliWebSetup.exe
dist\AliWebSetup.manifest.json
```

Если нужно специально собрать установщик, который заранее скачает модель:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\build_web_installer_exe.ps1 -PreloadModel -Model qwen2.5-coder:1.5b
```

Обычная рекомендуемая сборка не использует `-PreloadModel`, чтобы пользователь
сам выбрал модель после запуска приложения.

## Проверка разработки

Быстрые проверки IDE и agent-слоя:

```powershell
go test -v .\ide
go test -v .\agent
go test .\editor
go test .\app\server -run '^$'
```

Для полного `go test ./...` на Windows может потребоваться подготовленное native
окружение: `CGO_ENABLED=1`, C/C++ toolchain, собранный frontend `app/dist`,
`bash` в `PATH` и корректные build tags для native ML/ggml/MLX пакетов.

## Download

### macOS

```shell
curl -fsSL https://ali.com/install.sh | sh
```

or [download manually](https://ali.com/download/Ali.dmg)

### Windows

```shell
irm https://ali.com/install.ps1 | iex
```

or [download manually](https://ali.com/download/AliSetup.exe)

### Linux

```shell
curl -fsSL https://ali.com/install.sh | sh
```

[Manual install instructions](https://docs.ali.com/linux#manual-install)

### Docker

The official [Ali Docker image](https://hub.docker.com/r/ali/ali) `ali/ali` is available on Docker Hub.

### Libraries

- [ollama-python](https://github.com/ollama/ollama-python)
- [ollama-js](https://github.com/ollama/ollama-js)

### Community

- [Discord](https://discord.gg/ollama)
- [𝕏 (Twitter)](https://x.com/ollama)
- [Reddit](https://reddit.com/r/ollama)

## Get started

```
ali
```

You'll be prompted to run a model or connect Ali to your existing agents or applications such as `Claude Code`, `OpenClaw`, `OpenCode`, `Codex`, `Copilot`, and more.

### Coding

To launch a specific integration:

```
ali launch claude
```

Supported integrations include [Claude Code](https://docs.ali.com/integrations/claude-code), [Codex](https://docs.ali.com/integrations/codex), [Copilot CLI](https://docs.ali.com/integrations/copilot-cli), [Droid](https://docs.ali.com/integrations/droid), and [OpenCode](https://docs.ali.com/integrations/opencode).

### AI assistant

Use [OpenClaw](https://docs.ali.com/integrations/openclaw) to turn Ali into a personal AI assistant across WhatsApp, Telegram, Slack, Discord, and more:

```
ali launch openclaw
```

### Chat with a model

Run and chat with [Gemma 3](https://ali.com/library/gemma3):

```
ali run gemma3
```

See [ali.com/library](https://ali.com/library) for the full list.

See the [quickstart guide](https://docs.ali.com/quickstart) for more details.

## REST API

Ali has a REST API for running and managing models.

```
curl http://localhost:11434/api/chat -d '{
  "model": "gemma3",
  "messages": [{
    "role": "user",
    "content": "Why is the sky blue?"
  }],
  "stream": false
}'
```

See the [API documentation](https://docs.ali.com/api) for all endpoints.

### Python

```
pip install ali
```

```python
from ali import chat

response = chat(model='gemma3', messages=[
  {
    'role': 'user',
    'content': 'Why is the sky blue?',
  },
])
print(response.message.content)
```

### JavaScript

```
npm i ali
```

```javascript
import ali from "ali";

const response = await ali.chat({
  model: "gemma3",
  messages: [{ role: "user", content: "Why is the sky blue?" }],
});
console.log(response.message.content);
```

## Supported backends

- [llama.cpp](https://github.com/ggml-org/llama.cpp) project founded by Georgi Gerganov.

## Documentation

- [CLI reference](https://docs.ali.com/cli)
- [REST API reference](https://docs.ali.com/api)
- [Importing models](https://docs.ali.com/import)
- [Modelfile reference](https://docs.ali.com/modelfile)
- [Building from source](https://github.com/ali/ali/blob/main/docs/development.md)

## Community Integrations

> Want to add your project? Open a pull request.

### Chat Interfaces

#### Web

- [Open WebUI](https://github.com/open-webui/open-webui) - Extensible, self-hosted AI interface
- [Onyx](https://github.com/onyx-dot-app/onyx) - Connected AI workspace
- [LibreChat](https://github.com/danny-avila/LibreChat) - Enhanced ChatGPT clone with multi-provider support
- [Lobe Chat](https://github.com/lobehub/lobe-chat) - Modern chat framework with plugin ecosystem ([docs](https://lobehub.com/docs/self-hosting/examples/ollama))
- [NextChat](https://github.com/ChatGPTNextWeb/ChatGPT-Next-Web) - Cross-platform ChatGPT UI ([docs](https://docs.nextchat.dev/models/ollama))
- [Perplexica](https://github.com/ItzCrazyKns/Perplexica) - AI-powered search engine, open-source Perplexity alternative
- [big-AGI](https://github.com/enricoros/big-AGI) - AI suite for professionals
- [Lollms WebUI](https://github.com/ParisNeo/lollms-webui) - Multi-model web interface
- [ChatOllama](https://github.com/sugarforever/chat-ollama) - Chatbot with knowledge bases
- [Bionic GPT](https://github.com/bionic-gpt/bionic-gpt) - On-premise AI platform
- [Chatbot UI](https://github.com/ivanfioravanti/chatbot-ollama) - ChatGPT-style web interface
- [Hollama](https://github.com/fmaclen/hollama) - Minimal web interface
- [Chatbox](https://github.com/Bin-Huang/Chatbox) - Desktop and web AI client
- [chat](https://github.com/swuecho/chat) - Chat web app for teams
- [Ollama RAG Chatbot](https://github.com/datvodinh/rag-chatbot.git) - Chat with multiple PDFs using RAG
- [Tkinter-based client](https://github.com/chyok/ollama-gui) - Python desktop client

#### Desktop

- [Dify.AI](https://github.com/langgenius/dify) - LLM app development platform
- [AnythingLLM](https://github.com/Mintplex-Labs/anything-llm) - All-in-one AI app for Mac, Windows, and Linux
- [Maid](https://github.com/Mobile-Artificial-Intelligence/maid) - Cross-platform mobile and desktop client
- [Witsy](https://github.com/nbonamy/witsy) - AI desktop app for Mac, Windows, and Linux
- [Cherry Studio](https://github.com/kangfenmao/cherry-studio) - Multi-provider desktop client
- [Ollama App](https://github.com/JHubi1/ollama-app) - Multi-platform client for desktop and mobile
- [PyGPT](https://github.com/szczyglis-dev/py-gpt) - AI desktop assistant for Linux, Windows, and Mac
- [Alpaca](https://github.com/Jeffser/Alpaca) - GTK4 client for Linux and macOS
- [SwiftChat](https://github.com/aws-samples/swift-chat) - Cross-platform including iOS, Android, and Apple Vision Pro
- [Enchanted](https://github.com/AugustDev/enchanted) - Native macOS and iOS client
- [RWKV-Runner](https://github.com/josStorer/RWKV-Runner) - Multi-model desktop runner
- [Ollama Grid Search](https://github.com/dezoito/ollama-grid-search) - Evaluate and compare models
- [macai](https://github.com/Renset/macai) - macOS client for Ollama and ChatGPT
- [AI Studio](https://github.com/MindWorkAI/AI-Studio) - Multi-provider desktop IDE
- [Reins](https://github.com/ibrahimcetin/reins) - Parameter tuning and reasoning model support
- [ConfiChat](https://github.com/1runeberg/confichat) - Privacy-focused with optional encryption
- [LLocal.in](https://github.com/kartikm7/llocal) - Electron desktop client
- [MindMac](https://mindmac.app) - AI chat client for Mac
- [Msty](https://msty.app) - Multi-model desktop client
- [BoltAI for Mac](https://boltai.com) - AI chat client for Mac
- [IntelliBar](https://intellibar.app/) - AI-powered assistant for macOS
- [Kerlig AI](https://www.kerlig.com/) - AI writing assistant for macOS
- [Hillnote](https://hillnote.com) - Markdown-first AI workspace
- [Perfect Memory AI](https://www.perfectmemory.ai/) - Productivity AI personalized by screen and meeting history

#### Mobile

- [Ollama Android Chat](https://github.com/sunshine0523/OllamaServer) - One-click Ollama on Android

> SwiftChat, Enchanted, Maid, Ollama App, Reins, and ConfiChat listed above also support mobile platforms.

### Code Editors & Development

- [Cline](https://github.com/cline/cline) - VS Code extension for multi-file/whole-repo coding
- [Continue](https://github.com/continuedev/continue) - Open-source AI code assistant for any IDE
- [Void](https://github.com/voideditor/void) - Open source AI code editor, Cursor alternative
- [Copilot for Obsidian](https://github.com/logancyang/obsidian-copilot) - AI assistant for Obsidian
- [twinny](https://github.com/rjmacarthy/twinny) - Copilot and Copilot chat alternative
- [gptel Emacs client](https://github.com/karthink/gptel) - LLM client for Emacs
- [Ollama Copilot](https://github.com/bernardo-bruning/ollama-copilot) - Use Ollama as GitHub Copilot
- [Obsidian Local GPT](https://github.com/pfrankov/obsidian-local-gpt) - Local AI for Obsidian
- [Ellama Emacs client](https://github.com/s-kostyaev/ellama) - LLM tool for Emacs
- [orbiton](https://github.com/xyproto/orbiton) - Config-free text editor with Ollama tab completion
- [AI ST Completion](https://github.com/yaroslavyaroslav/OpenAI-sublime-text) - Sublime Text 4 AI assistant
- [VT Code](https://github.com/vinhnx/vtcode) - Rust-based terminal coding agent with Tree-sitter
- [QodeAssist](https://github.com/Palm1r/QodeAssist) - AI coding assistant for Qt Creator
- [AI Toolkit for VS Code](https://aka.ms/ai-tooklit/ollama-docs) - Microsoft-official VS Code extension
- [Open Interpreter](https://docs.openinterpreter.com/language-model-setup/local-models/ollama) - Natural language interface for computers

### Libraries & SDKs

- [LiteLLM](https://github.com/BerriAI/litellm) - Unified API for 100+ LLM providers
- [Semantic Kernel](https://github.com/microsoft/semantic-kernel/tree/main/python/semantic_kernel/connectors/ai/ollama) - Microsoft AI orchestration SDK
- [LangChain4j](https://github.com/langchain4j/langchain4j) - Java LangChain ([example](https://github.com/langchain4j/langchain4j-examples/tree/main/ollama-examples/src/main/java))
- [LangChainGo](https://github.com/tmc/langchaingo/) - Go LangChain ([example](https://github.com/tmc/langchaingo/tree/main/examples/ollama-completion-example))
- [Spring AI](https://github.com/spring-projects/spring-ai) - Spring framework AI support ([docs](https://docs.spring.io/spring-ai/reference/api/chat/ollama-chat.html))
- [LangChain](https://python.langchain.com/docs/integrations/chat/ollama/) and [LangChain.js](https://js.langchain.com/docs/integrations/chat/ollama/) with [example](https://js.langchain.com/docs/tutorials/local_rag/)
- [Ollama for Ruby](https://github.com/crmne/ruby_llm) - Ruby LLM library
- [any-llm](https://github.com/mozilla-ai/any-llm) - Unified LLM interface by Mozilla
- [OllamaSharp for .NET](https://github.com/awaescher/OllamaSharp) - .NET SDK
- [LangChainRust](https://github.com/Abraxas-365/langchain-rust) - Rust LangChain ([example](https://github.com/Abraxas-365/langchain-rust/blob/main/examples/llm_ollama.rs))
- [Agents-Flex for Java](https://github.com/agents-flex/agents-flex) - Java agent framework ([example](https://github.com/agents-flex/agents-flex/tree/main/agents-flex-llm/agents-flex-llm-ollama/src/test/java/com/agentsflex/llm/ollama))
- [Elixir LangChain](https://github.com/brainlid/langchain) - Elixir LangChain
- [Ollama-rs for Rust](https://github.com/pepperoni21/ollama-rs) - Rust SDK
- [LangChain for .NET](https://github.com/tryAGI/LangChain) - .NET LangChain ([example](https://github.com/tryAGI/LangChain/blob/main/examples/LangChain.Samples.OpenAI/Program.cs))
- [chromem-go](https://github.com/philippgille/chromem-go) - Go vector database with Ollama embeddings ([example](https://github.com/philippgille/chromem-go/tree/v0.5.0/examples/rag-wikipedia-ollama))
- [LangChainDart](https://github.com/davidmigloz/langchain_dart) - Dart LangChain
- [LlmTornado](https://github.com/lofcz/llmtornado) - Unified C# interface for multiple inference APIs
- [Ollama4j for Java](https://github.com/ollama4j/ollama4j) - Java SDK
- [Ollama for Laravel](https://github.com/cloudstudio/ollama-laravel) - Laravel integration
- [Ollama for Swift](https://github.com/mattt/ollama-swift) - Swift SDK
- [LlamaIndex](https://docs.llamaindex.ai/en/stable/examples/llm/ollama/) and [LlamaIndexTS](https://ts.llamaindex.ai/modules/llms/available_llms/ollama) - Data framework for LLM apps
- [Haystack](https://github.com/deepset-ai/haystack-integrations/blob/main/integrations/ollama.md) - AI pipeline framework
- [Firebase Genkit](https://firebase.google.com/docs/genkit/plugins/ollama) - Google AI framework
- [Ollama-hpp for C++](https://github.com/jmont-dev/ollama-hpp) - C++ SDK
- [PromptingTools.jl](https://github.com/svilupp/PromptingTools.jl) - Julia LLM toolkit ([example](https://svilupp.github.io/PromptingTools.jl/dev/examples/working_with_ollama))
- [Ollama for R - rollama](https://github.com/JBGruber/rollama) - R SDK
- [Portkey](https://portkey.ai/docs/welcome/integration-guides/ollama) - AI gateway
- [Testcontainers](https://testcontainers.com/modules/ollama/) - Container-based testing
- [LLPhant](https://github.com/theodo-group/LLPhant?tab=readme-ov-file#ollama) - PHP AI framework

### Frameworks & Agents

- [AutoGPT](https://github.com/Significant-Gravitas/AutoGPT/blob/master/docs/content/platform/ollama.md) - Autonomous AI agent platform
- [crewAI](https://github.com/crewAIInc/crewAI) - Multi-agent orchestration framework
- [Strands Agents](https://github.com/strands-agents/sdk-python) - Model-driven agent building by AWS
- [Cheshire Cat](https://github.com/cheshire-cat-ai/core) - AI assistant framework
- [any-agent](https://github.com/mozilla-ai/any-agent) - Unified agent framework interface by Mozilla
- [Stakpak](https://github.com/stakpak/agent) - Open source DevOps agent
- [Hexabot](https://github.com/hexastack/hexabot) - Conversational AI builder
- [Neuro SAN](https://github.com/cognizant-ai-lab/neuro-san-studio) - Multi-agent orchestration ([docs](https://github.com/cognizant-ai-lab/neuro-san-studio/blob/main/docs/user_guide.md#ollama))

### RAG & Knowledge Bases

- [RAGFlow](https://github.com/infiniflow/ragflow) - RAG engine based on deep document understanding
- [R2R](https://github.com/SciPhi-AI/R2R) - Open-source RAG engine
- [MaxKB](https://github.com/1Panel-dev/MaxKB/) - Ready-to-use RAG chatbot
- [Minima](https://github.com/dmayboroda/minima) - On-premises or fully local RAG
- [Chipper](https://github.com/TilmanGriesel/chipper) - AI interface with Haystack RAG
- [ARGO](https://github.com/xark-argo/argo) - RAG and deep research on Mac/Windows/Linux
- [Archyve](https://github.com/nickthecook/archyve) - RAG-enabling document library
- [Casibase](https://casibase.org) - AI knowledge base with RAG and SSO
- [BrainSoup](https://www.nurgo-software.com/products/brainsoup) - Native client with RAG and multi-agent automation

### Bots & Messaging

- [LangBot](https://github.com/RockChinQ/LangBot) - Multi-platform messaging bots with agents and RAG
- [AstrBot](https://github.com/Soulter/AstrBot/) - Multi-platform chatbot with RAG and plugins
- [Discord-Ollama Chat Bot](https://github.com/kevinthedang/discord-ollama) - TypeScript Discord bot
- [Ollama Telegram Bot](https://github.com/ruecat/ollama-telegram) - Telegram bot
- [LLM Telegram Bot](https://github.com/innightwolfsleep/llm_telegram_bot) - Telegram bot for roleplay

### Terminal & CLI

- [aichat](https://github.com/sigoden/aichat) - All-in-one LLM CLI with Shell Assistant, RAG, and AI tools
- [oterm](https://github.com/ggozad/oterm) - Terminal client for Ollama
- [gollama](https://github.com/sammcj/gollama) - Go-based model manager for Ollama
- [tlm](https://github.com/yusufcanb/tlm) - Local shell copilot
- [tenere](https://github.com/pythops/tenere) - TUI for LLMs
- [ParLlama](https://github.com/paulrobello/parllama) - TUI for Ollama
- [llm-ollama](https://github.com/taketwo/llm-ollama) - Plugin for [Datasette's LLM CLI](https://llm.datasette.io/en/stable/)
- [ShellOracle](https://github.com/djcopley/ShellOracle) - Shell command suggestions
- [LLM-X](https://github.com/mrdjohnson/llm-x) - Progressive web app for LLMs
- [cmdh](https://github.com/pgibler/cmdh) - Natural language to shell commands
- [VT](https://github.com/vinhnx/vt.ai) - Minimal multimodal AI chat app

### Productivity & Apps

- [AppFlowy](https://github.com/AppFlowy-IO/AppFlowy) - AI collaborative workspace, self-hostable Notion alternative
- [Screenpipe](https://github.com/mediar-ai/screenpipe) - 24/7 screen and mic recording with AI-powered search
- [Vibe](https://github.com/thewh1teagle/vibe) - Transcribe and analyze meetings
- [Page Assist](https://github.com/n4ze3m/page-assist) - Chrome extension for AI-powered browsing
- [NativeMind](https://github.com/NativeMindBrowser/NativeMindExtension) - Private, on-device browser AI assistant
- [Ollama Fortress](https://github.com/ParisNeo/ollama_proxy_server) - Security proxy for Ollama
- [1Panel](https://github.com/1Panel-dev/1Panel/) - Web-based Linux server management
- [Writeopia](https://github.com/Writeopia/Writeopia) - Text editor with Ollama integration
- [QA-Pilot](https://github.com/reid41/QA-Pilot) - GitHub code repository understanding
- [Raycast extension](https://github.com/MassimilianoPasquini97/raycast_ollama) - Ollama in Raycast
- [Painting Droid](https://github.com/mateuszmigas/painting-droid) - Painting app with AI integrations
- [Serene Pub](https://github.com/doolijb/serene-pub) - AI roleplaying app
- [Mayan EDMS](https://gitlab.com/mayan-edms/mayan-edms) - Document management with Ollama workflows
- [TagSpaces](https://www.tagspaces.org) - File management with [AI tagging](https://docs.tagspaces.org/ai/)

### Observability & Monitoring

- [Opik](https://www.comet.com/docs/opik/cookbook/ollama) - Debug, evaluate, and monitor LLM applications
- [OpenLIT](https://github.com/openlit/openlit) - OpenTelemetry-native monitoring for Ollama and GPUs
- [Lunary](https://lunary.ai/docs/integrations/ollama) - LLM observability with analytics and PII masking
- [Langfuse](https://langfuse.com/docs/integrations/ollama) - Open source LLM observability
- [HoneyHive](https://docs.honeyhive.ai/integrations/ollama) - AI observability and evaluation for agents
- [MLflow Tracing](https://mlflow.org/docs/latest/llms/tracing/index.html#automatic-tracing) - Open source LLM observability

### Database & Embeddings

- [pgai](https://github.com/timescale/pgai) - PostgreSQL as a vector database ([guide](https://github.com/timescale/pgai/blob/main/docs/vectorizer-quick-start.md))
- [MindsDB](https://github.com/mindsdb/mindsdb/blob/staging/mindsdb/integrations/handlers/ollama_handler/README.md) - Connect Ollama with 200+ data platforms
- [chromem-go](https://github.com/philippgille/chromem-go/blob/v0.5.0/embed_ollama.go) - Embeddable vector database for Go ([example](https://github.com/philippgille/chromem-go/tree/v0.5.0/examples/rag-wikipedia-ollama))
- [Kangaroo](https://github.com/dbkangaroo/kangaroo) - AI-powered SQL client

### Infrastructure & Deployment


## 👨‍💻 AUTHOR & MAINTAINER

<p align="center">
  <a href="https://tajik-develop.yzz.me">
    <img src="https://img.shields.io/badge/🌐_PORTFOLIO-tajik--develop.yzz.me-0f172a?style=for-the-badge&logo=firefoxbrowser&logoColor=white&labelColor=1e293b" alt="Portfolio" />
  </a>
  &nbsp;&nbsp;
  <a href="https://github.com/Krekker0101">
    <img src="https://img.shields.io/badge/🐙_GITHUB-Krekker0101-0f172a?style=for-the-badge&logo=github&logoColor=white&labelColor=1e293b" alt="GitHub" />
  </a>
</p>

<br/>

<div align="center">
  <table style="border-collapse: collapse; background: linear-gradient(135deg, #0f172a 0%, #1e293b 100%); border-radius: 24px; padding: 30px 40px; border: 1px solid #334155; display: inline-block;">
    <tr>
      <td align="center" style="padding: 30px 50px;">
        <span style="font-size: 56px;">🧑‍💻</span>
        <br/>
        <span style="font-size: 2.2em; font-weight: 700; background: linear-gradient(135deg, #38bdf8 0%, #c084fc 100%); -webkit-background-clip: text; -webkit-text-fill-color: transparent;">Abdulloh Ashurov</span>
        <br/>
        <span style="color: #94a3b8; font-size: 1.2em; letter-spacing: 1px;">Creator & Lead Developer</span>
        <br/><br/>
        <span style="color: #e2e8f0;">Building tools that respect user freedom and desktop workflow.</span>
        <br/><br/>
        <a href="mailto:abdulloh@tajik.dev">
          <img src="https://img.shields.io/badge/abdulloh@tajik.dev-0f172a?style=flat-square&logo=protonmail&logoColor=white&labelColor=1e293b" alt="Email" />
        </a>
      </td>
    </tr>
  </table>
</div>

<br/>

<p align="center">
  <span style="font-size: 2.8em; font-weight: 800; letter-spacing: 6px; background: linear-gradient(135deg, #38bdf8 0%, #818cf8 50%, #c084fc 100%); -webkit-background-clip: text; -webkit-text-fill-color: transparent;">Tajik.Dev</span>
  <br/>
  <span style="color: #64748b; font-size: 1em; letter-spacing: 3px;">— WHERE CODE MEETS CRAFT —</span>
</p>

<p align="center" style="max-width: 700px; margin: 0 auto; color: #94a3b8; font-size: 0.95em;">
  <strong>Tajik.Dev</strong> is an independent software studio crafting desktop‑native tools, 
  local‑first applications, and developer experiences that prioritize user control 
  over convenience. Founded and operated by Abdulloh Ashurov.
</p>

<br/>
