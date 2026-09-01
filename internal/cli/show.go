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
	"time"

	wadaemon "github.com/Manan0708/GhostWA/internal/daemon"
	"github.com/Manan0708/GhostWA/internal/store"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type showModel struct {
	chats       []store.ChatSummary
	filtered    []store.ChatSummary
	selectedIdx int
	activeChat  store.ChatSummary
	messages    []store.Message

	searchVal  string
	messageVal string

	// focusIdx: 0 = search, 1 = chats, 2 = input
	focusIdx int

	width  int
	height int

	db       *store.Store
	conn     net.Conn
	err      error
	quitting bool
}

type incomingMsg wadaemon.Event
type tickMsg time.Time

func (m showModel) Init() tea.Cmd {
	return tea.Tick(1*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m showModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			if m.conn != nil {
				m.conn.Close()
			}
			return m, tea.Quit

		case "esc":
			if m.focusIdx != 1 {
				m.focusIdx = 1
				return m, nil
			}
			m.quitting = true
			if m.conn != nil {
				m.conn.Close()
			}
			return m, tea.Quit

		case "/":
			if m.focusIdx != 0 {
				m.focusIdx = 0
				return m, nil
			}

		case "tab":
			m.focusIdx = (m.focusIdx + 1) % 3
			return m, nil

		case "shift+tab":
			m.focusIdx = (m.focusIdx - 1 + 3) % 3
			return m, nil

		case "up":
			if m.focusIdx == 1 && len(m.filtered) > 0 {
				if m.selectedIdx > 0 {
					m.selectedIdx--
					m.activeChat = m.filtered[m.selectedIdx]
					m.loadMessages()
					_ = m.db.ResetUnreadCount(m.activeChat.JID)
					m.reloadChats()
				}
			}
			return m, nil

		case "down":
			if m.focusIdx == 1 && len(m.filtered) > 0 {
				if m.selectedIdx < len(m.filtered)-1 {
					m.selectedIdx++
					m.activeChat = m.filtered[m.selectedIdx]
					m.loadMessages()
					_ = m.db.ResetUnreadCount(m.activeChat.JID)
					m.reloadChats()
				}
			}
			return m, nil

		case "backspace":
			if m.focusIdx == 0 {
				if len(m.searchVal) > 0 {
					m.searchVal = m.searchVal[:len(m.searchVal)-1]
					m.filterChats()
				}
			} else if m.focusIdx == 2 {
				if len(m.messageVal) > 0 {
					m.messageVal = m.messageVal[:len(m.messageVal)-1]
				}
			}
			return m, nil

		case "ctrl+d", "delete":
			if m.activeChat.JID != "" {
				_ = m.db.DeleteChat(m.activeChat.JID)
				m.reloadChats()
			}
			return m, nil

		case "enter":
			if m.focusIdx == 0 {
				m.focusIdx = 1
			} else if m.focusIdx == 1 {
				m.focusIdx = 2
			} else if m.focusIdx == 2 && m.messageVal != "" && m.activeChat.JID != "" {
				text := strings.TrimSpace(m.messageVal)
				if strings.HasPrefix(text, "/react ") || strings.HasPrefix(text, "/r ") {
					parts := strings.SplitN(text, " ", 2)
					if len(parts) == 2 && len(m.messages) > 0 {
						lastMsg := m.messages[len(m.messages)-1]
						_ = sendReactionViaDaemon(m.activeChat.JID, lastMsg.ID, parts[1])
					}
				} else {
					_ = sendMessageViaDaemon(m.activeChat.JID, text)
				}

				m.messageVal = ""
				m.loadMessages()
			}
			return m, nil

		default:
			char := msg.String()
			if len(char) == 1 {
				if m.focusIdx == 0 {
					m.searchVal += char
					m.filterChats()
				} else if m.focusIdx == 2 {
					m.messageVal += char
				}
			} else if char == "space" {
				if m.focusIdx == 0 {
					m.searchVal += " "
					m.filterChats()
				} else if m.focusIdx == 2 {
					m.messageVal += " "
				}
			}
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		m.reloadChats()
		return m, tea.Tick(1*time.Second, func(t time.Time) tea.Msg {
			return tickMsg(t)
		})

	case incomingMsg:
		m.reloadChats()
		if m.activeChat.JID != "" {
			m.loadMessages()
			_ = m.db.ResetUnreadCount(m.activeChat.JID)
		}
		return m, nil
	}

	return m, nil
}

func (m *showModel) filterChats() {
	if m.searchVal == "" {
		m.filtered = m.chats
	} else {
		var list []store.ChatSummary
		q := strings.ToLower(m.searchVal)
		for _, c := range m.chats {
			if strings.Contains(strings.ToLower(c.Name), q) || strings.Contains(strings.ToLower(c.JID), q) {
				list = append(list, c)
			}
		}
		m.filtered = list
	}
	if m.selectedIdx >= len(m.filtered) {
		m.selectedIdx = 0
	}
	if len(m.filtered) > 0 {
		m.activeChat = m.filtered[m.selectedIdx]
		m.loadMessages()
	} else {
		m.activeChat = store.ChatSummary{}
		m.messages = nil
	}
}

func (m *showModel) reloadChats() {
	chats, err := m.db.GetChatList()
	if err == nil {
		m.chats = chats
		m.filterChats()
	}
}

func (m *showModel) loadMessages() {
	if m.activeChat.JID == "" {
		m.messages = nil
		return
	}
	msgs, err := m.db.GetRecentMessages(m.activeChat.JID, 30)
	if err == nil {
		m.messages = msgs
	}
}

func (m showModel) View() string {
	if m.quitting {
		return "Exiting GhostWA Dashboard...\n"
	}

	if m.width == 0 {
		m.width = 100
	}
	if m.height == 0 {
		m.height = 30
	}

	// Palette definitions
	purpleBg := lipgloss.Color("#6C5CE7")
	neonGreen := lipgloss.Color("#00FFA3")
	neonPink := lipgloss.Color("#FF0055")
	cyanAccent := lipgloss.Color("#00D2FF")
	headerBg := lipgloss.Color("#12121E")
	textColor := lipgloss.Color("#DFE6E9")
	mutedText := lipgloss.Color("#636E72")

	// Header Banner
	appTitle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(purpleBg).
		Padding(0, 2).
		Render("⚡ GHOSTWA v2.5.2")

	daemonBadge := lipgloss.NewStyle().
		Bold(true).
		Foreground(neonGreen).
		Background(headerBg).
		Padding(0, 1).
		Render("● ONLINE")

	clockStr := time.Now().Format("15:04:05")
	clockWidget := lipgloss.NewStyle().Foreground(cyanAccent).Render("🕒 " + clockStr)

	headerBar := lipgloss.JoinHorizontal(
		lipgloss.Center,
		appTitle,
		"  ",
		daemonBadge,
		"  ",
		clockWidget,
		"  ",
		lipgloss.NewStyle().Foreground(mutedText).Render("Silent WhatsApp Workspace"),
	)

	// Layout Widths
	sidebarWidth := m.width/3 - 2
	if sidebarWidth < 28 {
		sidebarWidth = 28
	}
	chatWidth := m.width - sidebarWidth - 5
	if chatWidth < 40 {
		chatWidth = 40
	}

	// 1. Search Box Component
	searchStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(mutedText).
		Width(sidebarWidth).
		Padding(0, 1)

	if m.focusIdx == 0 {
		searchStyle = searchStyle.BorderForeground(cyanAccent)
	}

	searchContent := "🔍 " + m.searchVal
	if m.searchVal == "" && m.focusIdx != 0 {
		searchContent = "🔍 Search chats... [/]"
	}
	searchBox := searchStyle.Render(searchContent)

	listHeight := m.height - 10
	if listHeight < 5 {
		listHeight = 5
	}

	// Calculate scrolling window so selected item is always visible
	startIdx := 0
	endIdx := len(m.filtered)

	if len(m.filtered) > listHeight {
		startIdx = m.selectedIdx - (listHeight / 2)
		if startIdx < 0 {
			startIdx = 0
		}
		endIdx = startIdx + listHeight
		if endIdx > len(m.filtered) {
			endIdx = len(m.filtered)
			startIdx = endIdx - listHeight
			if startIdx < 0 {
				startIdx = 0
			}
		}
	}

	// 2. Sidebar Chat List Component
	var listItems []string
	for idx := startIdx; idx < endIdx; idx++ {
		c := m.filtered[idx]
		timeStr := ""
		if !c.LastMessageTime.IsZero() {
			timeStr = c.LastMessageTime.Local().Format("15:04")
		}

		unreadBadge := ""
		if c.UnreadCount > 0 {
			unreadBadge = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(neonPink).
				Bold(true).
				Padding(0, 1).
				Render(fmt.Sprintf("%d", c.UnreadCount))
		}

		// Clean rune-safe truncation for contact names
		runes := []rune(c.Name)
		maxNameWidth := sidebarWidth - 14
		if maxNameWidth < 8 {
			maxNameWidth = 8
		}

		nameStr := string(runes)
		if len(runes) > maxNameWidth {
			nameStr = string(runes[:maxNameWidth-1]) + "…"
		}

		nameStyle := lipgloss.NewStyle().Width(maxNameWidth).MaxWidth(maxNameWidth).Inline(true).Render(nameStr)
		timeStyle := lipgloss.NewStyle().Width(5).Align(lipgloss.Right).Render(timeStr)

		itemLine := nameStyle + " " + timeStyle
		if unreadBadge != "" {
			itemLine = nameStyle + " " + unreadBadge
		}

		if idx == m.selectedIdx && m.focusIdx == 1 {
			listItems = append(listItems, lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(purpleBg).
				Bold(true).
				Width(sidebarWidth - 2).
				MaxHeight(1).
				Padding(0, 1).
				Render("▶ "+itemLine))
		} else if idx == m.selectedIdx {
			listItems = append(listItems, lipgloss.NewStyle().
				Foreground(purpleBg).
				Bold(true).
				Width(sidebarWidth - 2).
				MaxHeight(1).
				Padding(0, 1).
				Render("▶ "+itemLine))
		} else {
			listItems = append(listItems, lipgloss.NewStyle().
				Foreground(textColor).
				Width(sidebarWidth - 2).
				MaxHeight(1).
				Padding(0, 1).
				Render("  "+itemLine))
		}
	}

	if len(listItems) == 0 {
		listItems = []string{lipgloss.NewStyle().Foreground(mutedText).Padding(0, 1).Render("No active chats.")}
	}

	chatListBorderColor := mutedText
	if m.focusIdx == 1 {
		chatListBorderColor = purpleBg
	}

	chatsList := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(chatListBorderColor).
		Width(sidebarWidth).
		Height(listHeight).
		Render(strings.Join(listItems, "\n"))

	sidebar := lipgloss.JoinVertical(lipgloss.Left, searchBox, chatsList)

	// 3. Right Chat Panel Component
	chatHeaderTitle := "Select a conversation to view messages"
	if m.activeChat.Name != "" {
		chatHeaderTitle = "💬 " + m.activeChat.Name + " (" + m.activeChat.JID + ")"
	}

	chatHeader := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(headerBg).
		Width(chatWidth + 2).
		Padding(0, 1).
		Bold(true).
		Render(chatHeaderTitle)

	var messageLines []string
	for _, msg := range m.messages {
		timeStr := msg.Timestamp.Local().Format("15:04")
		sender := m.activeChat.Name
		if msg.IsFromMe {
			sender = "You"
		} else if strings.HasSuffix(m.activeChat.JID, "@g.us") {
			var savedName string
			_ = m.db.DB.QueryRow("SELECT COALESCE(name, push_name, '') FROM contacts WHERE jid = ?", msg.SenderJID).Scan(&savedName)
			if savedName != "" {
				sender = savedName
			} else {
				parts := strings.Split(msg.SenderJID, "@")
				sender = "+" + parts[0]
			}
		}

		reactionStr := ""
		if msg.Reaction != "" {
			reactionStr = " " + lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD700")).Bold(true).Render("["+msg.Reaction+"]")
		}

		var line string
		if msg.IsFromMe {
			msgText := lipgloss.NewStyle().Foreground(textColor).Render(msg.Content)
			line = fmt.Sprintf("[%s] %s: %s%s", timeStr, lipgloss.NewStyle().Foreground(neonGreen).Bold(true).Render("YOU"), msgText, reactionStr)
		} else {
			senderTag := lipgloss.NewStyle().Foreground(cyanAccent).Bold(true).Render(strings.ToUpper(sender))
			msgText := lipgloss.NewStyle().Foreground(textColor).Render(msg.Content)
			line = fmt.Sprintf("[%s] %s: %s%s", timeStr, senderTag, msgText, reactionStr)
		}
		messageLines = append(messageLines, line)
	}

	historyHeight := m.height - 11
	if historyHeight < 5 {
		historyHeight = 5
	}

	// Slice message lines to fit viewport height cleanly
	if len(messageLines) > historyHeight-1 {
		messageLines = messageLines[len(messageLines)-(historyHeight-1):]
	}

	historyBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(mutedText).
		Width(chatWidth).
		Height(historyHeight).
		Render(strings.Join(messageLines, "\n"))

	// 4. Input Box Component
	inputBorderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(mutedText).
		Width(chatWidth).
		Padding(0, 1)

	if m.focusIdx == 2 {
		inputBorderStyle = inputBorderStyle.BorderForeground(neonGreen)
	}

	inputPrompt := "❯ " + m.messageVal
	if m.messageVal == "" && m.focusIdx != 2 {
		inputPrompt = "❯ Type a message... [Press Tab to focus]"
	}
	inputBox := inputBorderStyle.Render(inputPrompt)

	chatPanel := lipgloss.JoinVertical(lipgloss.Left, chatHeader, historyBox, inputBox)

	// Combine Left Sidebar and Right Main Panel
	mainBody := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, " ", chatPanel)

	// Footer Help Bar
	footerLegend := lipgloss.NewStyle().
		Foreground(mutedText).
		Render("[Tab] Switch Focus  │  [Enter] Send Message  │  [d] Delete Chat  │  [/] Search  │  [Esc] Quit")

	return lipgloss.JoinVertical(lipgloss.Left, headerBar, "\n", mainBody, "\n", footerLegend)
}

// runShow launches the redesigned TUI Dashboard.
func runShow(stdout, stderr io.Writer) int {
	dataDir, err := store.GetDefaultDataDir()
	if err != nil {
		fmt.Fprintf(stderr, "Error finding data directory: %v\n", err)
		return 1
	}

	sessionPath := filepath.Join(dataDir, "session.db")
	if _, err := os.Stat(sessionPath); os.IsNotExist(err) {
		fmt.Fprintln(stderr, "Not logged in. Please run 'ghostwa login' to link your WhatsApp device.")
		return 1
	}

	dbPath := filepath.Join(dataDir, "messages.db")

	s, err := store.NewStore(dbPath)
	if err != nil {
		fmt.Fprintf(stderr, "Error opening messages database: %v\n", err)
		return 1
	}
	defer s.Close()

	conn, _ := net.Dial("tcp", "127.0.0.1:9090")
	if conn != nil {
		req := wadaemon.Request{Type: "subscribe"}
		data, _ := json.Marshal(req)
		_, _ = conn.Write(append(data, '\n'))
	}

	m := showModel{
		db:       s,
		conn:     conn,
		focusIdx: 1,
	}
	m.reloadChats()

	p := tea.NewProgram(m, tea.WithAltScreen())

	if conn != nil {
		go func() {
			reader := bufio.NewReader(conn)
			for {
				line, err := reader.ReadBytes('\n')
				if err != nil {
					break
				}
				var evt wadaemon.Event
				if err := json.Unmarshal(line, &evt); err == nil && evt.Type != "" {
					p.Send(incomingMsg(evt))
				}
			}
		}()
	}

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(stderr, "Error running TUI: %v\n", err)
		return 1
	}

	return 0
}

func sendReactionViaDaemon(toJID, msgID, emoji string) error {
	conn, err := wadaemon.ConnectOrStartDaemon()
	if err != nil {
		return err
	}
	defer conn.Close()

	req := wadaemon.Request{
		Type:  "react",
		To:    toJID,
		MsgID: msgID,
		Emoji: emoji,
	}
	data, _ := json.Marshal(req)
	_, err = conn.Write(append(data, '\n'))
	return err
}
