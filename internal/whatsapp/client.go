package whatsapp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Manan0708/wacli/internal/store"
	_ "modernc.org/sqlite"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"
)

// Client wraps the whatsmeow Client, its underlying sqlstore Container, and the application store.
type Client struct {
	whatsmeowClient *whatsmeow.Client
	container       *sqlstore.Container
	store           *store.Store
}

// ResolveLIDToPN attempts to translate a LID JID into a phone number JID using the Whatsmeow store cache.
// If the JID is not an LID, or if no mapping exists, it returns the original JID.
func (c *Client) ResolveLIDToPN(ctx context.Context, jid types.JID) types.JID {
	if jid.Server == "lid" {
		pn, err := c.whatsmeowClient.Store.LIDs.GetPNForLID(ctx, jid)
		if err == nil && !pn.IsEmpty() {
			return pn
		}
	}
	return jid
}

// NewClient initializes the whatsmeow SQLite database store and creates the Whatsmeow client.
// The session database is stored inside the specified dataDir directory.
func NewClient(dataDir string, s *store.Store) (*Client, error) {
	// Create the directory if it doesn't exist
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	sessionDbPath := filepath.Join(dataDir, "session.db")

	// Initialize the whatsmeow sqlstore container.
	// We use the "sqlite" driver (provided by modernc.org/sqlite).
	container, err := sqlstore.New(context.Background(), "sqlite", "file:"+sessionDbPath+"?_foreign_keys=on", waLog.Noop)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize session store container: %w", err)
	}

	deviceStore, err := container.GetFirstDevice(context.Background())
	if err != nil {
		container.Close()
		return nil, fmt.Errorf("failed to get first device from store: %w", err)
	}

	// Instantiate the whatsmeow client.
	whatsmeowClient := whatsmeow.NewClient(deviceStore, waLog.Noop)

	return &Client{
		whatsmeowClient: whatsmeowClient,
		container:       container,
		store:           s,
	}, nil
}

// Close closes the underlying session database container.
func (c *Client) Close() error {
	return c.container.Close()
}

// IsLoggedIn checks whether a session ID exists in the device store,
// which indicates the user has completed the QR authentication.
func (c *Client) IsLoggedIn() bool {
	return c.whatsmeowClient.Store.ID != nil
}

// IsConnected returns whether the client is currently connected to WhatsApp servers.
func (c *Client) IsConnected() bool {
	return c.whatsmeowClient.IsConnected()
}

// Connect establishes connection with the WhatsApp servers.
func (c *Client) Connect() error {
	return c.whatsmeowClient.Connect()
}

// Disconnect gracefully disconnects from the WhatsApp servers.
func (c *Client) Disconnect() {
	c.whatsmeowClient.Disconnect()
}

// PhoneNumber returns the logged-in user's phone number, or empty string if not logged in.
func (c *Client) PhoneNumber() string {
	if c.whatsmeowClient.Store.ID == nil {
		return ""
	}
	return c.whatsmeowClient.Store.ID.User
}

// GetWhatsmeowClient returns the raw whatsmeow client instance for advanced operations.
func (c *Client) GetWhatsmeowClient() *whatsmeow.Client {
	return c.whatsmeowClient
}
