package whatsapp

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
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
}

func NewManager(ctx context.Context, databaseURL string) (*Manager, error) {
	if databaseURL == "" {
		return nil, errors.New("database url required")
	}
	container, err := sqlstore.New(ctx, "pgx", databaseURL, nil)
	if err != nil {
		return nil, err
	}
	return &Manager{
		container: container,
		sessions:  map[string]*Session{},
	}, nil
}

func (m *Manager) SetSyncer(syncer *Syncer) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.syncer = syncer
}

func (m *Manager) StartSession(ctx context.Context, tenantID int64) (*Session, error) {
	device := m.container.NewDevice()
	client := whatsmeow.NewClient(device, nil)
	if m.syncer != nil {
		m.syncer.Attach(tenantID, client)
	}

	qrChan, err := client.GetQRChannel(ctx)
	if err != nil {
		return nil, err
	}
	if err := client.Connect(); err != nil {
		return nil, err
	}

	session := &Session{
		ID:        uuid.NewString(),
		TenantID:  tenantID,
		Client:    client,
		Status:    "pending",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	m.mu.Lock()
	m.sessions[session.ID] = session
	m.mu.Unlock()

	go m.consumeQR(session, qrChan)

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

func (m *Manager) consumeQR(session *Session, qrChan <-chan whatsmeow.QRChannelItem) {
	for item := range qrChan {
		m.mu.Lock()
		session.UpdatedAt = time.Now().UTC()

		switch item.Event {
		case "code":
			session.Status = "pending"
			session.LastQR = item.Code
			session.LastExpiry = item.Timeout
		case "success":
			session.Status = "connected"
		case "timeout":
			session.Status = "timeout"
		case "error":
			session.Status = "error"
			if item.Error != nil {
				session.Error = item.Error.Error()
			}
		default:
			session.Status = item.Event
		}
		m.mu.Unlock()
	}
}
