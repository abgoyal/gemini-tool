# Architecture Decision Record (ADR)

This document records the architectural decisions made for the Gemini LAN Tool.

## 1. Backend Technology: Go with Standard Library
- **Decision:** Use Go with its standard `net/http` package for the backend server.
- **Rationale:**
    - **Simplicity & Performance:** Go is compiled, fast, and has low memory overhead, making it ideal for running on low-powered devices like a Raspberry Pi.
    - **Single Binary Deployment:** Go excels at creating statically linked binaries. This fulfills the core requirement of easy deployment.
    - **Concurrency:** Go's built-in concurrency model is excellent for handling HTTP requests efficiently without a heavy framework.
- **Alternatives Considered:** Python (Flask/FastAPI), Node.js (Express). Rejected due to runtime dependencies which complicate deployment.

## 2. Frontend Technology: Svelte 5 with Runes
- **Decision:** Use Svelte 5 and its modern, Rune-based reactivity model.
- **Rationale:**
    - **No Runtime Overhead:** Svelte is a compiler that generates highly optimized, vanilla JavaScript, resulting in a fast and lightweight UI.
    - **Explicit Reactivity:** Svelte 5's Rune-based API (`$state`, `$derived`) is explicit, powerful, and easy to reason about for state management.
    - **Build-time Compilation:** Aligns perfectly with the goal of embedding the final UI assets into the Go binary.
- **Alternatives Considered:** React, Vue. Rejected due to their larger runtime footprints.

## 3. Asset Handling: Go `embed` package
- **Decision:** The compiled SvelteKit static assets (`ui/build` directory) are embedded directly into the Go binary at compile time.
- **Rationale:** This is the cornerstone of the single-binary deployment strategy. It removes the need to manage and serve separate static files, simplifying deployment to copying and running one file.

## 4. Database: SQLite with Automated Migration
- **Decision:** Use SQLite for all data storage. The application itself will handle schema migrations and data backfilling on startup.
- **Rationale:**
    - **Serverless & Portable:** SQLite requires no separate server process. The database is a single, easily manageable file.
    - **Automated Upgrades:** By building migration logic into the Go application's startup sequence, we provide a seamless user experience. Users can replace the old binary with a new one, and the application will automatically update the database schema and backfill data as needed, preventing any data loss from previous versions. This is critical for a tool that stores valuable history.
- **Alternatives Considered:** External migration tools (e.g., `golang-migrate/migrate`). Rejected to maintain the "zero external dependencies at runtime" principle and to simplify the user's upgrade path.

## 5. Build Process: Unified via `go generate`
- **Decision:** Use Go's built-in `go generate` directive to orchestrate the build process.
- **Rationale:**
    - **Developer Experience:** This creates a single, idiomatic Go command (`go generate ./... && go build ...`) to build the entire application, including the frontend assets.
    - **Reliability:** It ensures the UI is always built with the latest source code before the Go binary is compiled, preventing stale assets from being embedded.
    - **Simplicity:** It avoids the need for external build scripts (like Makefiles or shell scripts) for a simple project, keeping the process contained within the Go toolchain.

