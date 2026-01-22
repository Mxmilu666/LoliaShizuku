# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

LoliaShizuku is a desktop application built with [Wails v2](https://wails.io/) - a framework for building native desktop apps using Go backend and Vue 3 frontend. The app uses TypeScript, Vite for the frontend build tool, and Bun as the package manager.

## Development Commands

### Running the Application
- **Live development mode**: `wails dev` - Runs Vite dev server with hot reload; also starts a dev server at http://localhost:34115 for browser-based development with Go method access
- **Production build**: `wails build` - Creates a redistributable production binary

### Frontend-Only Commands
Run these from the `frontend/` directory:
- **Install dependencies**: `bun install`
- **Vite dev server**: `bun run dev`
- **Build frontend**: `bun run build` - Runs `vue-tsc --noEmit` for type checking then `vite build`
- **Preview build**: `bun run preview`

## Architecture

### Backend (Go)
- **Entry point**: `main.go` - Initializes Wails app with embedded frontend assets
- **Backend package**: `backend/app.go` - Contains the `App` struct with exported methods that are callable from the frontend
- **Context**: The `App` struct holds a `context.Context` for calling Wails runtime methods
- **Exported methods**: Any exported method on the `App` struct (e.g., `Greet`) is automatically bound to the frontend via Wails

### Frontend (Vue 3 + TypeScript)
- **Framework**: Vue 3 with Composition API (`<script setup>`)
- **Build tool**: Vite
- **Entry point**: `frontend/src/main.ts`
- **Root component**: `frontend/src/App.vue`
- **Type checking**: `vue-tsc` for TypeScript validation

### Go-Frontend Bridge
Wails automatically generates TypeScript bindings for Go backend methods:
- **Generated bindings**: Located in `frontend/wailsjs/go/backend/`
- **Type definitions**: `.d.ts` files are auto-generated (do not edit manually)
- **Import example**: `import {Greet} from '../../wailsjs/go/main/App'` (note: uses `main` in path, not `backend`)

When you add new exported methods to the `App` struct in `backend/app.go`, Wails automatically regenerates the TypeScript bindings during development.

### Frontend Assets
- The production frontend build is embedded into the Go binary using `//go:embed all:frontend/dist` in `main.go`
- Built output goes to `frontend/dist/`

## Key Files

- `wails.json` - Wails project configuration (author info, build settings, frontend commands)
- `go.mod` - Go module dependencies
- `frontend/package.json` - Node.js dependencies and scripts
- `frontend/vite.config.ts` - Vite configuration
- `frontend/tsconfig.json` - TypeScript configuration

## Development Notes

- Bun is used as the package manager (configured in `wails.json`)
- The app window is configured for 1024x768 with a dark background color (#1B2636)
- All Go methods exported from the `App` struct are automatically available to the frontend as async functions that return Promises
