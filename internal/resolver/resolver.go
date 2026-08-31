package resolver

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/Manan0708/wacli/internal/store"
	"go.mau.fi/whatsmeow/types"
)

// Resolver resolves user inputs into full WhatsApp JIDs.
type Resolver struct {
	store *store.Store
}

// NewResolver returns a new Resolver instance.
func NewResolver(s *store.Store) *Resolver {
	return &Resolver{store: s}
}

// Resolve translates a input string (JID, phone number, or local contact name) into a WhatsApp JID.
func (r *Resolver) Resolve(input string) (types.JID, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return types.EmptyJID, errors.New("empty identifier")
	}

	// 1. If it contains "@", treat it as a direct JID (e.g., 91XXXXXXXXXX@s.whatsapp.net or group JID)
	if strings.Contains(input, "@") {
		jid, err := types.ParseJID(input)
		if err != nil {
			return types.EmptyJID, fmt.Errorf("invalid JID format: %w", err)
		}
		return jid, nil
	}

	// 2. Clean the input from common phone formatting characters (spaces, dashes, parens)
	cleanNum := strings.TrimPrefix(input, "+")
	cleanNum = strings.ReplaceAll(cleanNum, " ", "")
	cleanNum = strings.ReplaceAll(cleanNum, "-", "")
	cleanNum = strings.ReplaceAll(cleanNum, "(", "")
	cleanNum = strings.ReplaceAll(cleanNum, ")", "")

	isNumeric := true
	for _, c := range cleanNum {
		if c < '0' || c > '9' {
			isNumeric = false
			break
		}
	}

	if isNumeric && len(cleanNum) >= 8 {
		jid, err := types.ParseJID(cleanNum + "@s.whatsapp.net")
		if err != nil {
			return types.EmptyJID, fmt.Errorf("failed to parse JID from phone number: %w", err)
		}
		return jid, nil
	}

	// 3. Look up the contact in the local SQLite database by name (partial match, case-insensitive)
	var jidStr string
	query := "SELECT jid FROM contacts WHERE name LIKE ? COLLATE NOCASE LIMIT 1"
	err := r.store.DB.QueryRow(query, "%"+input+"%").Scan(&jidStr)
	if err == nil {
		jid, err := types.ParseJID(jidStr)
		if err != nil {
			return types.EmptyJID, fmt.Errorf("stored JID is invalid: %w", err)
		}
		return jid, nil
	}

	if errors.Is(err, sql.ErrNoRows) {
		return types.EmptyJID, fmt.Errorf("could not resolve %q to a phone number, JID, or contact name", input)
	}

	return types.EmptyJID, fmt.Errorf("database query error: %w", err)
}
