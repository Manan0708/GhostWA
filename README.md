# GhostWA v2.5 (Stealthy WhatsApp Terminal Suite & TUI Dashboard)

GhostWA is a high-performance, developer-focused terminal client and stealthy background messaging suite for WhatsApp. Engineered in Go, GhostWA leverages a decoupled **client-server architecture** using a persistent background daemon, a local SQLite database running in **WAL (Write-Ahead Logging)** mode, a TCP-based JSON IPC protocol, and interactive terminal interfaces built with Charm's **Bubble Tea** and **Lip Gloss** frameworks.

---

## ⚡ 1-Line Quick Installation (No Go Required!)

You can install and set up GhostWA on any machine with a **single command**. No programming tools or Go installations required!

### Windows (PowerShell)
Paste this into PowerShell or VS Code Terminal:
```powershell
irm https://raw.githubusercontent.com/Manan0708/GhostWA/main/install.ps1 | iex
```

### Linux / macOS (Bash)
Paste this into Terminal:
```bash
curl -fsSL https://raw.githubusercontent.com/Manan0708/GhostWA/main/install.sh | sh
```

Once installed, simply run `ghostwa login` to scan your QR code and get started!

---

## Table of Contents
1. [Architecture & Flow](#architecture--flow)
2. [Database Schema & Multi-Process Concurrency](#database-schema--multi-process-concurrency)
3. [Deep Dive: Key Subsystems](#deep-dive-key-subsystems)
   * [LID to PN (Phone Number) Merging](#lid-to-pn-phone-number-merging)
   * [Background Daemon Service (IPC Protocol)](#background-daemon-service-ipc-protocol)
   * [Real-time Right-Aligned Notification System](#real-time-right-aligned-notification-system)
   * [Automated Media Downloader & Smart Uploader](#automated-media-downloader--smart-uploader)
4. [TUI Dashboard Architecture (`wacli show`)](#tui-dashboard-architecture-wacli-show)
5. [CLI & Interactive Command Reference](#cli--interactive-command-reference)
6. [Manual Build & Installation](#manual-build--installation)
7. [Troubleshooting & Developer Guidelines](#troubleshooting--developer-guidelines)

---

## Architecture & Flow

To avoid blocking the shell during authentication or large network synchronizations, WACLI splits operations between a foreground client and a background daemon:

```
+-----------------------------------------------------------------------------------+
|                                 WhatsApp Servers                                  |
+------------------------------------------▲----------------------------------------+
                                           │
                                           │ (Secure WebSocket Session)
                                           ▼
+-----------------------------------------------------------------------------------+
|                             Background Daemon (Server)                            |
|                                                                                   |
|  [whatsmeow Client]                                                                |
|      ├── Event Handlers (Messages, Read Receipts, History Syncs)                  |
|      └── Media Downloader/Uploader                                                |
|                                                                                   |
|  [SQLite Database] <── (WAL Mode - Concurrent Write & Read Access)                |
|      ├── Chats & Contacts Metadata                                                |
|      └── Full Message Logs                                                        |
+------------------------------------------▲----------------------------------------+
                                           │
                                           │ (TCP Loopback Socket - Port 9090)
                                           ▼
+-----------------------------------------------------------------------------------+
|                              CLI & TUI Clients                                    |
|                                                                                   |
|  [Interactive Dashboard] (wacli show)     [Interactive Chat Loop] (wacli open)    |
|      └── Bubble Tea TUI                       └── Command-line session            |
|                                                                                   |
|  [Single Commands]                                                                |
|      └── wacli chats, wacli groups, wacli send, wacli search, etc.                |
+-----------------------------------------------------------------------------------+
```

1. **WhatsApp Server Handshake**: The background daemon runs a WebSocket client wrapping the `whatsmeow` library.
2. **TCP Loopback IPC**: The daemon listens on local TCP port `127.0.0.1:9090`. Clients subscribe to live updates or send commands via structured JSON-line payloads.
3. **Shared Storage (WAL Mode)**: All historical data, cache, and metadata are committed to a local SQLite database. By running in WAL mode, database access is fully non-blocking, ensuring TUI rendering is fast even while the background daemon writes incoming message bursts.

---

## Database Schema & Multi-Process Concurrency

WACLI stores all local records in a SQLite database file located at `~/.local/share/wacli/messages.db`. 

### The Schema Definition

```sql
-- Contacts Table: Caches contact metadata resolved from WhatsApp directories
CREATE TABLE IF NOT EXISTS contacts (
    jid TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    phone TEXT NOT NULL,
    push_name TEXT,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Chats Table: Stores conversation summaries and real-time unread statistics
CREATE TABLE IF NOT EXISTS chats (
    jid TEXT PRIMARY KEY,
    name TEXT,
    unread_count INTEGER DEFAULT 0,
    last_message_time DATETIME,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Messages Table: Full message logs, including texts and media file pointers
CREATE TABLE IF NOT EXISTS messages (
    id TEXT PRIMARY KEY,
    chat_jid TEXT,
    sender_jid TEXT,
    content TEXT,
    timestamp DATETIME,
    is_from_me INTEGER,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(chat_jid) REFERENCES chats(jid)
);

-- Full-Text Performance Index
CREATE INDEX IF NOT EXISTS idx_messages_content ON messages(content);
```

### Concurrency with WAL (Write-Ahead Logging)
Standard SQLite locks the database file whenever a write operation is in progress, blocking concurrent readers and throwing `database is locked` errors. WACLI initializes the database using WAL journal mode:
```go
if _, err := db.Exec("PRAGMA journal_mode = WAL;"); err != nil {
    return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
}
```
In WAL mode, updates are appended to a separate `-wal` write-ahead log file. Readers can read original pages from the database file while the daemon concurrently writes to the log, allowing simultaneous reads and writes without blocking.

---

## Deep Dive: Key Subsystems

### LID to PN (Phone Number) Merging
On modern WhatsApp protocols, users are represented by two JIDs:
1. **LID (LID JID, suffix `@lid`)**: Private identifiers used internally by WhatsApp to link accounts.
2. **PN (Phone Number JID, suffix `@s.whatsapp.net`)**: Public phone number identifiers.

This split can lead to chat duplication (e.g., a contact appearing in `wacli chats` once as their phone number and again as a raw LID). WACLI resolves this using a mapping merge algorithm:

1. **Resolution Cache**: The daemon queries WhatsApp's contact store to resolve the Phone Number (PN) corresponding to the incoming LID JID.
2. **Transaction Database Merge**:
   * If the corresponding PN chat does not exist in the database, WACLI inserts a new PN row.
   * WACLI merges the unread badge counts:
     `UPDATE chats SET unread_count = unread_count + LID_unread WHERE jid = PN_JID`
   * Re-links message histories by updating constraints:
     `UPDATE messages SET chat_jid = PN_JID WHERE chat_jid = LID_JID`
     `UPDATE messages SET sender_jid = PN_JID WHERE sender_jid = LID_JID`
   * Safely purges the duplicate LID entries from the `chats` and `contacts` tables.

This guarantees a consolidated chat log.

### Background Daemon Service (IPC Protocol)
All network-dependent CLI commands serialize queries into a standard JSON payload containing a trailing newline (`\n`), sending it via a TCP socket to `127.0.0.1:9090`.

#### Request Schema:
```json
{
  "type": "send",
  "to": "919588003434@s.whatsapp.net",
  "body": "Hello World!"
}
```

#### Response Schema:
```json
{
  "success": true,
  "msg_id": "3EB0A123456789ABCDEF",
  "error": ""
}
```

When a client runs `subscribe`, the daemon establishes a persistent socket connection and pipes real-time events (`"message"`, `"receipt"`, or `"conn_status"`) as they arrive.

### Real-time Right-Aligned Notification System
To prevent background chat noise from cluttering active conversations, WACLI splits terminal displays:
* The active text flow is printed on the left side of the terminal.
* Background notifications are pushed to the far-right margin.

To achieve this, the client queries standard output dimensions dynamically:
```go
width, _, err := term.GetSize(int(os.Stdout.Fd()))
```
It computes the width of the minimal notification format (e.g. `🔔 [Raj Mehra]`), accounts for double-width emoji cells (`🔔`), pads the remainder of the line with spaces, and renders:
```text
[18:31] You: Let's meet at 7 PM.
[18:32] Raj Mehra: Perfect, see you!                                           🔔 [Mumbai Region]
> 
```

### Automated Media Downloader & Smart Uploader

#### Background Media Downloads:
When the daemon receives a message event containing an image, video, document, or audio file:
1. It requests decryption keys and streams the binary blob using the `whatsmeow` client downloader.
2. The blob is saved locally inside a `./downloads/` directory.
3. The content field of the database message record is saved as a local reference: `📷 [Image received: downloads\IMG_20260831_184712.jpg]`.

#### Smart Path-Cleaned Uploads:
When sending media, you simply supply a local path to the `wacli send` command:
```powershell
wacli send "Aryan" "D:\photos\vacation.jpg"
```
WACLI automatically:
1. Strips any surrounding quotes (e.g., `"D:\photos\vacation.jpg"`) that may have been generated by Windows "Copy as path".
2. Examines the file extension to assign correct mime-types and Whatsmeow constants (e.g. `whatsmeow.MediaImage`, `whatsmeow.MediaVideo`, or `whatsmeow.MediaDocument`).
3. Uploads the encrypted payload to WhatsApp's media servers and transmits the message.

---

## TUI Dashboard Architecture (`wacli show`)

The dashboard is designed as a full-terminal interface utilizing the **Model-View-Update (MVU)** loop:

```
        ┌────────────────────────────────────────────────────────┐
        │                        Update                          │
        │  (Processes keys, ticks, or incoming TCP messages,     │
        │   updates state, and triggers UI repaint)               │
        └─────────────▲────────────────────────────┬─────────────┘
                      │                            │
                   (Events)                    (New Model)
                      │                            │
        ┌─────────────┴──────────┐     ┌───────────▼─────────────┐
        │        Terminal        │     │          Model          │
        │      User Input        │     │   (App State & Logs)    │
        └────────────────────────┘     └───────────┬─────────────┘
                                                   │
                                                (Render)
                                                   │
                                       ┌───────────▼─────────────┐
                                       │          View           │
                                       │    (Renders Columns     │
                                       │     with Lip Gloss)     │
                                       └─────────────────────────┘
```

* **Bubble Tea Loop**: Listens for background updates from the daemon TCP subscription. When a message is received, it triggers a `reloadChats()` and `loadMessages()` query, refreshing list boxes instantly.
* **Responsive Rendering (`lipgloss`)**: Automatically splits the terminal screen into a left column (list of sorted chats with unread badges) and a right column (active chat window, scrollable text history, and text input block).
* **WAL Database Safety**: Because the dashboard reads from `messages.db` on a 1-second tick to update unread counts, running in WAL mode ensures zero collision warnings or lockups.

---

## CLI & Interactive Command Reference

### Global Commands

| Command | Usage | Description |
| :--- | :--- | :--- |
| `login` | `wacli login` | Starts authentication; prints a terminal-friendly QR code. |
| `logout` | `wacli logout` | Terminates active sessions, drops cache, and wipes stored login keys. |
| `status` | `wacli status` | Reports daemon connection status, login profile JID, and server logs. |
| `daemon` | `wacli daemon <start\|stop\|restart\|status>` | Starts/stops the windowless background service process. |
| `chats` | `wacli chats` | Lists direct conversations sorted by active timestamps with unread counts. |
| `groups` | `wacli groups` | Lists group conversations, resolving group names from cache. |
| `show` | `wacli show` | Opens the dual-column, interactive Bubble Tea terminal user interface. |
| `open` | `wacli open <recipient>` | Launches a text-based console loop for live, focused texting. |
| `send` | `wacli send <recipient> <text\|path>` | Transmits plain text or uploads a media file (images, PDFs, videos). |
| `search` | `wacli search <query>` | Case-insensitive full-text search across contacts, group names, and text logs. |
| `commands`| `wacli commands` | Prints the reference table of all commands. |

### Subcommands Inside Interactive Chat (`wacli open`)

While in an active console stream (`wacli open <name>`), type these commands directly into the prompt:

| Command | Usage | Description |
| :--- | :--- | :--- |
| `/history` | `/history [limit]` | Reloads older text logs (default is last 10 messages). |
| `/search` | `/search <query>` | Case-insensitive text search filtering message logs for this chat. |
| `/media` | `/media` | Displays a list of all exchanged media files with local file paths. |
| `/mute` | `/mute` | Silences background chat alerts during your active session. |
| `/unmute` | `/unmute` | Enables background chat alerts. |
| `/alerts` | `/alerts` | Lists pending unread messages received from other chats. |
| `/exit` | `/exit` or `/quit` | Closes the active session and returns to terminal. |

---

## Manual Build & Installation

If you are a developer and prefer building from source code:

### Prerequisites
* **Go Compiler**: Go 1.20 or newer.

### Build Steps
1. Clone the repository and compile:
   ```powershell
   git clone https://github.com/Manan0708/Whatsapp-CLI.git
   cd Whatsapp-CLI
   go install ./cmd/wacli
   ```
2. Authenticate session:
   ```powershell
   wacli login
   ```
3. Spawn background service:
   ```powershell
   wacli daemon start
   ```

---

## Troubleshooting & Developer Guidelines

### Standard Error Profiles

#### HTTP Status Code 415 (Unsupported Media Type)
* **Cause**: WACLI passed incorrect string categories (e.g., `"image"`, `"video"`) during uploads.
* **Resolution**: The code uses official SDK constants: `whatsmeow.MediaImage`, `whatsmeow.MediaVideo`, or `whatsmeow.MediaDocument`. Ensure new media formats are mapped to these constants.

#### Locked Executables during compilation (`go install`)
* **Cause**: Windows locks the executable binary (`wacli.exe`) if the background daemon is running.
* **Resolution**: Stop the daemon, install the updates, and restart it:
  ```powershell
  wacli daemon stop
  go install ./cmd/wacli
  wacli daemon start
  ```

#### Missing Incoming Messages / Event Stop
* **Cause**: Receiving protocol/action packets (like reactions or status notices) with a `nil` message body causes a panic in the whatsmeow callback.
* **Resolution**: Ensure all handlers contain safety checks at the very entry points:
  ```go
  if v.Message == nil {
      return
  }
  ```

### Developer Logs
All background process activities are logged inside:
* **Windows**: `C:\Users\<User>\.local\share\wacli\daemon.log`
* **Linux/macOS**: `~/.local/share/wacli/daemon.log`
