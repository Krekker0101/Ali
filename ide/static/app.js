const state = {
  workspace: null,
  settings: null,
  tabs: new Map(),
  activePath: null,
  changes: [],
  steps: [],
  highlightFrame: 0
};

const $ = (id) => document.getElementById(id);

const els = {
  workspaceForm: $("workspaceForm"),
  workspacePath: $("workspacePath"),
  projectName: $("projectName"),
  fileTree: $("fileTree"),
  refreshTree: $("refreshTree"),
  newFile: $("newFile"),
  projectSearch: $("projectSearch"),
  searchResults: $("searchResults"),
  tabs: $("tabs"),
  editor: $("codeEditor"),
  highlight: $("highlightLayer"),
  editorSearch: $("editorSearch"),
  saveFile: $("saveFile"),
  editorStatus: $("editorStatus"),
  providerSelect: $("providerSelect"),
  modelSelect: $("modelSelect"),
  settingsButton: $("settingsButton"),
  settingsDialog: $("settingsDialog"),
  themeMode: $("themeMode"),
  colorSettings: $("colorSettings"),
  cloudBaseUrl: $("cloudBaseUrl"),
  cloudApiKey: $("cloudApiKey"),
  saveSettings: $("saveSettings"),
  chatLog: $("chatLog"),
  agentForm: $("agentForm"),
  agentTask: $("agentTask"),
  changesPane: $("changesPane"),
  applyChanges: $("applyChanges"),
  confirmDelete: $("confirmDelete")
};

async function api(path, options = {}) {
  const init = {
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...(options.headers || {})
    }
  };
  const response = await fetch(`/api/v1/ide${path}`, init);
  const text = await response.text();
  const data = text ? JSON.parse(text) : null;
  if (!response.ok) {
    throw new Error((data && data.error) || response.statusText);
  }
  return data;
}

function body(data) {
  return { body: JSON.stringify(data) };
}

async function init() {
  bindEvents();
  await loadSettings();
  await loadWorkspace();
  await loadModels();
  await refreshTree();
  document.body.classList.add("is-ready");
  setStatus("Ready");
}

function bindEvents() {
  window.addEventListener("beforeunload", (event) => {
    if (hasDirtyTabs()) {
      event.preventDefault();
      event.returnValue = "";
    }
  });

  window.addEventListener("keydown", (event) => {
    const key = event.key.toLowerCase();
    if ((event.ctrlKey || event.metaKey) && key === "s") {
      event.preventDefault();
      runTask(saveActiveFile);
    }
    if ((event.ctrlKey || event.metaKey) && key === "f" && !event.shiftKey) {
      event.preventDefault();
      els.editorSearch.focus();
      els.editorSearch.select();
    }
    if ((event.ctrlKey || event.metaKey) && event.shiftKey && key === "f") {
      event.preventDefault();
      els.projectSearch.focus();
      els.projectSearch.select();
    }
    if ((event.ctrlKey || event.metaKey) && key === "enter") {
      event.preventDefault();
      runTask(runAgent);
    }
  });

  els.workspaceForm.addEventListener("submit", async (event) => {
    event.preventDefault();
    if (!confirmDiscardDirtyTabs()) return;
    await runTask(async () => {
      const workspace = await api("/workspace/open", { method: "POST", ...body({ path: els.workspacePath.value }) });
      setWorkspace(workspace);
      closeAllTabs();
      await refreshTree();
    });
  });

  els.refreshTree.addEventListener("click", () => runTask(refreshTree));
  els.newFile.addEventListener("click", () => runTask(createNewFile));
  els.saveFile.addEventListener("click", () => runTask(saveActiveFile));

  els.editor.addEventListener("input", () => {
    const tab = activeTab();
    if (!tab) return;
    tab.content = els.editor.value;
    tab.dirty = true;
    renderTabs();
    scheduleHighlight();
    setStatus("Modified");
  });

  els.editor.addEventListener("scroll", () => {
    els.highlight.scrollTop = els.editor.scrollTop;
    els.highlight.scrollLeft = els.editor.scrollLeft;
  });

  els.editorSearch.addEventListener("keydown", (event) => {
    if (event.key === "Enter") {
      event.preventDefault();
      findInEditor();
    }
  });

  let searchTimer = 0;
  els.projectSearch.addEventListener("input", () => {
    clearTimeout(searchTimer);
    searchTimer = setTimeout(() => runTask(searchProject), 250);
  });

  els.providerSelect.addEventListener("change", async () => {
    if (state.settings) {
      state.settings.ai.provider = els.providerSelect.value;
      await loadModels();
    }
  });

  els.settingsButton.addEventListener("click", () => {
    renderColorSettings();
    els.settingsDialog.showModal();
  });

  els.saveSettings.addEventListener("click", () => runTask(saveSettings));
  els.agentForm.addEventListener("submit", (event) => {
    event.preventDefault();
    runTask(runAgent);
  });
  els.applyChanges.addEventListener("click", () => runTask(applySelectedChanges));
}

async function loadWorkspace() {
  const workspace = await api("/workspace");
  setWorkspace(workspace);
}

function setWorkspace(workspace) {
  state.workspace = workspace;
  els.workspacePath.value = workspace.root || "";
  els.projectName.textContent = workspace.name || "Project";
}

async function loadSettings() {
  state.settings = await api("/settings");
  els.providerSelect.value = state.settings.ai.provider || "local";
  els.themeMode.value = state.settings.theme.mode || "dark";
  els.cloudBaseUrl.value = state.settings.ai.cloud_base_url || "";
  applyTheme(state.settings.theme);
}

async function saveSettings() {
  const colors = {};
  document.querySelectorAll("[data-color-key]").forEach((input) => {
    colors[input.dataset.colorKey] = input.value;
  });
  const payload = {
    theme: { mode: els.themeMode.value, colors },
    ai: {
      provider: els.providerSelect.value,
      model: els.modelSelect.value,
      cloud_base_url: els.cloudBaseUrl.value,
      cloud_api_key: els.cloudApiKey.value,
      temperature: state.settings.ai.temperature || 0.2,
      max_tokens: state.settings.ai.max_tokens || 2048
    }
  };
  state.settings = await api("/settings", { method: "PUT", ...body(payload) });
  applyTheme(state.settings.theme);
  els.settingsDialog.close();
  await loadModels();
}

async function loadModels() {
  const provider = els.providerSelect.value || "local";
  els.modelSelect.innerHTML = "";
  const placeholder = document.createElement("option");
  placeholder.value = "";
  placeholder.textContent = "Select model";
  els.modelSelect.appendChild(placeholder);
  try {
    const data = await api(`/models?provider=${encodeURIComponent(provider)}`);
    const models = data.models || [];
    for (const model of models) {
      const option = document.createElement("option");
      option.value = model;
      option.textContent = model;
      els.modelSelect.appendChild(option);
    }
    const configured = state.settings && state.settings.ai.model;
    if (configured && !models.includes(configured)) {
      const option = document.createElement("option");
      option.value = configured;
      option.textContent = configured;
      els.modelSelect.appendChild(option);
    }
    if (configured) els.modelSelect.value = configured;
  } catch (err) {
    const option = document.createElement("option");
    option.value = "";
    option.textContent = err.message;
    els.modelSelect.appendChild(option);
  }
}

function applyTheme(theme) {
  const defaults = {
    dark: {
      background: "#0f1115",
      panel: "rgba(24, 26, 32, 0.68)",
      panel2: "rgba(36, 39, 48, 0.62)",
      text: "#f4f6fb",
      muted: "#a5adbb",
      accent: "#65d6ad",
      border: "rgba(255, 255, 255, 0.13)",
      danger: "#ff6b7a"
    },
    light: {
      background: "#f7f8fb",
      panel: "rgba(255, 255, 255, 0.68)",
      panel2: "rgba(255, 255, 255, 0.54)",
      text: "#171a22",
      muted: "#667085",
      accent: "#0c8f72",
      border: "rgba(25, 28, 36, 0.14)",
      danger: "#cf3144"
    }
  };
  const colors = theme.mode === "custom" ? theme.colors : defaults[theme.mode] || defaults.dark;
  Object.entries(colors || defaults.dark).forEach(([key, value]) => {
    document.documentElement.style.setProperty(`--${key}`, value);
  });
}

function renderColorSettings() {
  const colors = state.settings.theme.colors || {};
  els.colorSettings.innerHTML = "";
  for (const [key, value] of Object.entries(colors)) {
    const row = document.createElement("label");
    row.className = "color-row";
    row.textContent = key;
    const input = document.createElement("input");
    input.type = String(value).startsWith("#") ? "color" : "text";
    input.value = value;
    input.dataset.colorKey = key;
    row.appendChild(input);
    els.colorSettings.appendChild(row);
  }
}

async function refreshTree() {
  const data = await api("/tree?depth=4");
  renderTree(data.nodes || []);
}

function renderTree(nodes) {
  els.fileTree.innerHTML = "";
  const fragment = document.createDocumentFragment();
  for (const node of nodes) {
    fragment.appendChild(renderNode(node));
  }
  els.fileTree.appendChild(fragment);
}

function renderNode(node) {
  const wrap = document.createElement("div");
  const button = document.createElement("button");
  button.type = "button";
  button.className = `tree-node ${state.activePath === node.path ? "active" : ""}`;
  button.dataset.path = node.path;
  button.innerHTML = `<span class="node-icon">${node.type === "directory" ? "[D]" : "[F]"}</span><span class="node-name"></span>`;
  button.querySelector(".node-name").textContent = node.name;
  wrap.appendChild(button);

  if (node.type === "directory") {
    const children = document.createElement("div");
    children.className = "tree-children";
    children.hidden = false;
    for (const child of node.children || []) {
      children.appendChild(renderNode(child));
    }
    button.addEventListener("click", () => {
      children.hidden = !children.hidden;
    });
    wrap.appendChild(children);
  } else {
    button.addEventListener("click", () => openFile(node.path));
  }
  return wrap;
}

async function openFile(path) {
  if (state.tabs.has(path)) {
    activateTab(path);
    return;
  }
  const file = await api(`/files/read?path=${encodeURIComponent(path)}`);
  state.tabs.set(path, {
    path,
    name: path.split("/").pop(),
    language: file.language,
    content: file.content,
    saved: file.content,
    dirty: false
  });
  activateTab(path);
}

function activateTab(path) {
  state.activePath = path;
  const tab = activeTab();
  els.editor.value = tab ? tab.content : "";
  renderTabs();
  scheduleHighlight();
  refreshActiveTreeMarkers();
  setStatus(tab ? tab.language : "");
}

function activeTab() {
  return state.activePath ? state.tabs.get(state.activePath) : null;
}

function renderTabs() {
  els.tabs.innerHTML = "";
  for (const tab of state.tabs.values()) {
    const item = document.createElement("div");
    item.className = `tab ${tab.path === state.activePath ? "active" : ""}`;
    item.innerHTML = `<span class="tab-name"></span><span>${tab.dirty ? "*" : ""}</span><span class="tab-close">x</span>`;
    item.querySelector(".tab-name").textContent = tab.name;
    item.addEventListener("click", () => activateTab(tab.path));
    item.querySelector(".tab-close").addEventListener("click", (event) => {
      event.stopPropagation();
      closeTab(tab.path);
    });
    els.tabs.appendChild(item);
  }
}

function closeTab(path) {
  const tab = state.tabs.get(path);
  if (tab && tab.dirty && !confirm(`Close ${tab.name} without saving?`)) {
    return;
  }
  state.tabs.delete(path);
  if (state.activePath === path) {
    const next = state.tabs.keys().next();
    state.activePath = next.done ? null : next.value;
  }
  activateTab(state.activePath);
}

function closeAllTabs() {
  state.tabs.clear();
  state.activePath = null;
  activateTab(null);
}

function hasDirtyTabs() {
  return [...state.tabs.values()].some((tab) => tab.dirty);
}

function confirmDiscardDirtyTabs() {
  return !hasDirtyTabs() || confirm("Discard unsaved changes?");
}

async function saveActiveFile() {
  const tab = activeTab();
  if (!tab) return;
  await api("/files/write", { method: "PUT", ...body({ path: tab.path, content: tab.content }) });
  tab.saved = tab.content;
  tab.dirty = false;
  renderTabs();
  setStatus("Saved");
}

async function createNewFile() {
  const path = prompt("New file path");
  if (!path) return;
  await api("/files/create", { method: "POST", ...body({ path, content: "" }) });
  await refreshTree();
  await openFile(path);
}

function renderHighlight() {
  const tab = activeTab();
  const code = els.editor.value;
  if (code.length > 220000) {
    els.highlight.textContent = code;
    setStatus("Large file: plain mode");
    return;
  }
  els.highlight.innerHTML = highlight(code, tab ? tab.language : "text");
}

function scheduleHighlight() {
  if (state.highlightFrame) {
    cancelAnimationFrame(state.highlightFrame);
  }
  state.highlightFrame = requestAnimationFrame(() => {
    state.highlightFrame = 0;
    renderHighlight();
  });
}

function highlight(code, language) {
  return code.split("\n").map((line) => highlightLine(line, language)).join("\n") + "\n";
}

function highlightLine(line, language) {
  const marker = commentMarker(language);
  let code = line;
  let comment = "";
  if (marker) {
    const index = line.indexOf(marker);
    if (index >= 0) {
      code = line.slice(0, index);
      comment = line.slice(index);
    }
  }
  return highlightCode(code, language) + (comment ? `<span class="tok-comment">${escapeHtml(comment)}</span>` : "");
}

function highlightCode(code, language) {
  let html = escapeHtml(code);
  const strings = [];
  html = html.replace(/(&quot;.*?&quot;|'.*?'|`.*?`)/g, (match) => {
    const id = strings.length;
    strings.push(`<span class="tok-string">${match}</span>`);
    return `@@S${id}@@`;
  });

  const keywords = keywordSet(language);
  if (keywords.length) {
    html = html.replace(new RegExp(`\\b(${keywords.join("|")})\\b`, "g"), '<span class="tok-keyword">$1</span>');
  }
  html = html.replace(/\b([0-9]+(?:\.[0-9]+)?)\b/g, '<span class="tok-number">$1</span>');
  html = html.replace(/@@S(\d+)@@/g, (_, id) => strings[Number(id)]);
  return html;
}

function keywordSet(language) {
  if (language === "go") {
    return ["package", "import", "func", "type", "struct", "interface", "return", "if", "else", "for", "range", "switch", "case", "default", "go", "defer", "var", "const", "map", "chan", "select"];
  }
  if (["javascript", "typescript", "jsx", "tsx"].includes(language)) {
    return ["const", "let", "var", "function", "return", "if", "else", "for", "while", "class", "new", "async", "await", "import", "export", "from", "try", "catch"];
  }
  if (language === "python") {
    return ["def", "class", "return", "if", "elif", "else", "for", "while", "import", "from", "as", "try", "except", "with", "lambda"];
  }
  return [];
}

function commentMarker(language) {
  if (["python", "shell", "yaml", "toml"].includes(language)) return "#";
  if (language === "markdown") return null;
  return "//";
}

function escapeHtml(value) {
  return value
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;");
}

function findInEditor() {
  const query = els.editorSearch.value;
  if (!query) return;
  const start = els.editor.selectionEnd || 0;
  let index = els.editor.value.indexOf(query, start);
  if (index < 0) index = els.editor.value.indexOf(query, 0);
  if (index >= 0) {
    els.editor.focus();
    els.editor.setSelectionRange(index, index + query.length);
    setStatus(`Found at ${index + 1}`);
  } else {
    setStatus("No match");
  }
}

async function searchProject() {
  const query = els.projectSearch.value.trim();
  els.searchResults.innerHTML = "";
  if (!query) return;
  const data = await api(`/search?q=${encodeURIComponent(query)}`);
  for (const hit of data.results || []) {
    const button = document.createElement("button");
    button.type = "button";
    button.className = "search-hit";
    button.innerHTML = `<strong></strong><span></span>`;
    button.querySelector("strong").textContent = `${hit.path}:${hit.line}`;
    button.querySelector("span").textContent = hit.preview;
    button.addEventListener("click", () => openFile(hit.path));
    els.searchResults.appendChild(button);
  }
}

async function runAgent() {
  const task = els.agentTask.value.trim();
  if (!task) return;
  const provider = els.providerSelect.value;
  const model = els.modelSelect.value;
  if (!model) {
    setStatus("Select model first");
    addMessage("assistant", "Choose a model first. For local models, Ali will download it only after you run a task.");
    return;
  }
  addMessage("user", task);
  els.agentTask.value = "";
  setStatus(provider === "local" ? "Agent running. Local model may download first." : "Agent running");
  const files = state.activePath ? [state.activePath] : [];
  document.body.classList.add("agent-running");
  try {
    const response = await api("/agent/run", {
      method: "POST",
      ...body({
        task,
        files,
        provider,
        model
      })
    });
    addMessage("assistant", response.message || response.raw || "No message");
    state.steps = response.steps || [];
    if (state.steps.length) {
      addMessage("assistant", formatAgentTrace(state.steps));
    }
    state.changes = response.changes || [];
    renderChanges();
    setStatus(state.changes.length ? "Diff ready" : "Done");
  } finally {
    document.body.classList.remove("agent-running");
  }
}

function addMessage(role, text) {
  const item = document.createElement("div");
  item.className = `message ${role}`;
  item.textContent = text;
  els.chatLog.appendChild(item);
  els.chatLog.scrollTop = els.chatLog.scrollHeight;
}

function renderChanges() {
  els.changesPane.innerHTML = "";
  state.changes.forEach((change, index) => {
    const item = document.createElement("div");
    item.className = "change-item";
    item.innerHTML = `
      <div class="change-title">
        <input type="checkbox" data-change-index="${index}" checked />
        <code>${escapeHtml(change.action)}</code>
        <span>${escapeHtml(change.path)}</span>
      </div>
      <pre class="diff">${formatDiff(change.diff || "")}</pre>
    `;
    els.changesPane.appendChild(item);
  });
}

function formatAgentTrace(steps) {
  const lines = ["Agent trace:"];
  for (const step of steps) {
    const tools = (step.tool_calls || [])
      .map((call) => call.tool || call.name)
      .filter(Boolean)
      .join(", ");
    const errors = (step.results || []).filter((result) => result.error).length;
    const suffix = errors ? `, ${errors} error(s)` : "";
    lines.push(`Round ${step.round}: ${tools || "final"}${suffix}`);
    if (step.thought) lines.push(`  ${step.thought}`);
  }
  return lines.join("\n");
}

function formatDiff(diff) {
  return escapeHtml(diff)
    .split("\n")
    .map((line) => {
      if (line.startsWith("+") && !line.startsWith("+++")) return `<span class="add">${line}</span>`;
      if (line.startsWith("-") && !line.startsWith("---")) return `<span class="del">${line}</span>`;
      return line;
    })
    .join("\n");
}

async function applySelectedChanges() {
  const selected = [...document.querySelectorAll("[data-change-index]:checked")]
    .map((input) => state.changes[Number(input.dataset.changeIndex)])
    .filter(Boolean);
  if (!selected.length) return;
  const result = await api("/changes/apply", {
    method: "POST",
    ...body({ changes: selected, confirm_delete: els.confirmDelete.checked })
  });
  addMessage("assistant", `Applied: ${result.applied.length}\nSkipped: ${result.skipped.length}`);
  state.changes = result.skipped || [];
  renderChanges();
  await refreshTree();
  if (state.activePath) {
    const active = state.activePath;
    state.tabs.delete(active);
    await openFile(active).catch(() => activateTab(null));
  }
}

function refreshActiveTreeMarkers() {
  document.querySelectorAll(".tree-node").forEach((node) => {
    node.classList.toggle("active", node.dataset.path === state.activePath);
  });
}

function setStatus(text) {
  els.editorStatus.textContent = text || "";
}

async function runTask(fn) {
  try {
    await fn();
  } catch (err) {
    setStatus("Error");
    addMessage("assistant", err.message);
  }
}

init().catch((err) => {
  setStatus("Error");
  addMessage("assistant", err.message);
});
