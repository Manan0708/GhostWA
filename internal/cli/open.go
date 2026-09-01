package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	wadaemon "github.com/Manan0708/GhostWA/internal/daemon"
	"github.com/Manan0708/GhostWA/internal/resolver"
	"github.com/Manan0708/GhostWA/internal/store"
	"golang.org/x/term"
)

// runOpen launches an interactive chat session using the background daemon for network traffic.
func runOpen(args []string, stdout, stderr io.Writer) int {
	if len(args) < 1 {
		fmt.Fprintln(stderr, "Error: missing contact name or phone number.")
		fmt.Fprintln(stderr, "Usage: wacli open <chat> [limit]")
		return 1
	}

	target := args[0]

	dataDir, err := store.GetDefaultDataDir()
	if err != nil {
		fmt.Fprintf(stderr, "Error finding data directory: %v\n", err)
		return 1
	}

	dbPath := filepath.Join(dataDir, "messages.db")
	s, err := store.NewStore(dbPath)
	if err != nil {
		fmt.Fprintf(stderr, "Error opening database: %v\n", err)
		return 1
	}
	defer s.Close()

	res := resolver.NewResolver(s)
	targetJID, err := res.Resolve(target)
	if err != nil {
		fmt.Fprintf(stderr, "Resolution error: %v\n", err)
		return 1
	}

	if os.Getenv("WACLI_TEST_MODE") == "true" {
		fmt.Fprintln(stderr, "Error: Not logged in. Run 'wacli login' first.")
		return 1
	}

	// Reset unread count locally
	_ = s.ResetUnreadCount(targetJID.String())

	// Parse optional history limit from the second argument (defaults to 3)
	historyLimit := 3
	if len(args) > 1 {
		var parsedLimit int
		_, err := fmt.Sscanf(args[1], "%d", &parsedLimit)
		if err == nil && parsedLimit >= 0 {
			historyLimit = parsedLimit
		}
	}

	// Determine friendly display name
	displayName := targetJID.String()
	var contactName string
	err = s.DB.QueryRow("SELECT COALESCE(name, push_name, jid) FROM contacts WHERE jid = ?", targetJID.String()).Scan(&contactName)
	if err == nil && contactName != "" {
		displayName = contactName
	}

	fmt.Fprintln(stdout, displayName)
	fmt.Fprintln(stdout, strings.Repeat("─", 40))
	fmt.Fprintln(stdout)

	// Fetch and print history locally
	if historyLimit > 0 {
		history, err := s.GetRecentMessages(targetJID.String(), historyLimit)
		if err == nil {
			for _, msg := range history {
				timestamp := msg.Timestamp.Local().Format("15:04")
				sender := displayName
				if msg.IsFromMe {
					sender = "You"
				} else if targetJID.Server == "g.us" {
					var senderName string
					err := s.DB.QueryRow("SELECT COALESCE(name, push_name, '') FROM contacts WHERE jid = ?", msg.SenderJID).Scan(&senderName)
					if err == nil && senderName != "" {
						sender = senderName
					} else {
						parts := strings.Split(msg.SenderJID, "@")
						sender = "+" + parts[0]
					}
				}
				fmt.Fprintf(stdout, "[%s] %s: %s\n", timestamp, sender, msg.Content)
			}
			if len(history) > 0 {
				fmt.Fprintln(stdout)
			}
		}
	}

	// Connect to or start the daemon
	conn, err := wadaemon.ConnectOrStartDaemon()
	if err != nil {
		fmt.Fprintf(stderr, "Error connecting to daemon: %v\n", err)
		return 1
	}
	defer conn.Close()

	// Subscribe to event updates
	subReq := wadaemon.Request{Type: "subscribe"}
	subData, _ := json.Marshal(subReq)
	_, _ = conn.Write(append(subData, '\n'))

	reader := bufio.NewReader(conn)
	// Read the subscription confirmation response
	_, _ = reader.ReadBytes('\n')

	stopLoop := make(chan struct{})
	state := &chatState{}

	// Spin up real-time receiver goroutine
	go func() {
		for {
			line, err := reader.ReadBytes('\n')
			if err != nil {
				select {
				case <-stopLoop:
					return
				default:
					fmt.Fprintf(stderr, "\nConnection lost with background daemon.\n")
					os.Exit(1)
				}
			}

			var evt wadaemon.Event
			_ = json.Unmarshal(line, &evt)

			if evt.Type == "message" {
				if !evt.IsRecent {
					continue
				}

				if evt.Chat == targetJID.String() {
					_ = s.ResetUnreadCount(targetJID.String())
					senderName := displayName
					if targetJID.Server == "g.us" {
						senderName = evt.SenderName
						if senderName == "" {
							senderName = "+" + evt.Sender
						}
					}
					fmt.Fprintf(stdout, "\r[%s] %s: %s\n> ", evt.Timestamp, senderName, evt.Body)
				} else {
					if state.isMuted() {
						continue
					}

					chatName := ""
					if strings.HasSuffix(evt.Chat, "@g.us") {
						_ = s.DB.QueryRow("SELECT COALESCE(name, '') FROM chats WHERE jid = ?", evt.Chat).Scan(&chatName)
						if chatName == "" {
							chatName = "Group"
						}
					} else {
						_ = s.DB.QueryRow("SELECT COALESCE(name, push_name, '') FROM contacts WHERE jid = ?", evt.Chat).Scan(&chatName)
						if chatName == "" {
							chatName = evt.SenderName
							if chatName == "" {
								chatName = "+" + evt.Sender
							}
						}
					}

					notification := fmt.Sprintf("🔔 [%s]", chatName)
					width, _, err := term.GetSize(int(os.Stdout.Fd()))
					if err != nil || width <= 0 {
						width = 80
					}

					runeCount := len([]rune(notification))
					displayLen := runeCount + 1
					padding := width - displayLen - 1

					if padding > 0 {
						fmt.Fprintf(stdout, "\r%s%s\n> ", strings.Repeat(" ", padding), notification)
					} else {
						fmt.Fprintf(stdout, "\r%s\n> ", notification)
					}
				}
			}
		}
	}()

	fmt.Fprint(stdout, "> ")

	// Start reading terminal inputs
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		if text == "" {
			fmt.Fprint(stdout, "> ")
			continue
		}

		if text == "/exit" || text == "/quit" {
			break
		}

		// Handle /history command dynamically
		if strings.HasPrefix(text, "/history") {
			parts := strings.Fields(text)
			limit := 10
			if len(parts) > 1 {
				var parsedLimit int
				_, err := fmt.Sscanf(parts[1], "%d", &parsedLimit)
				if err == nil && parsedLimit > 0 {
					limit = parsedLimit
				}
			}

			fmt.Fprintln(stdout, "\r─────────────────── [Older Messages] ───────────────────")
			history, err := s.GetRecentMessages(targetJID.String(), limit)
			if err == nil {
				for _, msg := range history {
					timestamp := msg.Timestamp.Local().Format("15:04")
					sender := displayName
					if msg.IsFromMe {
						sender = "You"
					}
					reactionStr := ""
					if msg.Reaction != "" {
						reactionStr = "  " + msg.Reaction
					}
					fmt.Fprintf(stdout, "[%s] %s: %s%s\n", timestamp, sender, msg.Content, reactionStr)
				}
				fmt.Fprintln(stdout, "────────────────────────────────────────────────────────")
			} else {
				fmt.Fprintf(stderr, "\rError loading history: %v\n", err)
			}
			fmt.Fprint(stdout, "> ")
			continue
		}

		// Handle /search command inside open chat
		if strings.HasPrefix(text, "/search") {
			parts := strings.SplitN(text, " ", 2)
			if len(parts) < 2 || strings.TrimSpace(parts[1]) == "" {
				fmt.Fprintln(stdout, "\rError: Missing search term. Usage: /search <term>")
				fmt.Fprint(stdout, "> ")
				continue
			}
			term := strings.TrimSpace(parts[1])

			fmt.Fprintf(stdout, "\r─────────────────── [Search Results: %s] ───────────────────\n", term)
			rows, err := s.DB.Query(`
				SELECT sender_jid, content, timestamp, is_from_me 
				FROM messages 
				WHERE chat_jid = ? AND content LIKE ? COLLATE NOCASE
				ORDER BY timestamp ASC
			`, targetJID.String(), "%"+term+"%")
			if err == nil {
				count := 0
				for rows.Next() {
					var senderJID, content string
					var ts time.Time
					var isFromMeInt int
					if err := rows.Scan(&senderJID, &content, &ts, &isFromMeInt); err == nil {
						count++
						timestamp := ts.Local().Format("Jan 02, 15:04")
						sender := displayName
						if isFromMeInt == 1 {
							sender = "You"
						} else if targetJID.Server == "g.us" {
							var senderName string
							_ = s.DB.QueryRow("SELECT COALESCE(name, push_name, '') FROM contacts WHERE jid = ?", senderJID).Scan(&senderName)
							if senderName != "" {
								sender = senderName
							} else {
								parts := strings.Split(senderJID, "@")
								sender = "+" + parts[0]
							}
						}
						fmt.Fprintf(stdout, "[%s] %s: %s\n", timestamp, sender, content)
					}
				}
				rows.Close()
				if count == 0 {
					fmt.Fprintln(stdout, "No matching messages found in this conversation.")
				}
				fmt.Fprintln(stdout, "────────────────────────────────────────────────────────")
			} else {
				fmt.Fprintf(stderr, "\rError searching history: %v\n", err)
			}
			fmt.Fprint(stdout, "> ")
			continue
		}

		// Handle /media command inside open chat
		if text == "/media" {
			fmt.Fprintln(stdout, "\r─────────────────── [Exchanged Media Files] ───────────────────")
			rows, err := s.DB.Query(`
				SELECT sender_jid, content, timestamp, is_from_me 
				FROM messages 
				WHERE chat_jid = ? AND (content LIKE '📷%' OR content LIKE '🎬%' OR content LIKE '📄%')
				ORDER BY timestamp ASC
			`, targetJID.String())
			if err == nil {
				count := 0
				for rows.Next() {
					var senderJID, content string
					var ts time.Time
					var isFromMeInt int
					if err := rows.Scan(&senderJID, &content, &ts, &isFromMeInt); err == nil {
						count++
						timestamp := ts.Local().Format("Jan 02, 15:04")
						sender := displayName
						if isFromMeInt == 1 {
							sender = "You"
						} else if targetJID.Server == "g.us" {
							var senderName string
							_ = s.DB.QueryRow("SELECT COALESCE(name, push_name, '') FROM contacts WHERE jid = ?", senderJID).Scan(&senderName)
							if senderName != "" {
								sender = senderName
							} else {
								parts := strings.Split(senderJID, "@")
								sender = "+" + parts[0]
							}
						}
						fmt.Fprintf(stdout, "[%s] %s: %s\n", timestamp, sender, content)
					}
				}
				rows.Close()
				if count == 0 {
					fmt.Fprintln(stdout, "No media files found in this conversation.")
				}
				fmt.Fprintln(stdout, "───────────────────────────────────────────────────────────────")
			} else {
				fmt.Fprintf(stderr, "\rError loading media files: %v\n", err)
			}
			continue
		}

		// Handle /mute command inside open chat
		if text == "/mute" {
			state.setMuted(true)
			fmt.Fprintln(stdout, "\rBackground notifications muted. Type /unmute to unmute, or /alerts to view pending messages.")
			fmt.Fprint(stdout, "> ")
			continue
		}

		// Handle /unmute command inside open chat
		if text == "/unmute" {
			state.setMuted(false)
			fmt.Fprintln(stdout, "\rBackground notifications unmuted.")
			fmt.Fprint(stdout, "> ")
			continue
		}

		// Handle /alerts command inside open chat
		if text == "/alerts" {
			fmt.Fprintln(stdout, "\r─────────────────── [Pending Alerts] ───────────────────")
			rows, err := s.DB.Query(`
				SELECT c.jid, COALESCE(c.name, ''), m.sender_jid, m.content, m.timestamp 
				FROM chats c
				JOIN messages m ON c.jid = m.chat_jid
				WHERE c.jid != ? AND c.unread_count > 0
				ORDER BY m.timestamp DESC
			`, targetJID.String())
			if err == nil {
				count := 0
				displayed := make(map[string]bool)
				for rows.Next() {
					var chatJID, chatName, senderJID, content string
					var ts time.Time
					if err := rows.Scan(&chatJID, &chatName, &senderJID, &content, &ts); err == nil {
						if displayed[chatJID] {
							continue
						}
						displayed[chatJID] = true
						count++

						timeStr := ts.Local().Format("15:04")
						chatTitle := chatName
						if chatTitle == "" {
							if strings.HasSuffix(chatJID, "@g.us") {
								chatTitle = "Group"
							} else {
								var name string
								_ = s.DB.QueryRow("SELECT COALESCE(name, push_name, '') FROM contacts WHERE jid = ?", chatJID).Scan(&name)
								if name != "" {
									chatTitle = name
								} else {
									chatTitle = chatJID
								}
							}
						}

						senderTitle := senderJID
						var sName string
						_ = s.DB.QueryRow("SELECT COALESCE(name, push_name, '') FROM contacts WHERE jid = ?", senderJID).Scan(&sName)
						if sName != "" {
							senderTitle = sName
						}

						if strings.HasSuffix(chatJID, "@g.us") {
							fmt.Fprintf(stdout, "  • %s (in %s): \"%s\" (%s)\n", senderTitle, chatTitle, content, timeStr)
						} else {
							fmt.Fprintf(stdout, "  • %s: \"%s\" (%s)\n", chatTitle, content, timeStr)
						}
					}
				}
				rows.Close()
				if count == 0 {
					fmt.Fprintln(stdout, "No pending background messages.")
				}
			} else {
				fmt.Fprintf(stderr, "\rError querying alerts: %v\n", err)
			}
			fmt.Fprintln(stdout, "────────────────────────────────────────────────────────")
			fmt.Fprint(stdout, "> ")
			continue
		}

		// Handle /react command inside open chat
		if strings.HasPrefix(text, "/react") {
			parts := strings.Fields(text)
			if len(parts) < 2 {
				fmt.Fprintln(stdout, "\rUsage: /react <emoji>")
			} else {
				emoji := parts[1]
				err := sendDaemonRequest("react", targetJID.String(), "", emoji)
				if err != nil {
					fmt.Fprintf(stderr, "\rError reacting: %v\n", err)
				} else {
					fmt.Fprintf(stdout, "\r✓ Reacted %s to recent message\n", emoji)
				}
			}
			fmt.Fprint(stdout, "> ")
			continue
		}

		// Handle /delete command inside open chat
		if text == "/delete" {
			err := sendDaemonRequest("delete_chat", targetJID.String(), "", "")
			if err != nil {
				fmt.Fprintf(stderr, "\rError deleting chat: %v\n", err)
			} else {
				fmt.Fprintln(stdout, "\r✓ Chat deleted successfully.")
				return 0
			}
			fmt.Fprint(stdout, "> ")
			continue
		}

		// Handle /sync command inside open chat
		if text == "/sync" || text == "/sync history" {
			err := sendDaemonRequest("sync_history", targetJID.String(), "", "")
			if err != nil {
				fmt.Fprintf(stderr, "\rError syncing chat: %v\n", err)
			} else {
				fmt.Fprintln(stdout, "\r✓ History and contact sync triggered for this chat.")
			}
			fmt.Fprint(stdout, "> ")
			continue
		}

		// Send message via daemon over a separate connection to avoid messing with current reader stream
		err := sendMessageViaDaemon(targetJID.String(), text)
		if err != nil {
			fmt.Fprintf(stderr, "\033[A\rError sending message: %v\n", err)
		} else {
			timestamp := time.Now().Format("15:04")
			// \033[A moves cursor UP one line, and \r moves it to the start to overwrite the typed line
			fmt.Fprintf(stdout, "\033[A\r[%s] You: %s\n", timestamp, text)
		}

		fmt.Fprint(stdout, "> ")
	}

	close(stopLoop)
	fmt.Fprintln(stdout, "\nExiting chat...")
	return 0
}

// sendMessageViaDaemon makes a quick connection to send the text message and disconnects.
func sendMessageViaDaemon(toJID string, body string) error {
	conn, err := net.Dial("tcp", "127.0.0.1:9090")
	if err != nil {
		return fmt.Errorf("daemon is not running: %w", err)
	}
	defer conn.Close()

	req := wadaemon.Request{
		Type: "send",
		To:   toJID,
		Body: body,
	}
	data, _ := json.Marshal(req)
	_, _ = conn.Write(append(data, '\n'))

	reader := bufio.NewReader(conn)
	line, err := reader.ReadBytes('\n')
	if err != nil {
		return fmt.Errorf("connection lost: %w", err)
	}

	var resp wadaemon.Response
	_ = json.Unmarshal(line, &resp)
	if !resp.Success {
		return fmt.Errorf("%s", resp.Error)
	}
	return nil
}

type chatState struct {
	sync.RWMutex
	muted bool
}

func (c *chatState) isMuted() bool {
	c.RLock()
	defer c.RUnlock()
	return c.muted
}

func (c *chatState) setMuted(v bool) {
	c.Lock()
	defer c.Unlock()
	c.muted = v
}

func sendDaemonRequest(reqType, to, body, emoji string) error {
	conn, err := net.Dial("tcp", "127.0.0.1:9090")
	if err != nil {
		return fmt.Errorf("daemon is not running: %w", err)
	}
	defer conn.Close()

	req := wadaemon.Request{
		Type:  reqType,
		To:    to,
		Body:  body,
		Emoji: emoji,
	}
	data, _ := json.Marshal(req)
	_, _ = conn.Write(append(data, '\n'))

	reader := bufio.NewReader(conn)
	line, err := reader.ReadBytes('\n')
	if err != nil {
		return fmt.Errorf("connection lost: %w", err)
	}

	var resp wadaemon.Response
	_ = json.Unmarshal(line, &resp)
	if !resp.Success {
		return fmt.Errorf("%s", resp.Error)
	}
	return nil
}
