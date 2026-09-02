# ⚡ GhostWA v2.5+ Exhaustive Technical Compendium (64 Questions & Answers)

> This compendium provides an exhaustive, line-by-line technical breakdown covering GhostWA's system architecture, codebase design, database schemas, terminal UI rendering mechanics, IPC protocols, and underlying technologies.

---

### Q1. [1. Package Management, Ecosystem & Environment] What is a Package Manager in software engineering?

A Package Manager is an automated software utility that manages a project's external code libraries, software packages, and binary dependencies. It automates the fetching, compiling, configuring, locking, upgrading, and removing of third-party dependencies.

Key Functions:
1. Dependency Resolution: Calculates the dependency tree and resolves version conflicts between nested third-party packages.
2. Cryptographic Checksum Lockfiles: Maintains cryptographic hashes (e.g. go.sum, package-lock.json) to ensure reproducible builds across different developer environments and CI/CD pipelines.
3. Supply Chain Security: Prevents dependency confusion attacks and unauthorized code tampering.

---

### Q2. [1. Package Management, Ecosystem & Environment] Which Package Manager is used in GhostWA?

GhostWA uses Go Modules (go.mod and go.sum), which is the standard, native dependency management system introduced in Go 1.11 and made default in Go 1.13.

Why Go Modules was chosen for GhostWA:
• Zero External Tools Needed: Ships natively inside the Go toolchain (go build, go test, go get).
• Minimal Version Selection (MVS): Selects predictable, deterministic module versions without unexpected breaking upgrades.
• Checksum Database (go.sum): Stores SHA-256 hashes of every imported module, guaranteeing that the exact same source code is compiled across all developer machines.
• Direct Git Integration: Allows referencing modules directly from GitHub repositories (e.g., github.com/charmbracelet/bubbletea or go.mau.fi/whatsmeow) and tagging releases using semantic Git tags (v2.5.6).

---

### Q3. [1. Package Management, Ecosystem & Environment] What is the difference between go.mod and go.sum in Go?

• go.mod: The module definition file. It declares the module path (e.g. github.com/Manan0708/GhostWA), the minimum Go language version (e.g. go 1.22), and explicit direct and indirect dependency version requirements.
• go.sum: The checksum lockfile. It contains cryptographic SHA-256 hashes of the exact contents of every module version downloaded. It ensures that if a module repository is modified upstream, the local build will reject the modified source code due to checksum mismatch.

---

### Q4. [1. Package Management, Ecosystem & Environment] What are the historical and modern alternatives to Go Modules across different language ecosystems?

Alternatives across programming ecosystems:

1. Go Ecosystem (Historical):
   - GOPATH Mode: Monolithic single-folder workspace requiring all code to live in $GOPATH/src. (Obsolete since Go 1.13).
   - dep / glide / godep: Third-party package tools used before Go Modules was introduced.

2. Node.js / JavaScript / TypeScript:
   - npm (Node Package Manager)
   - yarn (Fast, lockfile-focused)
   - pnpm (Content-addressable storage, extremely space-efficient)

3. Python:
   - pip (Standard package installer)
   - uv (Extremely fast Rust-based Python package manager)
   - poetry / pipenv (Environment and lockfile managers)

4. Rust: cargo (Integrated compiler and package manager).
5. C# / .NET: NuGet.
6. Java: Maven, Gradle.

---

### Q5. [1. Package Management, Ecosystem & Environment] What is CGO in Go? Why is avoiding CGO crucial for Windows cross-compilation?

CGO is the feature in Go that allows Go packages to call C code directly using GCC or Clang compilers.

Why avoiding CGO is crucial for GhostWA:
• CGO Compiles to C Object Files: Building CGO packages on Windows requires installing GCC (MinGW-w64).
• Cross-Compilation Friction: Cross-compiling a CGO Go binary for Windows from Linux or macOS requires a full C cross-compiler toolchain.
• Static Binaries: Pure Go binaries (CGO_ENABLED=0) produce standalone, self-contained executables without DLL or C runtime dependencies. By using pure Go libraries (like modernc.org/sqlite), GhostWA builds instantly on any OS for Windows x64.

---

### Q6. [1. Package Management, Ecosystem & Environment] How does modernc.org/sqlite transpile C SQLite source code to pure Go?

modernc.org/sqlite uses a specialized C-to-Go transpiler called 'cznic/cc' (or 'wkc'). 

Instead of linking against the C shared library libsqlite3 via CGO, the C source code of SQLite (sqlite3.c) is converted line-by-line into pure, safe Go code. This allows GhostWA to run a full SQL database engine inside Go without needing any GCC compiler or CGO bindings.

---

### Q7. [1. Package Management, Ecosystem & Environment] What is the 'uv' package manager (referenced in AI skills) and how does it compare to standard 'pip'?

uv is an extremely fast Python package manager written in Rust by Astral (the creators of Ruff).

Comparison to pip:
• Speed: uv is 10x to 100x faster than standard pip because it is written in compiled Rust and uses parallel network requests and hardlink caching.
• Compatibility: Drop-in replacement for pip commands (uv pip install).
• Virtualenv: Built-in virtual environment management (uv venv).

---

### Q8. [1. Package Management, Ecosystem & Environment] How does the install-v2.5.ps1 PowerShell installer script work line-by-line?

install-v2.5.ps1 is GhostWA's one-line Windows installer script.

Step-by-Step Mechanics:
1. Security Protocol: Enforces TLS 1.2 via [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12.
2. Directory Setup: Creates %LOCALAPPDATA%\\Programs\\ghostwa and ~/.local/share/wacli.
3. Binary Download & Fallback: Tries downloading ghostwa-v2.5.6.exe using curl.exe first. If curl is unavailable, falls back to Invoke-WebRequest. Checks that downloaded file length is > 1MB.
4. PATH Environment Registration: Reads [Environment]::GetEnvironmentVariable("PATH", "User"). If the install directory is not present, appends it to the User PATH variable so 'ghostwa' runs from any terminal window.

---

### Q9. [1. Package Management, Ecosystem & Environment] How does the install.sh shell script work for Unix/macOS environments?

install.sh is the POSIX-compliant shell script installer for Linux and macOS. It uses curl/wget to fetch the compiled ghostwa binary, places it into /usr/local/bin or ~/.local/bin, makes it executable via 'chmod +x', and creates the data directory ~/.local/share/wacli.

---

### Q10. [1. Package Management, Ecosystem & Environment] Why are pre-compiled executable binaries stored in bin/ and tracked in Git via 'git add -f bin/*.exe'?

GhostWA's one-line installer script (install-v2.5.ps1) fetches the compiled executable directly from GitHub via raw.githubusercontent.com. 

Tracking versioned binaries (bin/ghostwa-v2.5.6.exe) in Git allows:
1. Instant end-user installation without requiring users to install Go or build from source.
2. No GitHub API Rate Limits: Downloading raw files bypasses GitHub REST API rate limits (which cap unauthenticated API requests to 60/hour).

---

### Q11. [2. Architecture & System Design] What is the Client-Daemon Architecture of GhostWA?

GhostWA is designed around a decoupled Client-Daemon Architecture:

1. GhostWA Background Daemon (internal/daemon): A long-running background server process (ghostwa daemon-run). It maintains the active, persistent WebSocket Noise connection to WhatsApp servers, manages SQLite database connection pools, and routes real-time events.
2. GhostWA CLI / TUI (internal/cli): Lightweight, short-lived commands (ghostwa show, ghostwa send, ghostwa sync, ghostwa react, ghostwa chats).

---

### Q12. [2. Architecture & System Design] Why did we decouple the daemon background process from the CLI/TUI?

Decoupling provides 4 major operational advantages:

1. Persistent Connection: WhatsApp Web requires an unbroken WebSocket connection. Reconnecting from scratch on every CLI command would take 5-10 seconds per command.
2. Instant Execution: CLI commands execute in under 50 milliseconds by talking to the active background daemon over local TCP IPC.
3. Instant Real-Time TUI Updates: Incoming WhatsApp messages are pushed instantly to ghostwa show without polling.
4. Robustness: The TUI can be opened, closed, or restarted at any time without dropping the WhatsApp WebSocket session.

---

### Q13. [2. Architecture & System Design] How does daemon startup sequence work (ConnectOrStartDaemon)?

When any CLI command runs:
1. ConnectOrStartDaemon() attempts to open a TCP connection to 127.0.0.1:42069.
2. If connection succeeds: It uses the active daemon connection immediately.
3. If connection fails (daemon not running): It silently spawns 'ghostwa daemon-run' as an independent background process using os/exec with detached process flags, waits up to 5 seconds for the daemon socket to accept connections, and then connects cleanly.

---

### Q14. [2. Architecture & System Design] What port does the daemon listen on (127.0.0.1:42069)? Why TCP over Unix Sockets or Named Pipes?

The daemon listens on 127.0.0.1:42069.

Why TCP Socket?
• Cross-Platform Parity: TCP loopback sockets (127.0.0.1) behave 100% identically across Windows, macOS, and Linux.
• Named Pipe / Unix Socket Complexity: Unix Domain Sockets do not exist natively on older Windows systems, and Windows Named Pipes require complex Windows API calls. TCP loopback provides clean, unified IPC across all operating systems.

---

### Q15. [2. Architecture & System Design] How does the JSON RPC protocol over TCP work in GhostWA?

Communication over the TCP socket uses newline-delimited JSON messages.

Structure:
• Request Payload: {"type": "send", "to": "...", "body": "...", "msg_id": "..."}
• Response Payload: {"success": true, "msg_id": "...", "error": "..."}
• Event Payload: {"type": "incoming_msg", "from": "...", "body": "...", "timestamp": "..."}

The client sends a JSON line ending in '\\n', and the daemon reads the bytes via bufio.NewReader(conn).ReadBytes('\\n') and unmarshals the payload.

---

### Q16. [2. Architecture & System Design] How does event subscription ('subscribe') and real-time event broadcasting work in server.go?

When ghostwa show launches:
1. It sends {"type": "subscribe"} over TCP to the daemon.
2. Server.addSubscriber(conn) registers the TCP connection in a thread-safe map (s.subscribers[conn] = true) protected by a sync.Mutex.
3. When an incoming message arrives via whatsmeow, s.broadcast(evt) iterates through all active subscribers and writes the JSON event payload asynchronously.

---

### Q17. [2. Architecture & System Design] What happens if the daemon crashes or is terminated while ghostwa show is running?

ghostwa show maintains an active TCP connection to the daemon in a background goroutine. If the daemon process is killed:
1. reader.ReadBytes('\\n') receives EOF or connection reset error.
2. The goroutine breaks cleanly.
3. The TUI continues displaying cached SQLite chat history, and displays '[OFFLINE]' status in the header badge.

---

### Q18. [2. Architecture & System Design] How do 'ghostwa daemon start', 'ghostwa daemon stop', and 'ghostwa daemon status' work?

• daemon start: Spawns 'ghostwa daemon-run' in background mode.
• daemon stop: Connects to 127.0.0.1:42069, sends {"type": "stop"}, and daemon closes DB handles and exits cleanly.
• daemon status: Queries the daemon for status ("connected", "disconnected", "not_logged_in") and active phone number.

---

### Q19. [3. WhatsApp Web Protocol & Whatsmeow Library] What is whatsmeow and why is it used instead of Puppeteer/Selenium web scraping?

whatsmeow (go.mau.fi/whatsmeow) is a pure Go implementation of the WhatsApp Web WebSocket protocol.

Why whatsmeow instead of Puppeteer/Selenium?
• Headless Web Scraping (Puppeteer/Selenium) consumes 500MB-1GB RAM, requires Chromium browser binaries, and is extremely slow.
• whatsmeow connects directly to WhatsApp WebSocket servers using native Go network sockets, consuming only 15MB-30MB RAM and running 100x faster!

---

### Q20. [3. WhatsApp Web Protocol & Whatsmeow Library] What is the Noise Protocol Framework in WhatsApp Web?

The Noise Protocol Framework is a crypto protocol framework used by WhatsApp for establishing encrypted communication channels.

Components:
1. Curve25519 for Diffie-Hellman ephemeral key exchange.
2. HKDF (HMAC-based Extract-and-Expand Key Derivation Function) for key derivation.
3. AES-256-GCM or ChaCha20-Poly1305 for symmetric payload encryption over WebSocket (web.whatsapp.com:443).

---

### Q21. [3. WhatsApp Web Protocol & Whatsmeow Library] How does device registration and session key storage work in session.db?

whatsmeow uses an underlying SQL store (go.mau.fi/whatsmeow/store/sqlstore) saved in ~/.local/share/wacli/session.db.

It stores:
1. Device identity keys and pre-keys.
2. Signal protocol session cipher keys for linked peer devices.
3. Server registration IDs and push secret tokens.

---

### Q22. [3. WhatsApp Web Protocol & Whatsmeow Library] How does WhatsApp handle End-to-End Encryption (E2EE) keys?

WhatsApp uses the Signal Protocol:
• Every message is encrypted with a unique pairwise message key derived from a ratchet key sequence.
• Even if one message key is compromised, past and future messages cannot be decrypted (Forward Secrecy & Break-in Recovery).
• Whatsmeow automatically manages Signal ratchets and decrypts incoming Protobuf payloads into raw message structs.

---

### Q23. [3. WhatsApp Web Protocol & Whatsmeow Library] What is HistorySync in WhatsApp Web protocol?

When a new device links to WhatsApp via QR code or Pairing Code, WhatsApp servers stream historical chats, recent messages, and contact lists over a series of compressed 'events.HistorySync' Protobuf payloads.

---

### Q24. [3. WhatsApp Web Protocol & Whatsmeow Library] How does GhostWA process events.HistorySync payloads when logging in?

In internal/whatsapp/events.go:
1. When *events.HistorySync is received, GhostWA triggers SyncStoreContacts().
2. Iterates through v.Data.GetConversations().
3. Resolves LIDs (Lightweight IDs) to Phone Number JIDs via c.ResolveLIDToPN().
4. Inserts chats via store.UpsertChat() and unread counts via store.SetUnreadCount().
5. Parses messages via whatsmeowClient.ParseWebMessage() and persists them to messages.db.

---

### Q25. [3. WhatsApp Web Protocol & Whatsmeow Library] How does c.ResolveLIDToPN() resolve LIDs (Lightweight Identifiers) to Phone Numbers (PN)?

WhatsApp recently introduced LIDs (e.g. 123456@lid) to mask phone numbers in group chats. 

c.ResolveLIDToPN() queries whatsmeow's contact store to map the LID to its underlying Phone Number JID (e.g. 919876543210@s.whatsapp.net). If no mapping is found, it returns the original JID.

---

### Q26. [3. WhatsApp Web Protocol & Whatsmeow Library] How does Whatsmeow's contact cache store sync with GhostWA's SQLite database (SyncStoreContacts)?

SyncStoreContacts() queries whatsmeow's internal contact store (c.whatsmeowClient.Store.Contacts.GetAllContacts()). For every contact, it extracts push names and address book names, and upserts them into GhostWA's SQLite contacts table.

---

### Q27. [4. Authentication Handshakes (QR Code & Phone Pairing)] How does QR Code generation and scanning work in handleLogin?

1. CLI issues {"type": "login"} to daemon.
2. handleLogin calls GetQRChannel(ctx) on whatsmeow client.
3. Whatsmeow returns a channel streaming QR code string events (evt.Event == "code").
4. Daemon streams QR code string to CLI over TCP socket.
5. CLI renders QR code in ANSI terminal using qrterminal.Draw().
6. User scans QR code with phone, WhatsApp sends "success" event, and session.db is saved.

---

### Q28. [4. Authentication Handshakes (QR Code & Phone Pairing)] Why must GetQRChannel() be called BEFORE Connect() during QR login?

Whatsmeow API Requirement: GetQRChannel() configures internal event handlers for QR generation. If Connect() is called first, Whatsmeow's WebSocket starts connecting without QR handlers registered, causing GetQRChannel() to panic with 'GetQRChannel must be called before connecting'. 

GhostWA fix: handleLogin explicitly calls s.client.Disconnect() before calling GetQRChannel().

---

### Q29. [4. Authentication Handshakes (QR Code & Phone Pairing)] How does qrterminal render QR codes directly inside ANSI terminal consoles?

qrterminal (github.com/mdp/qrterminal/v3) converts the QR matrix into Unicode half-block characters ('▀', '▄', '█') and ANSI background color escape codes. This allows rendering high-density QR codes cleanly inside PowerShell, CMD, or Linux terminal windows.

---

### Q30. [4. Authentication Handshakes (QR Code & Phone Pairing)] How does 8-digit Phone Pairing Code authentication work (handleLoginPhone)?

1. User runs 'ghostwa login', selects option [2], and inputs phone number (e.g. 919876543210).
2. CLI sends {"type": "login_phone", "body": "919876543210"}.
3. Daemon calls client.PairPhone("919876543210").
4. Whatsmeow calls WhatsApp servers and returns an 8-digit pairing code (e.g. ABC12345).
5. CLI formats code as 'ABC1 - 2345' in a cyan ASCII banner.
6. User enters code in WhatsApp under Linked Devices -> Link with Phone Number.

---

### Q31. [4. Authentication Handshakes (QR Code & Phone Pairing)] Why must Connect() be called BEFORE PairPhone() during phone pairing login?

Unlike QR code login, Phone Pairing requires an active, connected WebSocket connection to request the pairing code from WhatsApp's servers. Calling PairPhone() on a disconnected client returns 'failed to connect to WhatsApp servers'.

---

### Q32. [4. Authentication Handshakes (QR Code & Phone Pairing)] Why did we add a 5-second WebSocket connection wait loop in PairPhone()?

c.whatsmeowClient.Connect() is asynchronous — it returns immediately while the WebSocket TCP/TLS handshake completes in the background. 

If PairPhone() is called 1 millisecond after Connect(), IsConnected() is still false. The 5-second polling loop (for i := 0; i < 50; i++ { time.Sleep(100ms) }) waits for the WebSocket connection to become fully active before requesting the pairing code.

---

### Q33. [4. Authentication Handshakes (QR Code & Phone Pairing)] How does GhostWA clean and validate phone numbers (strings.Map digits filter)?

GhostWA strips all non-numeric characters from phone inputs using strings.Map:
cleanPhone := strings.Map(func(r rune) rune {
    if r >= '0' && r <= '9' {
        return r
    }
    return -1
}, phone)

This handles inputs like '+91 98765-43210', converting them cleanly to '919876543210'.

---

### Q34. [5. Database Architecture & SQLite Deep-Dive] What is SQLite and why is it used for local storage in GhostWA?

SQLite is a self-contained, file-based, serverless relational database. It stores the entire database in a single file on disk (messages.db), while providing full SQL query power and ACID transactions.

---

### Q35. [5. Database Architecture & SQLite Deep-Dive] What are ACID properties in SQLite?

• Atomicity: Database operations (like saving a message and updating unread count) complete fully or roll back completely.
• Consistency: Enforces column types and foreign key constraints.
• Isolation: Concurrency control via file locks.
• Durability: Once committed, message data persists on disk even during power failures.

---

### Q36. [5. Database Architecture & SQLite Deep-Dive] Why did we choose modernc.org/sqlite over mattn/go-sqlite3?

• mattn/go-sqlite3 requires CGO and MinGW GCC compilers on Windows.
• modernc.org/sqlite is transpiled C-to-Go, 100% CGO-free, and compiles natively on Windows x64 without any C toolchain!

---

### Q37. [5. Database Architecture & SQLite Deep-Dive] What caused the post-logout Windows file locking bug, and how did force-killing daemon locks solve it?

On Windows, open file handles are exclusively locked by the kernel. When ghostwa logout ran, the daemon held open SQLite file handles to messages.db. Windows blocked os.Remove(), leaving chats visible.

Fix: ghostwa logout executes 'taskkill /F /IM ghostwa.exe', closes SQLite handles, and purges the data folder.

---

### Q38. [5. Database Architecture & SQLite Deep-Dive] Detailed breakdown of the chats database table schema.

CREATE TABLE IF NOT EXISTS chats (
    jid TEXT PRIMARY KEY,
    name TEXT,
    unread_count INTEGER DEFAULT 0,
    last_message_time DATETIME,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

---

### Q39. [5. Database Architecture & SQLite Deep-Dive] Detailed breakdown of the messages database table schema.

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

---

### Q40. [5. Database Architecture & SQLite Deep-Dive] Detailed breakdown of the contacts database table schema.

CREATE TABLE IF NOT EXISTS contacts (
    jid TEXT PRIMARY KEY,
    name TEXT,
    push_name TEXT,
    phone_number TEXT
);

---

### Q41. [5. Database Architecture & SQLite Deep-Dive] How does RebuildChatsFromMessages() work as a self-healing SQL query?

RebuildChatsFromMessages() scans messages and contacts tables using:
INSERT INTO chats (jid, name, last_message_time, updated_at)
SELECT m.chat_jid, COALESCE(NULLIF(c.name, ''), NULLIF(c.push_name, ''), ''), MAX(m.timestamp), CURRENT_TIMESTAMP
FROM messages m LEFT JOIN contacts c ON m.chat_jid = c.jid
WHERE m.chat_jid IS NOT NULL AND m.chat_jid != '' GROUP BY m.chat_jid
ON CONFLICT(jid) DO UPDATE SET last_message_time = MAX(chats.last_message_time, excluded.last_message_time), updated_at = CURRENT_TIMESTAMP;

Guarantees no active chat is missing if messages exist in SQLite!

---

### Q42. [5. Database Architecture & SQLite Deep-Dive] How does SaveSessionMeta(), GetSessionMeta(), and ClearSessionMeta() maintain session_info.json?

session_info.json is stored at ~/.local/share/wacli/session_info.json. It stores {"logged_in": true, "phone": "919876543210"}. SaveSessionMeta() updates it on login, and ClearSessionMeta() deletes it on logout.

---

### Q43. [5. Database Architecture & SQLite Deep-Dive] How does GetDefaultDataDir() resolve cross-platform paths (~/.local/share/wacli)?

GetDefaultDataDir() checks for WACLI_DATA_DIR env variable first. If empty, uses os.UserHomeDir() + filepath.Join(".local", "share", "wacli"), resolving to C:\\Users\\<user>\\.local\\share\\wacli on Windows and ~/.local/share/wacli on Linux/macOS.

---

### Q44. [5. Database Architecture & SQLite Deep-Dive] How does DeleteChat() work in SQLite?

DeleteChat(jid) executes DELETE FROM messages WHERE chat_jid = ? followed by DELETE FROM chats WHERE jid = ?.

---

### Q45. [6. Terminal UI Engine (Bubble Tea & Lip Gloss)] What is Bubble Tea (The Elm Architecture: Model, Update, View)?

Bubble Tea implements Elm Architecture:
• Model: Struct holding application state (chats, messages, input buffer, focus index).
• Update: Function handling events (keypresses, window resize, incoming TCP messages) and returning updated model + commands.
• View: Function rendering model state into a string displayed in ANSI terminal.

---

### Q46. [6. Terminal UI Engine (Bubble Tea & Lip Gloss)] What is Lip Gloss and how does it provide CSS-like styling in terminal ANSI?

Lip Gloss provides fluent CSS-like styling (Border, Foreground, Background, Width, Height, Padding, Margin) and renders ANSI escape codes.

---

### Q47. [6. Terminal UI Engine (Bubble Tea & Lip Gloss)] What caused TUI layout distortion when real chats loaded in?

Root Causes:
1. UTF-8 Byte Slicing: String truncation via bytes (name[:N]) broke multi-byte UTF-8 emojis/accents.
2. Unbounded Line Wrapping: Time formatting pushed text past sidebar width, wrapping rows into 2 lines.
3. Unbounded List Rendering: Outputting 50 chats expanded sidebar box past terminal window height.

---

### Q48. [6. Terminal UI Engine (Bubble Tea & Lip Gloss)] UTF-8 Byte Slicing vs Rune Slicing ([]rune) for emojis and accents.

In Go, string indexing (s[:N]) indexes BYTES. UTF-8 emojis (like 💬) are 4 bytes long. Byte slicing produces invalid UTF-8 bytes. Converting to []rune slice ([]rune(s)[:N]) indexes visual characters safely!

---

### Q49. [6. Terminal UI Engine (Bubble Tea & Lip Gloss)] How does Sliding Viewport Scrolling work in show.go sidebar ([startIdx, endIdx])?

Sliding Viewport algorithm:
listHeight := m.height - 10
startIdx := m.selectedIdx - (listHeight / 2)
if startIdx < 0 { startIdx = 0 }
endIdx := startIdx + listHeight
if endIdx > len(m.filtered) { endIdx = len(m.filtered) }

Only renders visible window [startIdx, endIdx]. Sidebar height NEVER exceeds listHeight!

---

### Q50. [6. Terminal UI Engine (Bubble Tea & Lip Gloss)] How does fixed column width locking work in Lip Gloss?

By applying lipgloss.NewStyle().Width(maxNameWidth).MaxWidth(maxNameWidth).Inline(true).Render(nameStr), Lip Gloss forces every line to occupy exact visual columns, preventing line wrapping.

---

### Q51. [6. Terminal UI Engine (Bubble Tea & Lip Gloss)] How does Multi-Line Drafting work (Shift+Enter, Alt+Enter, Ctrl+J, backslash \ continuation)?

1. Listens for key events "shift+enter", "alt+enter", and "ctrl+j" and appends '\\n' to message buffer.
2. If text ends with '\\' when Enter is pressed, show.go strips '\\' and appends '\\n' without sending!

---

### Q52. [6. Terminal UI Engine (Bubble Tea & Lip Gloss)] How does showModel update state when an incomingMsg tea event is received?

When incomingMsg is received by Update(), showModel invokes reloadChats() and loadMessages(), re-querying SQLite and triggering p.Send() to re-render the view automatically.

---

### Q53. [7. Feature Mechanics & CLI/TUI Functionality] How do Message Reactions work in WhatsApp protocol (waE2E.ReactionMessage)?

Reactions construct waE2E.ReactionMessage protobuf referencing target message stanza ID (Key.ID), remote JID (Key.RemoteJID), and emoji string (Text).

---

### Q54. [7. Feature Mechanics & CLI/TUI Functionality] How do you send reactions in TUI (/react <emoji>) and CLI (ghostwa react)?

• TUI: Type '/react ❤️' or '/r 👍' in active chat input.
• CLI: Run 'ghostwa react <chat_or_phone> [msg_id] <emoji>'.

---

### Q55. [7. Feature Mechanics & CLI/TUI Functionality] Why did outgoing messages initially appear twice in show.go, and how was it fixed?

show.go was saving a duplicate local record ('out-150405') while the daemon was ALSO saving the official message ('3EB0...'). Removed manual SaveMessage in show.go, leaving single source of truth in daemon!

---

### Q56. [7. Feature Mechanics & CLI/TUI Functionality] How does media message sending work (auto-detecting images, videos, GIFs, documents)?

handleSend inspects file path via os.Stat(). If valid file, reads bytes and checks extension:
• .jpg / .png -> waE2E.ImageMessage
• .mp4 -> waE2E.VideoMessage
• .gif -> waE2E.VideoMessage (GifPlayback=true)
• Others -> waE2E.DocumentMessage. Uploads to WhatsApp media servers via whatsmeowClient.Upload().

---

### Q57. [7. Feature Mechanics & CLI/TUI Functionality] How does ghostwa chats differentiate between Saved Contacts and Unsaved Numbers using isSavedContact()?

isSavedContact() algorithm:
1. Strips JID suffix (@s.whatsapp.net).
2. Strips symbols (+, -, spaces, parens) from resolved name.
3. If cleanName == phoneDigits or name == JID -> UNSAVED. Else -> SAVED.

---

### Q58. [7. Feature Mechanics & CLI/TUI Functionality] How do 'ghostwa chats unsaved' and 'ghostwa chats all' work?

• 'ghostwa chats' (default): Displays saved contacts only.
• 'ghostwa chats unsaved': Displays unsaved phone numbers only.
• 'ghostwa chats all': Displays all direct chats combined.

---

### Q59. [7. Feature Mechanics & CLI/TUI Functionality] How does ghostwa sync (manual chat repair) work?

ghostwa sync invokes store.RebuildChatsFromMessages(), queries daemon status to re-trigger contact sync, and outputs count of repaired chats.

---

### Q60. [7. Feature Mechanics & CLI/TUI Functionality] How does ghostwa search <text> query SQLite across contacts, groups, and message bodies?

ghostwa search executes SELECT across contacts (name LIKE ?), chats (name LIKE ?), and messages (content LIKE ?) and outputs matching results.

---

### Q61. [8. Code Structure, Testing & Release Engineering] What is the directory structure of the GhostWA codebase?

• cmd/ghostwa: Entrypoint main.go.
• internal/cli: CLI command handlers and TUI dashboard.
• internal/daemon: Background server, TCP socket router, subscriber broadcaster.
• internal/store: SQLite database wrapper, schemas, self-healing queries.
• internal/whatsapp: Whatsmeow client wrapper, Noise protocol events.
• internal/resolver: Recipient resolution engine (maps names/phones to JIDs).

---

### Q62. [8. Code Structure, Testing & Release Engineering] How does internal/resolver resolve user input to valid target JIDs?

Resolver checks:
1. Is input already a valid JID (e.g. 919876543210@s.whatsapp.net)?
2. Is input in contacts table by friendly name?
3. Is input in chats table by chat name?
4. Is input a phone number? Appends @s.whatsapp.net.

---

### Q63. [8. Code Structure, Testing & Release Engineering] How are unit tests structured in root_test.go and database_test.go using t.Setenv('WACLI_DATA_DIR', tempDir)?

Unit tests use os.MkdirTemp() and t.Setenv("WACLI_DATA_DIR", tempDir) to isolate database tests inside temporary directories without modifying actual user databases.

---

### Q64. [8. Code Structure, Testing & Release Engineering] What is Semantic Versioning and how is patch progression enforced across files?

GhostWA uses Semantic Versioning (v2.5.0 -> v2.5.1 ... v2.5.6 -> v3.0.0). Every commit updates version strings across root.go, login.go, show.go, commands.go, root_test.go, and install-v2.5.ps1.

---

