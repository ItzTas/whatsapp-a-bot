package core

import (
	"context"
	"io"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
)

type Network interface {
	SendMessage(ctx context.Context, to types.JID, message *waE2E.Message, extra ...whatsmeow.SendRequestExtra) (whatsmeow.SendResponse, error)
	GetGroupInfo(ctx context.Context, jid types.JID) (*types.GroupInfo, error)
	GetJoinedGroups(ctx context.Context) ([]*types.GroupInfo, error)
	GetUserInfo(ctx context.Context, jids []types.JID) (map[types.JID]types.UserInfo, error)
	GetProfilePictureInfo(ctx context.Context, jid types.JID, params *whatsmeow.GetProfilePictureParams) (*types.ProfilePictureInfo, error)
	UpdateGroupParticipants(ctx context.Context, jid types.JID, changes []types.JID, action whatsmeow.ParticipantChange) ([]types.GroupParticipant, error)
	SendChatPresence(ctx context.Context, jid types.JID, state types.ChatPresence, media types.ChatPresenceMedia) error
	Upload(ctx context.Context, plaintext []byte, appInfo whatsmeow.MediaType) (whatsmeow.UploadResponse, error)
	UploadReader(ctx context.Context, plaintext io.Reader, tempFile io.ReadWriteSeeker, appInfo whatsmeow.MediaType) (whatsmeow.UploadResponse, error)
	Download(ctx context.Context, msg whatsmeow.DownloadableMessage) ([]byte, error)
	WaitForConnection(timeout time.Duration) bool
	IsLoggedIn() bool
}

type Client struct {
	*whatsmeow.Client
	Network Network
}

func (c *Client) SendMessage(ctx context.Context, to types.JID, message *waE2E.Message, extra ...whatsmeow.SendRequestExtra) (whatsmeow.SendResponse, error) {
	return c.Network.SendMessage(ctx, to, message, extra...)
}

func (c *Client) GetGroupInfo(ctx context.Context, jid types.JID) (*types.GroupInfo, error) {
	return c.Network.GetGroupInfo(ctx, jid)
}

func (c *Client) GetJoinedGroups(ctx context.Context) ([]*types.GroupInfo, error) {
	return c.Network.GetJoinedGroups(ctx)
}

func (c *Client) GetUserInfo(ctx context.Context, jids []types.JID) (map[types.JID]types.UserInfo, error) {
	return c.Network.GetUserInfo(ctx, jids)
}

func (c *Client) GetProfilePictureInfo(ctx context.Context, jid types.JID, params *whatsmeow.GetProfilePictureParams) (*types.ProfilePictureInfo, error) {
	return c.Network.GetProfilePictureInfo(ctx, jid, params)
}

func (c *Client) UpdateGroupParticipants(ctx context.Context, jid types.JID, changes []types.JID, action whatsmeow.ParticipantChange) ([]types.GroupParticipant, error) {
	return c.Network.UpdateGroupParticipants(ctx, jid, changes, action)
}

func (c *Client) SendChatPresence(ctx context.Context, jid types.JID, state types.ChatPresence, media types.ChatPresenceMedia) error {
	return c.Network.SendChatPresence(ctx, jid, state, media)
}

func (c *Client) Upload(ctx context.Context, plaintext []byte, appInfo whatsmeow.MediaType) (whatsmeow.UploadResponse, error) {
	return c.Network.Upload(ctx, plaintext, appInfo)
}

func (c *Client) UploadReader(ctx context.Context, plaintext io.Reader, tempFile io.ReadWriteSeeker, appInfo whatsmeow.MediaType) (whatsmeow.UploadResponse, error) {
	return c.Network.UploadReader(ctx, plaintext, tempFile, appInfo)
}

func (c *Client) Download(ctx context.Context, msg whatsmeow.DownloadableMessage) ([]byte, error) {
	return c.Network.Download(ctx, msg)
}

func (c *Client) WaitForConnection(timeout time.Duration) bool {
	return c.Network.WaitForConnection(timeout)
}

func (c *Client) IsLoggedIn() bool {
	return c.Network.IsLoggedIn()
}
