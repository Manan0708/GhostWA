package daemon

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Manan0708/GhostWA/internal/resolver"
	"github.com/Manan0708/GhostWA/internal/store"
	"github.com/Manan0708/GhostWA/internal/whatsapp"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waCommon"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

// Server coordinates the SQLite store, the WhatsApp client, and the TCP IPC endpoint.
type Server struct {
	store       *store.Store
	client      *whatsapp.Client
	subscribers map[net.Conn]bool
	subMutex    sync.Mutex
	listener    net.Listener
	stopChan    chan struct{}
	dataDir     string
}

// NewServer initializes a new background daemon server.
func NewServer(dataDir string) (*Server, error) {
	dbPath := filepath.Join(dataDir, "messages.db")
	s, err := store.NewStore(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database store: %w", err)
	}

	cli, err := whatsapp.NewClient(dataDir, s)
	if err != nil {
		s.Close()
		return nil, fmt.Errorf("failed to initialize WhatsApp client: %w", err)
	}

	return &Server{
		store:       s,
		client:      cli,
		subscribers: make(map[net.Conn]bool),
		stopChan:    make(chan struct{}),
		dataDir:     dataDir,
	}, nil
}

// Run executes the daemon loops, registers WhatsApp events, and starts the TCP listener.
func (s *Server) Run() error {
	// Auto-connect to WhatsApp if we have credentials already saved
	if s.client.IsLoggedIn() {
		log.Println("Session active. Connecting to WhatsApp servers in the background...")
		_ = s.client.Connect()
		go func() {
			time.Sleep(5 * time.Second)
			s.syncContacts()
		}()
	}

	// Register the global message callback to broadcast incoming messages to subscribed clients
	s.client.RegisterMessageEventHandler(func(evt whatsapp.MessageEvent) {
		log.Printf("Incoming message in %s from %s -> %s (Recent: %t)", evt.ChatJID, evt.SenderNum, evt.Text, evt.IsRecent)
		s.broadcast(Event{
			Type:       "message",
			Sender:     evt.SenderNum,
			SenderName: evt.SenderName,
			Chat:       evt.ChatJID,
			Body:       evt.Text,
			Timestamp:  evt.Timestamp.Local().Format("15:04"),
			IsRecent:   evt.IsRecent,
		})
	})

	// Start listening on loopback TCP port 9090
	l, err := net.Listen("tcp", "127.0.0.1:9090")
	if err != nil {
		return fmt.Errorf("failed to start TCP listener: %w", err)
	}
	s.listener = l
	log.Println("Daemon server listening on 127.0.0.1:9090")

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.stopChan:
				return nil
			default:
				log.Printf("Failed to accept connection: %v", err)
				continue
			}
		}
		go s.handleConnection(conn)
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer func() {
		s.removeSubscriber(conn)
		conn.Close()
	}()

	reader := bufio.NewReader(conn)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				log.Printf("TCP read error: %v", err)
			}
			break
		}

		var req Request
		if err := json.Unmarshal(line, &req); err != nil {
			s.sendError(conn, fmt.Sprintf("invalid payload: %v", err))
			continue
		}

		switch req.Type {
		case "status":
			s.handleStatus(conn)
		case "login":
			s.handleLogin(conn)
		case "login_phone":
			s.handleLoginPhone(conn, req)
		case "logout":
			s.handleLogout(conn)
		case "send":
			s.handleSend(conn, req)
		case "subscribe":
			s.addSubscriber(conn)
			s.sendResponse(conn, Response{Success: true})
		case "react":
			s.handleReact(conn, req)
		case "delete_chat":
			s.handleDeleteChat(conn, req)
		case "sync_chats":
			s.handleSyncChats(conn)
		case "sync_history":
			s.handleSyncHistory(conn, req)
		case "stop":
			s.sendResponse(conn, Response{Success: true})
			s.Stop()
			return
		default:
			s.sendError(conn, fmt.Sprintf("unknown command type: %s", req.Type))
		}
	}
}

func (s *Server) handleStatus(conn net.Conn) {
	status := "disconnected"
	phone := "-"
	if s.client.IsLoggedIn() {
		status = "disconnected"
		if s.client.IsConnected() {
			status = "connected"
			go s.syncContacts()
		}
		phone = "+" + s.client.PhoneNumber()
	} else {
		status = "not_logged_in"
	}

	_ = s.sendResponse(conn, Response{
		Success: true,
		Status:  status,
		Phone:   phone,
	})
}

func (s *Server) handleLogin(conn net.Conn) {
	if s.client.IsLoggedIn() {
		s.sendError(conn, "already logged in")
		return
	}

	qrChan, err := s.client.GetWhatsmeowClient().GetQRChannel(context.Background())
	if err != nil {
		s.sendError(conn, fmt.Sprintf("failed to request QR channel: %v", err))
		return
	}

	err = s.client.Connect()
	if err != nil {
		s.sendError(conn, fmt.Sprintf("failed to connect: %v", err))
		return
	}

	// Stream QR code pairing states
	go func() {
		for evt := range qrChan {
			if evt.Event == "code" {
				_ = s.sendEvent(conn, Event{Type: "qr", Code: evt.Code})
			} else if evt.Event == "success" {
				_ = s.sendEvent(conn, Event{Type: "login_success"})
				log.Println("Session linked successfully via client scan")
				return
			} else if evt.Event == "timeout" {
				_ = s.sendEvent(conn, Event{Type: "error", Code: "timeout"})
				return
			}
		}
	}()
}

func (s *Server) handleLogout(conn net.Conn) {
	if s.client.IsLoggedIn() {
		_ = s.client.GetWhatsmeowClient().Logout(context.Background())
	}
	s.client.Disconnect()
	// Force-wipe session.db file so daemon resets completely to not_logged_in
	sessionPath := filepath.Join(s.dataDir, "session.db")
	_ = os.Remove(sessionPath)

	_ = s.sendResponse(conn, Response{Success: true})
	log.Println("Device logged out cleanly. Shutting down daemon.")

	go func() {
		time.Sleep(100 * time.Millisecond)
		s.Stop()
	}()
}

func (s *Server) handleLoginPhone(conn net.Conn, req Request) {
	if s.client.IsLoggedIn() {
		s.sendError(conn, "device is already logged in")
		return
	}
	if req.Body == "" {
		s.sendError(conn, "phone number required for pairing code")
		return
	}

	code, err := s.client.PairPhone(req.Body)
	if err != nil {
		s.sendError(conn, fmt.Sprintf("failed to get pairing code: %v", err))
		return
	}

	s.sendResponse(conn, Response{
		Success: true,
		Code:    code,
	})
}

func (s *Server) handleSend(conn net.Conn, req Request) {
	if !s.client.IsLoggedIn() {
		s.sendError(conn, "device is not logged in")
		return
	}

	// Connect to WhatsApp if currently offline
	if !s.client.IsConnected() {
		_ = s.client.Connect()
		for i := 0; i < 30; i++ {
			if s.client.IsConnected() {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		if !s.client.IsConnected() {
			s.sendError(conn, "unable to connect to WhatsApp servers")
			return
		}
	}

	// Resolve recipient
	res := resolver.NewResolver(s.store)
	targetJID, err := res.Resolve(req.To)
	if err != nil {
		s.sendError(conn, fmt.Sprintf("recipient resolution error: %v", err))
		return
	}

	var msgID string
	displayText := req.Body

	// Strip surrounding quotes if copy-pasted from Windows File Explorer
	cleanedPath := strings.Trim(req.Body, "\"'")

	// Check if the message body points to a valid file on disk (Media message trigger)
	var fileInfo os.FileInfo
	fileInfo, err = os.Stat(cleanedPath)
	if err == nil && !fileInfo.IsDir() {
		// Read file bytes
		fileData, err := os.ReadFile(cleanedPath)
		if err != nil {
			s.sendError(conn, fmt.Sprintf("failed to read media file: %v", err))
			return
		}

		// Determine media type and mimetype based on file extension
		ext := strings.ToLower(filepath.Ext(cleanedPath))
		var mediaType whatsmeow.MediaType
		var mimetype string
		var isImage, isVideo, isGIF bool

		switch ext {
		case ".jpg", ".jpeg":
			mediaType = whatsmeow.MediaImage
			mimetype = "image/jpeg"
			isImage = true
		case ".png":
			mediaType = whatsmeow.MediaImage
			mimetype = "image/png"
			isImage = true
		case ".webp":
			mediaType = whatsmeow.MediaImage
			mimetype = "image/webp"
			isImage = true
		case ".gif":
			mediaType = whatsmeow.MediaVideo
			mimetype = "image/gif"
			isVideo = true
			isGIF = true
		case ".mp4":
			mediaType = whatsmeow.MediaVideo
			mimetype = "video/mp4"
			isVideo = true
		default:
			mediaType = whatsmeow.MediaDocument
			mimetype = "application/octet-stream"
		}

		// Upload the media file to WhatsApp servers
		respUpload, err := s.client.GetWhatsmeowClient().Upload(context.Background(), fileData, mediaType)
		if err != nil {
			log.Printf("Error uploading media file %s: %v", cleanedPath, err)
			s.sendError(conn, fmt.Sprintf("failed to upload media file: %v", err))
			return
		}

		// Construct appropriate protobuf message type
		var msgProto *waE2E.Message
		if isImage {
			imgMsg := &waE2E.ImageMessage{
				URL:           proto.String(respUpload.URL),
				DirectPath:    proto.String(respUpload.DirectPath),
				MediaKey:      respUpload.MediaKey,
				Mimetype:      proto.String(mimetype),
				FileLength:    proto.Uint64(uint64(len(fileData))),
				FileSHA256:    respUpload.FileSHA256,
				FileEncSHA256: respUpload.FileEncSHA256,
			}
			msgProto = &waE2E.Message{ImageMessage: imgMsg}
			displayText = fmt.Sprintf("📷 [Image sent: %s]", filepath.Base(cleanedPath))
		} else if isVideo {
			vidMsg := &waE2E.VideoMessage{
				URL:           proto.String(respUpload.URL),
				DirectPath:    proto.String(respUpload.DirectPath),
				MediaKey:      respUpload.MediaKey,
				Mimetype:      proto.String(mimetype),
				FileLength:    proto.Uint64(uint64(len(fileData))),
				FileSHA256:    respUpload.FileSHA256,
				FileEncSHA256: respUpload.FileEncSHA256,
				GifPlayback:   proto.Bool(isGIF),
			}
			msgProto = &waE2E.Message{VideoMessage: vidMsg}
			if isGIF {
				displayText = fmt.Sprintf("🎬 [GIF sent: %s]", filepath.Base(cleanedPath))
			} else {
				displayText = fmt.Sprintf("🎬 [Video sent: %s]", filepath.Base(cleanedPath))
			}
		} else {
			docMsg := &waE2E.DocumentMessage{
				URL:           proto.String(respUpload.URL),
				DirectPath:    proto.String(respUpload.DirectPath),
				MediaKey:      respUpload.MediaKey,
				Mimetype:      proto.String(mimetype),
				FileLength:    proto.Uint64(uint64(len(fileData))),
				FileSHA256:    respUpload.FileSHA256,
				FileEncSHA256: respUpload.FileEncSHA256,
				FileName:      proto.String(filepath.Base(cleanedPath)),
			}
			msgProto = &waE2E.Message{DocumentMessage: docMsg}
			displayText = fmt.Sprintf("📄 [Document sent: %s]", filepath.Base(cleanedPath))
		}

		// Send media message
		respSend, err := s.client.GetWhatsmeowClient().SendMessage(context.Background(), targetJID, msgProto)
		if err != nil {
			s.sendError(conn, fmt.Sprintf("failed to send media message: %v", err))
			return
		}
		msgID = respSend.ID
	} else {
		// Plain text message fallback
		msgID, err = s.client.SendTextMessage(context.Background(), targetJID, req.Body)
		if err != nil {
			s.sendError(conn, fmt.Sprintf("failed to send message: %v", err))
			return
		}
	}

	// Log local history sink
	myJID := s.client.PhoneNumber() + "@s.whatsapp.net"
	_ = s.store.UpsertChat(targetJID.String(), "", time.Now())
	_ = s.store.SaveMessage(msgID, targetJID.String(), myJID, displayText, time.Now(), true)

	_ = s.sendResponse(conn, Response{
		Success: true,
		MsgID:   msgID,
	})
}

func (s *Server) sendResponse(conn net.Conn, resp Response) error {
	data, _ := json.Marshal(resp)
	_, err := conn.Write(append(data, '\n'))
	return err
}

func (s *Server) sendEvent(conn net.Conn, evt Event) error {
	data, _ := json.Marshal(evt)
	_, err := conn.Write(append(data, '\n'))
	return err
}

func (s *Server) sendError(conn net.Conn, msg string) {
	_ = s.sendResponse(conn, Response{Success: false, Error: msg})
}

func (s *Server) addSubscriber(conn net.Conn) {
	s.subMutex.Lock()
	defer s.subMutex.Unlock()
	s.subscribers[conn] = true
	log.Printf("TCP CLI client subscribed. Active connections: %d", len(s.subscribers))
}

func (s *Server) removeSubscriber(conn net.Conn) {
	s.subMutex.Lock()
	defer s.subMutex.Unlock()
	delete(s.subscribers, conn)
}

func (s *Server) broadcast(evt Event) {
	s.subMutex.Lock()
	defer s.subMutex.Unlock()

	data, _ := json.Marshal(evt)
	payload := append(data, '\n')

	for conn := range s.subscribers {
		go func(c net.Conn) {
			_, _ = c.Write(payload)
		}(conn)
	}
}

// Stop closes all network connections and database stores safely.
func (s *Server) Stop() {
	log.Println("Shutting down daemon server...")
	close(s.stopChan)
	if s.listener != nil {
		s.listener.Close()
	}
	s.client.Close()
	s.store.Close()
}

// syncContacts imports mobile contacts cached by whatsmeow into the local database
func (s *Server) syncContacts() {
	// 1. Sync contacts list
	contacts, err := s.client.GetWhatsmeowClient().Store.Contacts.GetAllContacts(context.Background())
	if err == nil {
		count := 0
		for jid, info := range contacts {
			name := info.FullName
			if name == "" {
				name = info.FirstName
			}
			if name == "" {
				name = info.PushName
			}
			if name == "" {
				continue
			}
			err = s.store.UpsertContact(jid.String(), name, jid.User, info.PushName)
			if err == nil {
				count++
			}
		}
		log.Printf("✓ Synchronized %d contacts from WhatsApp mobile store cache to application database", count)
	} else {
		log.Printf("Failed to retrieve contacts from mobile store: %v", err)
	}

	// 2. Fetch group names for any group chat missing a name in our chats database
	rows, err := s.store.DB.Query("SELECT jid FROM chats WHERE jid LIKE '%@g.us' AND (name IS NULL OR name = '' OR name = jid)")
	if err == nil {
		var groupJIDs []string
		for rows.Next() {
			var jid string
			if err := rows.Scan(&jid); err == nil {
				groupJIDs = append(groupJIDs, jid)
			}
		}
		rows.Close()

		resolvedGroups := 0
		for _, jidStr := range groupJIDs {
			gJID, err := types.ParseJID(jidStr)
			if err == nil {
				info, err := s.client.GetWhatsmeowClient().GetGroupInfo(context.Background(), gJID)
				if err == nil && info != nil && info.Name != "" {
					_ = s.store.UpsertChat(jidStr, info.Name, time.Now())
					resolvedGroups++
				}
			}
		}
		if resolvedGroups > 0 {
			log.Printf("✓ Proactively resolved names for %d group chats in local database", resolvedGroups)
		}
	}

	// 3. Merge duplicate LID chats into their corresponding PN chats
	s.mergeLIDs()
}

// mergeLIDs scans the database for LID JIDs (ending with @lid) and translates/merges them into PNs if mapped in Whatsmeow
func (s *Server) mergeLIDs() {
	rows, err := s.store.DB.Query("SELECT jid, unread_count, last_message_time FROM chats WHERE jid LIKE '%@lid'")
	if err != nil {
		return
	}

	type lidChat struct {
		LID             string
		UnreadCount     int
		LastMessageTime time.Time
	}

	var lidsToMerge []lidChat
	for rows.Next() {
		var lc lidChat
		if err := rows.Scan(&lc.LID, &lc.UnreadCount, &lc.LastMessageTime); err == nil {
			lidsToMerge = append(lidsToMerge, lc)
		}
	}
	rows.Close()

	mergedCount := 0
	for _, lc := range lidsToMerge {
		lidJID, err := types.ParseJID(lc.LID)
		if err != nil {
			continue
		}

		pnJID, err := s.client.GetWhatsmeowClient().Store.LIDs.GetPNForLID(context.Background(), lidJID)
		if err == nil && !pnJID.IsEmpty() {
			pnStr := pnJID.String()

			tx, err := s.store.DB.Begin()
			if err != nil {
				continue
			}

			var pnExists int
			_ = tx.QueryRow("SELECT COUNT(*) FROM chats WHERE jid = ?", pnStr).Scan(&pnExists)
			if pnExists == 0 {
				var lidName string
				_ = tx.QueryRow("SELECT name FROM chats WHERE jid = ?", lc.LID).Scan(&lidName)
				_, _ = tx.Exec("INSERT OR IGNORE INTO chats (jid, name, unread_count, last_message_time) VALUES (?, ?, 0, ?)", pnStr, lidName, lc.LastMessageTime)
			}

			_, _ = tx.Exec("UPDATE chats SET unread_count = unread_count + ?, last_message_time = CASE WHEN ? > last_message_time THEN ? ELSE last_message_time END WHERE jid = ?", lc.UnreadCount, lc.LastMessageTime, lc.LastMessageTime, pnStr)
			_, _ = tx.Exec("UPDATE messages SET chat_jid = ? WHERE chat_jid = ?", pnStr, lc.LID)
			_, _ = tx.Exec("UPDATE messages SET sender_jid = ? WHERE sender_jid = ?", pnStr, lc.LID)
			_, _ = tx.Exec("DELETE FROM chats WHERE jid = ?", lc.LID)
			_, _ = tx.Exec("DELETE FROM contacts WHERE jid = ?", lc.LID)

			if err := tx.Commit(); err == nil {
				mergedCount++
			} else {
				tx.Rollback()
			}
		}
	}

	if mergedCount > 0 {
		log.Printf("✓ Successfully merged %d duplicate LID chats into their corresponding Phone Number (PN) chats", mergedCount)
	}
}

func (s *Server) handleReact(conn net.Conn, req Request) {
	if !s.client.IsLoggedIn() {
		s.sendError(conn, "device is not logged in")
		return
	}
	res := resolver.NewResolver(s.store)
	targetJID, err := res.Resolve(req.To)
	if err != nil {
		s.sendError(conn, fmt.Sprintf("recipient resolution error: %v", err))
		return
	}

	targetMsgID := req.MsgID
	if targetMsgID == "" {
		lastMsg, err := s.store.GetLastMessage(targetJID.String())
		if err != nil || lastMsg == nil {
			s.sendError(conn, "no recent message found to react to")
			return
		}
		targetMsgID = lastMsg.ID
	}

	reactMsg := &waE2E.Message{
		ReactionMessage: &waE2E.ReactionMessage{
			Key: &waCommon.MessageKey{
				RemoteJID: proto.String(targetJID.String()),
				FromMe:    proto.Bool(false),
				ID:        proto.String(targetMsgID),
			},
			Text:              proto.String(req.Emoji),
			SenderTimestampMS: proto.Int64(time.Now().UnixMilli()),
		},
	}
	_, err = s.client.GetWhatsmeowClient().SendMessage(context.Background(), targetJID, reactMsg)
	if err != nil {
		s.sendError(conn, fmt.Sprintf("failed to send reaction: %v", err))
		return
	}

	_ = s.store.SetMessageReaction(targetMsgID, req.Emoji)
	_ = s.sendResponse(conn, Response{Success: true})
}

func (s *Server) handleDeleteChat(conn net.Conn, req Request) {
	res := resolver.NewResolver(s.store)
	targetJID, err := res.Resolve(req.To)
	if err != nil {
		s.sendError(conn, fmt.Sprintf("recipient resolution error: %v", err))
		return
	}

	err = s.store.DeleteChat(targetJID.String())
	if err != nil {
		s.sendError(conn, fmt.Sprintf("failed to delete chat: %v", err))
		return
	}

	_ = s.sendResponse(conn, Response{Success: true})
}

func (s *Server) handleSyncChats(conn net.Conn) {
	go s.syncContacts()
	_ = s.sendResponse(conn, Response{Success: true})
}

func (s *Server) handleSyncHistory(conn net.Conn, req Request) {
	go s.syncContacts()
	_ = s.sendResponse(conn, Response{Success: true})
}
