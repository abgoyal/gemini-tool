
===================
Gemini LAN Tool
===================

A simple, self-hosted, Go-based web tool for interacting with the Google Gemini API.
It is designed for LAN usage, requires no authentication, and is deployed as a single,
self-contained binary.

Features:
- Web UI to select saved prompts and provide user input.
- Combines prompts and input, sends to Gemini API, and displays the Markdown-rendered output.
- Stores all prompts and chat history in a local SQLite database.
- Dedicated page for creating, editing, and cloning custom prompts.
- Displays detailed chat history which can be revisited and re-run with different models.
- UI built with Svelte 5, embedded directly into the Go binary.

-------------------
How to Build
-------------------

Prerequisites:
- Go (version 1.22+)
- Node.js and npm (for building the UI)
- A Google Gemini API Key

This project uses Go's `generate` tool to combine the UI and backend builds into a single, streamlined process.

**Step 1: Install UI Dependencies (Only needs to be done once)**
If this is your first time building, you must install the Node.js packages first.

    cd ui
    npm install
    cd ..

**Step 2: Build the Application (The One Command to Rule Them All)**
From the project's root directory, run the following command. This will first build the Svelte UI and then compile the Go application, embedding the newly built UI into the final binary.

**On Linux/macOS:**

    go generate . && go build -o gemini-tool .

**On Windows (Command Prompt):**

    go generate .
    go build -o gemini-lan-tool.exe .

This will create a single executable file named `gemini-lan-tool` (or `gemini-lan-tool.exe`) in the project root.

-------------------
How to Run
-------------------

1.  **Set Environment Variable:**
    You must provide your Gemini API key via the `GEMINI_API_KEY` environment variable.

    On Linux/macOS:
    export GEMINI_API_KEY="YOUR_API_KEY_HERE"

    On Windows (Command Prompt):
    set GEMINI_API_KEY="YOUR_API_KEY_HERE"

2.  **Execute the Binary:**
    Run the compiled application from your terminal.

    ./gemini-lan-tool

    The server will start, and a `gemini-tool.db` SQLite file will be created in the
    same directory. Any necessary database migrations will be applied automatically.

3.  **Access the Web UI:**
    Open your web browser and navigate to `http://localhost:8080`.

-------------------
API Details (for direct access)
-------------------

The server exposes a simple JSON API.

**GET /api/models**
Returns a list of supported Gemini models.
$ curl http://localhost:8080/api/models

**GET /api/prompts**
Returns a list of all saved prompts.
$ curl http://localhost:8080/api/prompts

**POST /api/prompts**
Creates a new prompt.
$ curl -X POST -H "Content-Type: application/json" \
  -d '{"name":"Summarizer","prompt_text":"Summarize the following text:","separator":"---","model":"models/gemini-1.5-flash"}' \
  http://localhost:8080/api/prompts
  
**PUT /api/prompts**
Updates an existing prompt.
$ curl -X PUT -H "Content-Type: application/json" \
  -d '{"id":1, "name":"My Updated Summarizer","prompt_text":"Please summarize this:","separator":"---","model":"models/gemini-1.5-pro"}' \
  http://localhost:8080/api/prompts

**GET /api/chats**
Returns the entire chat history.
$ curl http://localhost:8080/api/chats

**POST /api/generate**
Submits a request to the Gemini model. Can optionally override the model.
$ curl -X POST -H "Content-Type: application/json" \
  -d '{"prompt_id":1,"user_input":"This is the text to summarize.", "model":"models/gemini-1.5-pro"}' \
  http://localhost:8080/api/generate

