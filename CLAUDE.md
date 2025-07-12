# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**ThrowBackChat** is an IRC-style chat application with a Go backend and SolidJS frontend. It features real-time messaging via WebSockets, channel-based conversations, and operator permissions.

## Development Commands

- `make dev` - Start development environment if it's not yet running (runs both backend and frontend with live reload)
- `make install` - Install dependencies for both Go and Node.js
- `make build` - Build the Go server binary to `bin/server`
- `make format` - Format code using Go fmt and Prettier
- `make check` - Run static analysis (`go vet`) and TypeScript type checking
- `make tail-log` - View development logs (last 100 lines, ANSI stripped)

**Important:**

- never start the server in any other way than with `make dev`.
- the server and the client are logging into dev.log which you can look at with `make tail`
- if you are unsure, check the log

## Architecture

### Backend (Go)
- **Entry point**: `cmd/server/main.go` - Chi router with middleware
- **Packages**:
  - `internal/chat/` - Chat management logic
  - `internal/db/` - Database operations and migrations
  - `internal/models/` - Data models (User, etc.)
  - `internal/utils/` - Utility functions
  - `internal/web/` - Web layer (HTTP handlers, WebSocket management)
- **Database**: SQLite with migration system in `internal/db/migrations/`
- **Dependencies**: Chi router, Gorilla WebSocket, godotenv, modernc.org/sqlite, jmoiron/sqlx

### Frontend (SolidJS)
- **Location**: `web/` directory
- **Tech**: SolidJS + TypeScript, Vite build, TailwindCSS v4
- **Development**: Vite dev server proxies `/api` and `/ws` to Go backend
- **Dependencies**: SolidJS, SolidJS Router, marked for markdown rendering, Prettier

### Database Schema
Implemented in `internal/db/migrations/0001_initial.sql`:
- `users` - User accounts (id, nickname, is_serv flag)
- `channels` - Chat channels (id, name, topic)
- `messages` - Chat messages (id, channel_id, user_id, sent_at, message, is_passive, event, nickname)
- `ops` - Operator permissions (user_id, channel_id, granted_by, granted_at)
- `migrations` - Migration tracking (id, filename, applied_at)

## Configuration

Environment variables (use `.env` file):
- `TBCHAT_PORT` - Server port (default: 8080)
- `TBCHAT_HOST` - Server host (default: 0.0.0.0)
- `TBCHAT_DB` - SQLite database path (default: chat.db)

## WebSocket Protocol

**Commands**: `/kick`, `/topic`, `/join`, `/leave`, `/me`, `/nick`

**Message Types**:
- `cmd` + `req_id` - Client requests
- `type: response` + `req_id` + `okay` - Server responses  
- `type: message` - Chat messages with markdown support
- `type: event` - Channel/server events (joined, left, nick_change, etc.)

## Session Management

- Users take possession of nickname on join (no signup)
- Sessions require heartbeat every minute (5 missed = disconnect)
- Channel subscriptions maintained in memory (restored on reconnect)
- New channels: first joiner becomes operator

## Development Setup

Uses `shoreman.sh` (Procfile runner) to manage both frontend and backend processes simultaneously. Frontend runs on Vite dev server with API proxy, backend serves on localhost:8080.
