package daemon

// Request represents a JSON request sent from the CLI client to the background daemon.
type Request struct {
	Type string `json:"type"`            // "status", "login", "send", "subscribe", "stop"
	To   string `json:"to,omitempty"`   // recipient phone, JID, or contact name (for send)
	Body string `json:"body,omitempty"` // message content (for send)
}

// Response represents a JSON response sent from the background daemon to the CLI client.
type Response struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
	Status  string `json:"status,omitempty"` // "not_logged_in", "disconnected", "connected"
	Phone   string `json:"phone,omitempty"`
	MsgID   string `json:"msg_id,omitempty"`
}

// Event represents an asynchronous streaming event pushed from the daemon to subscribed clients.
type Event struct {
	Type       string `json:"type"`                          // "qr", "login_success", "message", "error"
	Code       string `json:"code,omitempty"`                // QR code data
	Sender     string `json:"sender,omitempty"`              // sender phone/JID
	SenderName string `json:"sender_name,omitempty"`         // friendly name of the sender
	Chat       string `json:"chat,omitempty"`                // chat room JID (direct or group JID)
	Body       string `json:"body,omitempty"`                // message content
	Timestamp  string `json:"timestamp,omitempty"`           // formatted message time
	IsRecent   bool   `json:"is_recent,omitempty"`
}
