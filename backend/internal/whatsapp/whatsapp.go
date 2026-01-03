package whatsapp

import (
	"context"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

type Client struct {
	client *whatsmeow.Client
}

func New(client *whatsmeow.Client) *Client {
	return &Client{client: client}
}

func (c *Client) HealthCheck(ctx context.Context) error {
	if c.client == nil {
		return whatsmeow.ErrNotLoggedIn
	}
	_, err := c.client.GetUserInfo(ctx, []types.JID{})
	return err
}
