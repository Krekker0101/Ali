# Ollama IDE Layer

This package adds an isolated AI-powered IDE surface without changing the existing
Ollama inference API. It mounts a browser UI at `/ide` and new IDE-only routes
under `/api/v1/ide`.

## Boundaries

- Existing `/api/*` and `/v1/*` routes are not changed.
- Workspace filesystem access is constrained to the opened project root.
- Delete operations require explicit confirmation.
- Agent output is converted into `FileChange` objects with a diff preview before
  any patch can be applied.
- The IDE API is limited to loopback clients by default. Set
  `OLLAMA_IDE_ALLOW_REMOTE=true` only for trusted deployments.

## Core Capabilities

- Open a project folder.
- Browse a recursive file tree.
- Read, create, write, and delete files through safe tools.
- Search inside the project.
- Edit files with tabs, in-file search, syntax highlighting, and save support.
- Switch between local Ollama models and OpenAI-compatible cloud providers.
- Configure dark, light, and custom color themes.
- Persist non-secret IDE settings between launches.
- Run a multi-step AI agent that inspects the project through tools, prepares
  controlled file changes, and shows diffs before applying them.
- Report release/smoke-test readiness through `/api/v1/ide/health`.

## Agent Mode

The agent runs as a tool loop instead of a single chat completion:

1. Seeds itself with the project tree and selected/open file context.
2. Calls safe tools such as `read_file`, `list_directory`, `project_tree`, and
   `search_project` when it needs more context.
3. Uses `prepare_change`, `write_file`, `create_file`, `delete_file`, or
   `apply_patch` only to produce pending `FileChange` objects.
4. Returns a trace of each round and a final diff set.
5. Applies nothing until the user explicitly approves the proposed changes.

`write_file` and `create_file` are intentionally non-mutating in the agent tool
layer and in `/api/v1/ide/tools/*`: they prepare reviewable diffs. `delete_file`
prepares a delete diff and still requires user confirmation when changes are
applied.

## Main Routes

```text
GET    /ide
GET    /api/v1/ide/health
GET    /api/v1/ide/workspace
POST   /api/v1/ide/workspace/open
GET    /api/v1/ide/tree
GET    /api/v1/ide/files/list
GET    /api/v1/ide/files/read
PUT    /api/v1/ide/files/write
POST   /api/v1/ide/files/create
POST   /api/v1/ide/files/delete
GET    /api/v1/ide/search
GET    /api/v1/ide/settings
PUT    /api/v1/ide/settings
GET    /api/v1/ide/models
POST   /api/v1/ide/agent/run
POST   /api/v1/ide/changes/apply
POST   /api/v1/ide/tools/read_file
POST   /api/v1/ide/tools/write_file
POST   /api/v1/ide/tools/create_file
POST   /api/v1/ide/tools/delete_file
POST   /api/v1/ide/tools/list_directory
POST   /api/v1/ide/tools/search_project
POST   /api/v1/ide/tools/apply_patch
```
