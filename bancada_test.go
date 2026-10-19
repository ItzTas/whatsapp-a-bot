package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waAdv"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/llm"
)

var errOffline = errors.New("rede falsa: chamada sem suporte no teste")

type sentMessage struct {
	chat    types.JID
	message *waE2E.Message
}

func (m sentMessage) text() string {
	if poll := m.message.GetPollCreationMessage(); poll != nil {
		return poll.GetName()
	}
	return core.MessageText(m.message)
}

type fakeNetwork struct {
	mu     sync.Mutex
	sent   []sentMessage
	groups map[types.JID]*types.GroupInfo
	ids    atomic.Int64
}

func (n *fakeNetwork) SendMessage(_ context.Context, to types.JID, message *waE2E.Message, _ ...whatsmeow.SendRequestExtra) (whatsmeow.SendResponse, error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.sent = append(n.sent, sentMessage{chat: to.ToNonAD(), message: message})
	return whatsmeow.SendResponse{ID: types.MessageID(fmt.Sprintf("BOT%d", n.ids.Add(1))), Timestamp: time.Now()}, nil
}

func (n *fakeNetwork) GetGroupInfo(_ context.Context, jid types.JID) (*types.GroupInfo, error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	group, ok := n.groups[jid.ToNonAD()]
	if !ok {
		return nil, whatsmeow.ErrGroupNotFound
	}
	return group, nil
}

func (n *fakeNetwork) GetJoinedGroups(context.Context) ([]*types.GroupInfo, error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	groups := make([]*types.GroupInfo, 0, len(n.groups))
	for _, group := range n.groups {
		groups = append(groups, group)
	}
	return groups, nil
}

func (n *fakeNetwork) GetUserInfo(context.Context, []types.JID) (map[types.JID]types.UserInfo, error) {
	return nil, errOffline
}

func (n *fakeNetwork) GetProfilePictureInfo(context.Context, types.JID, *whatsmeow.GetProfilePictureParams) (*types.ProfilePictureInfo, error) {
	return nil, errOffline
}

func (n *fakeNetwork) UpdateGroupParticipants(context.Context, types.JID, []types.JID, whatsmeow.ParticipantChange) ([]types.GroupParticipant, error) {
	return nil, errOffline
}

func (n *fakeNetwork) SendChatPresence(context.Context, types.JID, types.ChatPresence, types.ChatPresenceMedia) error {
	return nil
}

func (n *fakeNetwork) Upload(_ context.Context, plaintext []byte, _ whatsmeow.MediaType) (whatsmeow.UploadResponse, error) {
	return uploaded(len(plaintext)), nil
}

func (n *fakeNetwork) UploadReader(_ context.Context, plaintext io.Reader, _ io.ReadWriteSeeker, _ whatsmeow.MediaType) (whatsmeow.UploadResponse, error) {
	size, err := io.Copy(io.Discard, plaintext)
	return uploaded(int(size)), err
}

func uploaded(size int) whatsmeow.UploadResponse {
	return whatsmeow.UploadResponse{
		URL:           "https://mmg.whatsapp.net/teste",
		DirectPath:    "/teste",
		MediaKey:      make([]byte, 32),
		FileEncSHA256: make([]byte, 32),
		FileSHA256:    make([]byte, 32),
		FileLength:    uint64(size),
	}
}

func (n *fakeNetwork) Download(context.Context, whatsmeow.DownloadableMessage) ([]byte, error) {
	return nil, errOffline
}

func (n *fakeNetwork) WaitForConnection(time.Duration) bool {
	return true
}

func (n *fakeNetwork) IsLoggedIn() bool {
	return true
}

func (n *fakeNetwork) count() int {
	n.mu.Lock()
	defer n.mu.Unlock()
	return len(n.sent)
}

func (n *fakeNetwork) since(start int) []sentMessage {
	n.mu.Lock()
	defer n.mu.Unlock()
	return slices.Clone(n.sent[start:])
}

type errorLog struct {
	mu     sync.Mutex
	errors []string
}

func (l *errorLog) Errorf(msg string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.errors = append(l.errors, fmt.Sprintf(msg, args...))
}

func (l *errorLog) Warnf(string, ...any)    {}
func (l *errorLog) Infof(string, ...any)    {}
func (l *errorLog) Debugf(string, ...any)   {}
func (l *errorLog) Sub(string) waLog.Logger { return l }

func (l *errorLog) take() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	taken := l.errors
	l.errors = nil
	return taken
}

type person struct {
	name string
	jid  types.JID
}

func newPerson(name, number string) person {
	return person{name: name, jid: types.NewJID(number, types.DefaultUserServer)}
}

func mentionTag(p person) string {
	return "@" + p.jid.User
}

type bench struct {
	t     *testing.T
	ctx   context.Context
	bot   *core.Bot
	db    *sql.DB
	net   *fakeNetwork
	log   *errorLog
	owner person
	ids   atomic.Int64
	dir   string
	stop  func()
}

func newBench(t *testing.T) *bench {
	t.Helper()
	owner := newPerson("Dono", "5511990000001")
	t.Setenv("BOT_DONO", owner.jid.User)
	b := &bench{
		t:     t,
		net:   &fakeNetwork{groups: map[types.JID]*types.GroupInfo{}},
		owner: owner,
		dir:   t.TempDir(),
		stop:  func() {},
	}
	t.Cleanup(func() { b.stop() })
	b.start()
	return b
}

func (b *bench) restart() {
	b.t.Helper()
	b.stop()
	b.start()
}

func (b *bench) start() {
	b.t.Helper()
	ctx, cancel := context.WithCancel(b.t.Context())
	var closers []func() error
	b.stop = func() {
		cancel()
		for _, closeDB := range slices.Backward(closers) {
			_ = closeDB()
		}
	}

	container, err := sqlstore.New(ctx, "sqlite3", "file:"+filepath.Join(b.dir, "sessao.db")+"?_foreign_keys=on", waLog.Noop)
	if err != nil {
		b.t.Fatalf("abrir sessao.db: %v", err)
	}
	closers = append(closers, container.Close)
	device, err := b.device(ctx, container)
	if err != nil {
		b.t.Fatalf("preparar dispositivo: %v", err)
	}
	messages, err := core.OpenMessageStore(ctx, filepath.Join(b.dir, "mensagens.db"))
	if err != nil {
		b.t.Fatalf("abrir mensagens.db: %v", err)
	}
	closers = append(closers, messages.Close)
	gifs, err := core.LoadGIFs()
	if err != nil {
		b.t.Fatalf("carregar gifs: %v", err)
	}

	log := &errorLog{}
	bot, err := assemble(ctx, whatsmeow.NewClient(device, log), gifs, messages, &llm.Assistant{})
	if err != nil {
		b.t.Fatalf("montar o bot: %v", err)
	}
	bot.Client.Network = b.net
	bot.RestoreGames(ctx)
	b.ctx, b.bot, b.db, b.log = ctx, bot, messages.DB, log
}

func (b *bench) device(ctx context.Context, container *sqlstore.Container) (*store.Device, error) {
	device, err := container.GetFirstDevice(ctx)
	if err != nil || device.ID != nil {
		return device, err
	}
	jid := types.NewJID("5511990000000", types.DefaultUserServer)
	device.ID = &jid
	device.LID = types.NewJID("190000000000000", types.HiddenUserServer)
	device.Account = &waAdv.ADVSignedDeviceIdentity{
		Details:             []byte{},
		AccountSignature:    make([]byte, 64),
		AccountSignatureKey: make([]byte, 32),
		DeviceSignature:     make([]byte, 64),
	}
	return device, device.Save(ctx)
}

func (b *bench) group(name string, members ...person) types.JID {
	b.t.Helper()
	jid := types.NewJID(fmt.Sprintf("12036300000000%04d", len(b.net.groups)+1), types.GroupServer)
	info := &types.GroupInfo{JID: jid, GroupName: types.GroupName{Name: name}}
	for _, p := range append([]person{b.owner}, members...) {
		info.Participants = append(info.Participants, types.GroupParticipant{JID: p.jid, PhoneNumber: p.jid, DisplayName: p.name})
	}
	b.net.mu.Lock()
	defer b.net.mu.Unlock()
	b.net.groups[jid] = info
	return jid
}

func (b *bench) say(chat types.JID, from person, text string, mentions ...person) []string {
	b.t.Helper()
	message := &waE2E.Message{Conversation: proto.String(text)}
	if len(mentions) > 0 {
		jids := make([]types.JID, len(mentions))
		for i, p := range mentions {
			jids[i] = p.jid
		}
		message = core.MentionMessage(text, jids)
	}
	evt := &events.Message{
		Info: types.MessageInfo{
			MessageSource: types.MessageSource{Chat: chat, Sender: from.jid, IsGroup: chat.Server == types.GroupServer},
			ID:            types.MessageID(fmt.Sprintf("MSG%d", b.ids.Add(1))),
			PushName:      from.name,
			Timestamp:     time.Now(),
		},
		Message: message,
	}
	start := b.net.count()
	b.bot.HandleMessage(b.ctx, evt)
	for _, e := range b.log.take() {
		b.t.Errorf("%q gerou erro no log do bot: %s", text, e)
	}
	var replies []string
	for _, m := range b.net.since(start) {
		if m.chat == chat.ToNonAD() {
			replies = append(replies, m.text())
		}
	}
	return replies
}

func (b *bench) setup(chat types.JID, modes string) {
	b.t.Helper()
	b.say(chat, b.owner, "!whitelist")
	b.say(chat, b.owner, "!modo ativar "+modes)
}

func (b *bench) balance(p person) int64 {
	b.t.Helper()
	var saldo int64
	err := b.db.QueryRowContext(b.ctx, "SELECT COALESCE(SUM(saldo), 0) FROM carteiras WHERE pessoa = ?", p.jid.User).Scan(&saldo)
	if err != nil {
		b.t.Fatalf("ler saldo de %s: %v", p.name, err)
	}
	return saldo
}

func wantReply(t *testing.T, replies []string, fragment string) string {
	t.Helper()
	i := slices.IndexFunc(replies, func(r string) bool { return strings.Contains(r, fragment) })
	if i < 0 {
		t.Fatalf("esperava uma resposta com %q, veio %q", fragment, replies)
	}
	return replies[i]
}

func wantSilence(t *testing.T, replies []string) {
	t.Helper()
	if len(replies) > 0 {
		t.Fatalf("esperava o bot calado, veio %q", replies)
	}
}
