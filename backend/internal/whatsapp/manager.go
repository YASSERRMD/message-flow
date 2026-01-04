package whatsapp

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"
)

type Session struct {
	ID         string
	TenantID   int64
	Client     *whatsmeow.Client
	Status     string
	LastQR     string
	LastExpiry time.Duration
	Error      string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type Manager struct {
	mu        sync.RWMutex
	container *sqlstore.Container
	sessions  map[string]*Session
	syncer    *Syncer
	log       waLog.Logger
}

func NewManager(ctx context.Context, databaseURL string) (*Manager, error) {
	if databaseURL == "" {
		return nil, errors.New("database url required")
	}
	log := waLog.Stdout("WhatsApp", "DEBUG", true)
	container, err := sqlstore.New(ctx, "pgx", databaseURL, log)
	if err != nil {
		return nil, err
	}
	return &Manager{
		container: container,
		sessions:  map[string]*Session{},
		log:       log,
	}, nil
}

func (m *Manager) SetSyncer(syncer *Syncer) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.syncer = syncer
}

// AutoReconnect reconnects all existing logged-in devices on startup
func (m *Manager) AutoReconnect(ctx context.Context) error {
	devices, err := m.container.GetAllDevices(ctx)
	if err != nil {
		m.log.Warnf("Failed to get devices: %v", err)
		return err
	}

	m.log.Infof("Found %d stored device(s)", len(devices))

	for _, device := range devices {
		if device.ID == nil {
			continue
		}

		m.log.Infof("Reconnecting device: %s", device.ID.String())

		clientLog := waLog.Stdout("Client", "DEBUG", true)
		client := whatsmeow.NewClient(device, clientLog)

		// Attach syncer with default tenant 1 (you may want to store tenant per device)
		if m.syncer != nil {
			m.syncer.Attach(1, client)
		}

		if err := client.Connect(); err != nil {
			m.log.Errorf("Failed to reconnect device %s: %v", device.ID.String(), err)
			continue
		}

		session := &Session{
			ID:        uuid.NewString(),
			TenantID:  1,
			Client:    client,
			Status:    "connected",
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}

		m.mu.Lock()
		m.sessions[session.ID] = session
		m.mu.Unlock()

		m.log.Infof("Successfully reconnected device: %s", device.ID.String())
	}

	return nil
}

func (m *Manager) StartSession(ctx context.Context, tenantID int64) (*Session, error) {
	// Try to get an existing device first
	device, err := m.container.GetFirstDevice(ctx)
	if err != nil {
		return nil, err
	}

	clientLog := waLog.Stdout("Client", "DEBUG", true)
	client := whatsmeow.NewClient(device, clientLog)

	if m.syncer != nil {
		m.syncer.Attach(tenantID, client)
	}

	session := &Session{
		ID:        uuid.NewString(),
		TenantID:  tenantID,
		Client:    client,
		Status:    "pending",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	// Check if already logged in
	if client.Store.ID != nil {
		// Already logged in, just connect
		if err := client.Connect(); err != nil {
			return nil, err
		}
		session.Status = "connected"
		m.mu.Lock()
		m.sessions[session.ID] = session
		m.mu.Unlock()
		return session, nil
	}

	// Not logged in, need QR pairing
	qrChan, err := client.GetQRChannel(ctx)
	if err != nil {
		return nil, err
	}

	if err := client.Connect(); err != nil {
		return nil, err
	}

	m.mu.Lock()
	m.sessions[session.ID] = session
	m.mu.Unlock()

	// Start consuming QR events in background
	go m.consumeQR(session, qrChan, client)

	return session, nil
}

func (m *Manager) GetSession(sessionID string) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	session, ok := m.sessions[sessionID]
	if !ok {
		return nil, false
	}
	copy := *session
	return &copy, true
}

// SendMessage sends a text message to a specific JID using the tenant's active session
func (m *Manager) SendMessage(ctx context.Context, tenantID int64, recipientJID string, content string) error {
	m.mu.RLock()
	var client *whatsmeow.Client
	for _, session := range m.sessions {
		if session.TenantID == tenantID && session.Status == "connected" {
			client = session.Client
			break
		}
	}
	m.mu.RUnlock()

	if client == nil {
		return errors.New("no connected whatsapp session found for tenant")
	}

	// Ensure JID has a domain
	if !strings.Contains(recipientJID, "@") {
		if strings.HasPrefix(recipientJID, "12036") {
			recipientJID = recipientJID + "@g.us"
		} else {
			recipientJID = recipientJID + "@s.whatsapp.net"
		}
	}

	jid, err := types.ParseJID(recipientJID)
	if err != nil {
		return err
	}

	// Check if user is on WhatsApp to prime the cache (fixes no LID found error)
	if jid.Server == types.DefaultUserServer {
		_, _ = client.IsOnWhatsApp(ctx, []string{jid.User})
	}

	_, err = client.SendMessage(ctx, jid, &waE2E.Message{
		Conversation: &content,
	})
	return err
}

func (m *Manager) consumeQR(session *Session, qrChan <-chan whatsmeow.QRChannelItem, client *whatsmeow.Client) {
	for item := range qrChan {
		m.mu.Lock()
		session.UpdatedAt = time.Now().UTC()

		switch item.Event {
		case "code":
			session.Status = "pending"
			session.LastQR = item.Code
			session.LastExpiry = item.Timeout
			m.log.Debugf("QR code received, timeout: %v", item.Timeout)
		case "success":
			session.Status = "connected"
			m.log.Infof("WhatsApp pairing successful!")
		case "timeout":
			session.Status = "timeout"
			m.log.Warnf("QR code timed out")
		case "error":
			session.Status = "error"
			if item.Error != nil {
				session.Error = item.Error.Error()
				m.log.Errorf("QR error: %v", item.Error)
			}
		default:
			session.Status = item.Event
			m.log.Debugf("Unknown QR event: %s", item.Event)
		}
		m.mu.Unlock()
	}

	// QR channel closed - check final connection status
	m.mu.Lock()
	if client.Store.ID != nil {
		session.Status = "connected"
		m.log.Infof("Connection established, JID: %s", client.Store.ID.String())
	}
	m.mu.Unlock()
}

func (m *Manager) DisconnectSession(tenantID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var sessionID string
	var client *whatsmeow.Client

	for id, sess := range m.sessions {
		if sess.TenantID == tenantID {
			sessionID = id
			client = sess.Client
			break
		}
	}

	if sessionID != "" {
		if client != nil {
			client.Disconnect()
		}
		delete(m.sessions, sessionID)
	}
	return nil
}
