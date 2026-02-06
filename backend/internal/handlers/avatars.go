package handlers

import (
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

func (a *API) GetConversationAvatar(w http.ResponseWriter, r *http.Request, conversationID int64) {
	tenantID := a.tenantID(r)
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()

	var jidStr string
	var contactNumber string
	if err := a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		return conn.QueryRow(ctx, `
			SELECT COALESCE(whatsapp_jid, ''), contact_number
			FROM conversations
			WHERE id=$1 AND tenant_id=$2`, conversationID, tenantID).Scan(&jidStr, &contactNumber)
	}); err != nil {
		writeError(w, http.StatusNotFound, "conversation not found")
		return
	}

	jidStr = strings.TrimSpace(jidStr)
	if jidStr == "" {
		jidStr = strings.TrimSpace(contactNumber)
	}
	if jidStr == "" {
		writeError(w, http.StatusNotFound, "conversation jid not found")
		return
	}
	if !strings.Contains(jidStr, "@") {
		if strings.HasPrefix(jidStr, "12036") {
			jidStr += "@g.us"
		} else {
			jidStr += "@s.whatsapp.net"
		}
	}

	jid, err := types.ParseJID(jidStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid jid")
		return
	}
	if a.WhatsApp == nil {
		writeError(w, http.StatusServiceUnavailable, "whatsapp not configured")
		return
	}
	client, err := a.WhatsApp.ClientForTenant(tenantID)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "whatsapp not connected")
		return
	}

	info, err := client.GetProfilePictureInfo(ctx, jid, &whatsmeow.GetProfilePictureParams{Preview: true})
	if err != nil || info == nil || strings.TrimSpace(info.URL) == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Proxy the URL through our backend so CSP/CORS doesn't block the frontend.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, info.URL, nil)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if ct := resp.Header.Get("Content-Type"); ct != "" {
		w.Header().Set("Content-Type", ct)
	} else {
		w.Header().Set("Content-Type", "image/jpeg")
	}
	w.Header().Set("Cache-Control", "private, max-age=3600")
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, resp.Body)
}
