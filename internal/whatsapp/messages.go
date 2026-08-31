package whatsapp

import (
	"context"
	"fmt"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

// SendTextMessage sends a basic text message to the specified recipient JID.
// It returns the message ID on success.
func (c *Client) SendTextMessage(ctx context.Context, recipient types.JID, text string) (string, error) {
	// Construct the WhatsApp Message protobuf
	msg := &waE2E.Message{
		Conversation: proto.String(text),
	}

	// Send via whatsmeow
	resp, err := c.whatsmeowClient.SendMessage(ctx, recipient, msg)
	if err != nil {
		return "", fmt.Errorf("failed to send text message: %w", err)
	}

	return resp.ID, nil
}
