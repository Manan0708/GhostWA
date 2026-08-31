package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"path/filepath"
	"strings"
	"time"

	wadaemon "github.com/Manan0708/wacli/internal/daemon"
	"github.com/Manan0708/wacli/internal/store"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type showModel struct {
	chats        []store.ChatSummary
	filtered     []store.ChatSummary
	selectedIdx  int
	activeChat   store.ChatSummary
	messages     []store.Message

	searchVal    string
	messageVal   string

	// focusIdx: 0 = search, 1 = chats, 2 = input
	focusIdx     int

	width        int
	height       int

	db           *store.Store
	conn         net.Conn
	err          error
	quitting     bool
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
		case "ctrl+c", "esc":
			m.quitting = true
			if m.conn != nil {
				m.conn.Close()
			}
			return m, tea.Quit

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
				}
			}
			return m, nil

		case "down":
			if m.focusIdx == 1 && len(m.filtered) > 0 {
				if m.selectedIdx < len(m.filtered)-1 {
					m.selectedIdx++
					m.activeChat = m.filtered[m.selectedIdx]
					m.loadMessages()
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

		case "enter":
			if m.focusIdx == 2 && m.messageVal != "" && m.activeChat.JID != "" {
				// Send message via daemon
				_ = sendMessageViaDaemon(m.activeChat.JID, m.messageVal)
				
				// Save locally
				myJID := "me@s.whatsapp.net"
				_ = m.db.UpsertChat(m.activeChat.JID, m.activeChat.Name, time.Now())
				_ = m.db.SaveMessage("out-"+time.Now().Format("150405"), m.activeChat.JID, myJID, m.messageVal, time.Now(), true)
				
				m.messageVal = ""
				m.loadMessages()
			}
			return m, nil

		default:
			// Handle standard letters/numbers/spaces for text inputs
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
		if msg.Type == "message" {
			m.reloadChats()
			if m.activeChat.JID != "" && (msg.Chat == m.activeChat.JID || (msg.Chat == "" && msg.Sender+"@s.whatsapp.net" == m.activeChat.JID)) {
				m.loadMessages()
				_ = m.db.ResetUnreadCount(m.activeChat.JID)
			}
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
			if strings.Contains(strings.ToLower(c.Name), q) {
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
	list, err := m.db.GetChatList()
	if err == nil {
		m.chats = list
		m.filterChats()
	}
}

func (m *showModel) loadMessages() {
	if m.activeChat.JID != "" {
		msgs, err := m.db.GetRecentMessages(m.activeChat.JID, 25)
		if err == nil {
			var rev []store.Message
			for i := len(msgs) - 1; i >= 0; i-- {
				rev = append(rev, msgs[i])
			}
			m.messages = rev
		}
	}
}

// Styling definitions using Lipgloss
var (
	purpleColor  = lipgloss.Color("99")
	darkColor    = lipgloss.Color("238")
	lightColor   = lipgloss.Color("250")
	activeBorder = lipgloss.Border{
		Top:    "─", Bottom: "─", Left: "│", Right: "│",
		TopLeft: "┌", TopRight: "┐", BottomLeft: "└", BottomRight: "┘",
	}

	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("255")).
			Background(purpleColor).
			Padding(0, 1).
			Bold(true)
)

func (m showModel) View() string {
	if m.quitting {
		return "Exiting WACLI Dashboard...\n"
	}
	if m.err != nil {
		return fmt.Sprintf("Error initializing dashboard: %v\nPress Esc to exit.", m.err)
	}

	sidebarWidth := 32
	chatWidth := m.width - sidebarWidth - 6
	if chatWidth < 20 {
		chatWidth = 20
	}

	searchBorderColor := darkColor
	if m.focusIdx == 0 {
		searchBorderColor = purpleColor
	}
	searchBox := lipgloss.NewStyle().
		Border(activeBorder).
		BorderForeground(searchBorderColor).
		Width(sidebarWidth).
		Padding(0, 1).
		Render("🔍 Search: " + m.searchVal)

	chatsBorderColor := darkColor
	if m.focusIdx == 1 {
		chatsBorderColor = purpleColor
	}

	var listItems []string
	for idx, c := range m.filtered {
		badge := ""
		if c.UnreadCount > 0 {
			badge = fmt.Sprintf(" [%d]", c.UnreadCount)
		}
		itemText := fmt.Sprintf("• %s%s", c.Name, badge)
		if len(itemText) > sidebarWidth-2 {
			itemText = itemText[:sidebarWidth-5] + "..."
		}

		if idx == m.selectedIdx && m.focusIdx == 1 {
			listItems = append(listItems, lipgloss.NewStyle().
				Foreground(lipgloss.Color("255")).
				Background(purpleColor).
				Bold(true).
				Render(itemText))
		} else if idx == m.selectedIdx {
			listItems = append(listItems, lipgloss.NewStyle().
				Foreground(purpleColor).
				Bold(true).
				Render(itemText))
		} else {
			listItems = append(listItems, lipgloss.NewStyle().
				Foreground(lightColor).
				Render(itemText))
		}
	}

	listHeight := m.height - 8
	if listHeight < 5 {
		listHeight = 5
	}
	chatsList := lipgloss.NewStyle().
		Border(activeBorder).
		BorderForeground(chatsBorderColor).
		Width(sidebarWidth).
		Height(listHeight).
		Render(strings.Join(listItems, "\n"))

	sidebar := lipgloss.JoinVertical(lipgloss.Left, searchBox, chatsList)

	chatHeader := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255")).
		Background(lipgloss.Color("236")).
		Width(chatWidth + 2).
		Padding(0, 1).
		Bold(true).
		Render("👤 " + m.activeChat.Name)

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
		line := fmt.Sprintf("[%s] %s: %s", timeStr, sender, msg.Content)
		messageLines = append(messageLines, line)
	}

	historyHeight := m.height - 9
	if historyHeight < 5 {
		historyHeight = 5
	}
	historyBox := lipgloss.NewStyle().
		Border(activeBorder).
		BorderForeground(darkColor).
		Width(chatWidth).
		Height(historyHeight).
		Render(strings.Join(messageLines, "\n"))

	inputBorderColor := darkColor
	if m.focusIdx == 2 {
		inputBorderColor = purpleColor
	}
	inputBox := lipgloss.NewStyle().
		Border(activeBorder).
		BorderForeground(inputBorderColor).
		Width(chatWidth).
		Padding(0, 1).
		Render("💬 Msg: " + m.messageVal)

	chatPane := lipgloss.JoinVertical(lipgloss.Left, chatHeader, historyBox, inputBox)
	dashboardLayout := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, chatPane)

	appHeader := headerStyle.Render(" WACLI Terminal Dashboard ")
	return appHeader + "\n" + dashboardLayout
}

// runShow boots the Charm Bubble Tea TUI app.
func runShow(stdout, stderr io.Writer) int {
	dataDir, err := store.GetDefaultDataDir()
	if err != nil {
		fmt.Fprintf(stderr, "Error resolving data directory: %v\n", err)
		return 1
	}

	dbPath := filepath.Join(dataDir, "messages.db")
	s, err := store.NewStore(dbPath)
	if err != nil {
		fmt.Fprintf(stderr, "Error opening database: %v\n", err)
		return 1
	}

	conn, err := net.Dial("tcp", "127.0.0.1:9090")
	if err != nil {
		s.Close()
		fmt.Fprintf(stderr, "Error: Daemon is offline. Please run 'wacli daemon start' first.\n")
		return 1
	}

	req := wadaemon.Request{Type: "subscribe"}
	reqData, _ := json.Marshal(req)
	_, _ = conn.Write(append(reqData, '\n'))

	reader := bufio.NewReader(conn)
	_, _ = reader.ReadBytes('\n')

	initialModel := showModel{
		db:        s,
		conn:      conn,
		focusIdx:  1,
		searchVal: "",
	}
	initialModel.reloadChats()

	p := tea.NewProgram(initialModel, tea.WithAltScreen())

	go func() {
		for {
			line, err := reader.ReadBytes('\n')
			if err != nil {
				break
			}
			var evt wadaemon.Event
			if err := json.Unmarshal(line, &evt); err == nil {
				p.Send(incomingMsg(evt))
			}
		}
	}()

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(stderr, "Error running TUI dashboard: %v\n", err)
		return 1
	}

	return 0
}
