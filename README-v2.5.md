# ⚡ GhostWA v2.5+ Developer & Architecture Documentation

Welcome to **GhostWA v2.5+**, a high-performance, stealthy, terminal-native WhatsApp client built in Go. This document highlights all architectural upgrades, new features, database schemas, and system improvements introduced in the v2.5 series.

---

## 🚀 1. Key Improvements & Added Functionalities

### 📱 Dual Device Linking & Authentication
* **QR Code Terminal Rendering**: Instant ASCII QR code rendering in your terminal using `whatsmeow` Noise protocol handshakes.
* **8-Digit Phone Pairing Code**: Link your WhatsApp account using your phone number without scanning a QR code.

### 🎨 Cyberpunk TUI Dashboard (`ghostwa show`)
* **Sliding Viewport Scrolling**: Responsive sidebar that scales dynamically to any terminal window height without breaking layout boundaries.
* **Rune-Safe UTF-8 Rendering**: Full support for emojis, special unicode characters, and long contact titles without string corruption.
* **Real-time Event Streaming**: Inter-process communication via TCP sockets streams incoming messages and status changes directly into the UI without polling.

### 💬 Messaging, Media & Reactions
* **Single & Multi-line Support**: Press `Shift+Enter`, `Alt+Enter`, `Ctrl+J`, or end lines with `\` + `Enter` for multi-line drafting.
* **Message Reactions**: React to messages directly in TUI using `/react <emoji>` or via CLI using `ghostwa react <chat> <emoji>`.
* **Media File Transmission**: Auto-detects images (`.jpg`, `.png`), videos (`.mp4`), GIFs, and documents sent by file path.

### 🧹 Clean Logout & Manual Sync Engine
* **Complete Process & File Lock Purge**: `ghostwa logout` force-kills daemon background processes, closes SQLite handles, and purges session storage.
* **Manual Chat Repair**: `ghostwa sync` or `ghostwa sync chats` rebuilds local chat tables directly from raw message logs.

---

## 🏗️ 2. Architectural Design

GhostWA uses a decoupled **Client-Daemon Architecture**. A lightweight background daemon maintains the WebSocket connection to WhatsApp servers, while ephemeral CLI commands and the TUI communicate with the daemon over local IPC.

```
┌─────────────────────────────────────────────────────────────┐
│                     GhostWA CLI / TUI                       │
│  (ghostwa show / ghostwa send / ghostwa sync / ghostwa react)│
└──────────────────────────────┬──────────────────────────────┘
                               │ TCP Socket IPC (Port 42069)
┌──────────────────────────────▼──────────────────────────────┐
│                   GhostWA Background Daemon                 │
│       (whatsmeow Engine, Event Router, SQLite Pool)          │
└───────────────┬───────────────────────────────┬─────────────┘
                │                               │
┌───────────────▼───────────────┐ ┌─────────────▼─────────────┐
│  WhatsApp Noise Protocol WS   │ │   Local SQLite Storage    │
│    (web.whatsapp.com:443)     │ │ (messages.db & session.db)│
└───────────────────────────────┘ └───────────────────────────┘
```

### IPC Protocol (JSON over TCP Socket)
The daemon listens on `127.0.0.1:42069`. Requests and responses are newline-delimited JSON payloads.

**Example Request:**
```json
{
  "type": "send",
  "to": "919876543210@s.whatsapp.net",
  "body": "Hello from GhostWA v2.5!"
}
```

**Example Response:**
```json
{
  "success": true,
  "msg_id": "3EB0ABC123456789"
}
```

---

## 🗄️ 3. Database Schema (`messages.db`)

GhostWA uses **SQLite** (via `modernc.org/sqlite` CGO-free driver) stored in `~/.local/share/wacli/messages.db`.

### `chats` Table
Stores active direct and group conversations.
```sql
CREATE TABLE IF NOT EXISTS chats (
    jid TEXT PRIMARY KEY,
    name TEXT,
    unread_count INTEGER DEFAULT 0,
    last_message_time DATETIME,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### `messages` Table
Persists historical and real-time chat messages.
```sql
CREATE TABLE IF NOT EXISTS messages (
    id TEXT PRIMARY KEY,
    chat_jid TEXT,
    sender_jid TEXT,
    content TEXT,
    media_url TEXT,
    media_type TEXT,
    mimetype TEXT,
    file_size INTEGER DEFAULT 0,
    reaction TEXT DEFAULT '',
    timestamp DATETIME,
    is_from_me BOOLEAN,
    FOREIGN KEY(chat_jid) REFERENCES chats(jid)
);
```

### `contacts` Table
Maps JIDs to resolved push names and friendly contact names.
```sql
CREATE TABLE IF NOT EXISTS contacts (
    jid TEXT PRIMARY KEY,
    name TEXT,
    push_name TEXT,
    phone_number TEXT
);
```

---

## 🛠️ 4. Quick Command Reference

| Command | Description |
|---|---|
| `ghostwa login` | Link WhatsApp via QR Code or Phone Pairing Code |
| `ghostwa logout` | Unlink device, terminate daemon, and clear session data |
| `ghostwa show` | Launch interactive Cyberpunk TUI Dashboard |
| `ghostwa sync` | Manually sync and rebuild chat list database |
| `ghostwa send <to> <msg>` | Send text or media file from CLI |
| `ghostwa react <to> [msg_id] <emoji>` | Send reaction (`❤️`, `👍`, `🔥`) to a message |
| `ghostwa status` | Check connection and daemon background status |
| `ghostwa chats` | Output list of active direct chats |
| `ghostwa groups` | Output list of active group chats |

---

## ⚡ 5. Installation

```powershell
irm https://raw.githubusercontent.com/Manan0708/GhostWA/main/install-v2.5.ps1 | iex
```
