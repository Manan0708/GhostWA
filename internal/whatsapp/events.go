package whatsapp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// MessageEvent represents a simplified incoming message event.
type MessageEvent struct {
	SenderNum  string
	SenderName string
	ChatJID    string
	Text       string
	Timestamp  time.Time
	IsRecent   bool
}

// RegisterMessageEventHandler registers a callback that triggers whenever a new text message is received.
func (c *Client) RegisterMessageEventHandler(callback func(MessageEvent)) {
	c.whatsmeowClient.AddEventHandler(func(evt interface{}) {
		switch v := evt.(type) {
		case *events.HistorySync:
			if c.store == nil {
				return
			}
			for _, conv := range v.Data.GetConversations() {
				chatJID, _ := types.ParseJID(conv.GetID())
				if chatJID.IsEmpty() {
					continue
				}
				chatJID = c.ResolveLIDToPN(context.Background(), chatJID)

				// Attempt to get chat name (if available on conversation protobuf metadata)
				chatName := conv.GetName()

				// Register the chat in the local database
				_ = c.store.UpsertChat(chatJID.String(), chatName, time.Now())

				// Loop through and persist historical messages
				for _, historyMsg := range conv.GetMessages() {
					parsedEvt, err := c.whatsmeowClient.ParseWebMessage(chatJID, historyMsg.GetMessage())
					if err != nil {
						continue
					}

					text := ""
					if parsedEvt.Message.Conversation != nil {
						text = parsedEvt.Message.GetConversation()
					} else if parsedEvt.Message.ExtendedTextMessage != nil {
						text = parsedEvt.Message.ExtendedTextMessage.GetText()
					} else if parsedEvt.Message.ImageMessage != nil {
						text = "📷 [Image]"
						if caption := parsedEvt.Message.ImageMessage.GetCaption(); caption != "" {
							text += " " + caption
						}
					} else if parsedEvt.Message.VideoMessage != nil {
						text = "🎬 [Video/GIF]"
						if caption := parsedEvt.Message.VideoMessage.GetCaption(); caption != "" {
							text += " " + caption
						}
					} else if parsedEvt.Message.DocumentMessage != nil {
						filename := parsedEvt.Message.DocumentMessage.GetFileName()
						if filename == "" {
							filename = "Document"
						}
						text = fmt.Sprintf("📄 [%s]", filename)
					}

					if text != "" {
						normalizedSender := c.ResolveLIDToPN(context.Background(), parsedEvt.Info.Sender)
						senderJID := normalizedSender.String()

						// Register the sender contact details
						_ = c.store.UpsertContact(senderJID, chatName, normalizedSender.User, parsedEvt.Info.PushName)

						// Save the message content into SQLite
						_ = c.store.SaveMessage(parsedEvt.Info.ID, chatJID.String(), senderJID, text, parsedEvt.Info.Timestamp, parsedEvt.Info.IsFromMe)
					}
				}
			}
		case *events.Receipt:
			if c.store == nil {
				return
			}
			if v.Type == types.ReceiptTypeRead || v.Type == types.ReceiptTypeReadSelf {
				chatJID := c.ResolveLIDToPN(context.Background(), v.Chat).String()
				_ = c.store.ResetUnreadCount(chatJID)
			}

		case *events.Message:
			if v.Message == nil {
				return
			}
			// Extract the message text and handle media downloads
			text := ""
			if v.Message.Conversation != nil {
				text = v.Message.GetConversation()
			} else if v.Message.ExtendedTextMessage != nil {
				text = v.Message.ExtendedTextMessage.GetText()
			} else if v.Message.ImageMessage != nil {
				imgMsg := v.Message.ImageMessage
				data, err := c.whatsmeowClient.Download(context.Background(), imgMsg)
				if err == nil {
					ext := "jpg"
					if strings.Contains(imgMsg.GetMimetype(), "png") {
						ext = "png"
					} else if strings.Contains(imgMsg.GetMimetype(), "webp") {
						ext = "webp"
					}
					filename := fmt.Sprintf("IMG_%s.%s", time.Now().Format("20060102_150405"), ext)
					savePath := filepath.Join("downloads", filename)

					_ = os.MkdirAll("downloads", 0755)
					err = os.WriteFile(savePath, data, 0644)
					if err == nil {
						caption := imgMsg.GetCaption()
						if caption != "" {
							text = fmt.Sprintf("📷 [Image received: %s] %s", savePath, caption)
						} else {
							text = fmt.Sprintf("📷 [Image received: %s]", savePath)
						}
					}
				}
			} else if v.Message.VideoMessage != nil {
				vidMsg := v.Message.VideoMessage
				data, err := c.whatsmeowClient.Download(context.Background(), vidMsg)
				if err == nil {
					ext := "mp4"
					if strings.Contains(vidMsg.GetMimetype(), "gif") {
						ext = "gif"
					}
					filename := fmt.Sprintf("VID_%s.%s", time.Now().Format("20060102_150405"), ext)
					savePath := filepath.Join("downloads", filename)

					_ = os.MkdirAll("downloads", 0755)
					err = os.WriteFile(savePath, data, 0644)
					if err == nil {
						caption := vidMsg.GetCaption()
						if caption != "" {
							text = fmt.Sprintf("🎬 [Video/GIF received: %s] %s", savePath, caption)
						} else {
							text = fmt.Sprintf("🎬 [Video/GIF received: %s]", savePath)
						}
					}
				}
			} else if v.Message.DocumentMessage != nil {
				docMsg := v.Message.DocumentMessage
				data, err := c.whatsmeowClient.Download(context.Background(), docMsg)
				if err == nil {
					filename := docMsg.GetFileName()
					if filename == "" {
						filename = fmt.Sprintf("DOC_%s.bin", time.Now().Format("20060102_150405"))
					}
					savePath := filepath.Join("downloads", filename)

					_ = os.MkdirAll("downloads", 0755)
					err = os.WriteFile(savePath, data, 0644)
					if err == nil {
						text = fmt.Sprintf("📄 [Document received: %s]", savePath)
					}
				}
			}

			if text != "" {
				// Normalize JIDs by resolving LIDs to PNs if cached by whatsmeow
				normalizedSender := c.ResolveLIDToPN(context.Background(), v.Info.Sender)
				normalizedChat := c.ResolveLIDToPN(context.Background(), v.Info.Chat)

				senderJID := normalizedSender.String()
				chatJID := normalizedChat.String()

				if c.store != nil {
					// Auto-upsert the contact using the sender's PushName
					_ = c.store.UpsertContact(senderJID, v.Info.PushName, normalizedSender.User, v.Info.PushName)

					// Auto-upsert the chat metadata
					_ = c.store.UpsertChat(chatJID, "", v.Info.Timestamp)

					// If it is a group and name is missing, fetch group info from server in the background
					if v.Info.Chat.Server == "g.us" {
						var existingName string
						err := c.store.DB.QueryRow("SELECT name FROM chats WHERE jid = ?", chatJID).Scan(&existingName)
						if err != nil || existingName == "" || existingName == chatJID {
							go func() {
								info, err := c.whatsmeowClient.GetGroupInfo(context.Background(), v.Info.Chat)
								if err == nil && info != nil && info.Name != "" {
									_ = c.store.UpsertChat(chatJID, info.Name, time.Now())
								}
							}()
						}
					}

					// If the message is from others, increment unread count
					if !v.Info.IsFromMe {
						_ = c.store.IncrementUnreadCount(chatJID)
					}

					// Save message content
					_ = c.store.SaveMessage(v.Info.ID, chatJID, senderJID, text, v.Info.Timestamp, v.Info.IsFromMe)
				}

				// Only run the user callback for incoming messages sent by others
				if !v.Info.IsFromMe {
					senderName := v.Info.PushName
					if c.store != nil {
						var savedName string
						err := c.store.DB.QueryRow("SELECT name FROM contacts WHERE jid = ?", senderJID).Scan(&savedName)
						if err == nil && savedName != "" {
							senderName = savedName
						}
					}

					callback(MessageEvent{
						SenderNum:  normalizedSender.User,
						SenderName: senderName,
						ChatJID:    chatJID,
						Text:       text,
						Timestamp:  v.Info.Timestamp,
						IsRecent:   time.Since(v.Info.Timestamp) < 30 * time.Second,
					})
				}
			}
		}
	})
}
