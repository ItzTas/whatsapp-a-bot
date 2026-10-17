package economy

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
)

var errSameLedger = errors.New("os grupos já usam a mesma economia")

type ledgerBook struct {
	mu    sync.RWMutex
	roots map[types.JID]types.JID
}

var ledgers = &ledgerBook{roots: map[types.JID]types.JID{}}

func (l *ledgerBook) load(ctx context.Context, db *sql.DB) (err error) {
	rows, err := db.QueryContext(ctx, "SELECT chat, economia FROM economia_vinculos")
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()

	roots := map[types.JID]types.JID{}
	for rows.Next() {
		var chat, root types.JID
		if err := rows.Scan(&chat, &root); err != nil {
			return err
		}
		roots[chat] = root
	}
	if err := rows.Err(); err != nil {
		return err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.roots = roots
	return nil
}

func (l *ledgerBook) of(chat types.JID) types.JID {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.rootOf(chat)
}

func (l *ledgerBook) rootOf(chat types.JID) types.JID {
	if root, ok := l.roots[chat]; ok {
		return root
	}
	return chat
}

func (l *ledgerBook) linked(chat types.JID) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	_, ok := l.roots[chat]
	return ok
}

func (l *ledgerBook) members(root types.JID) []types.JID {
	l.mu.RLock()
	defer l.mu.RUnlock()
	groups := []types.JID{root}
	for chat, r := range l.roots {
		if r == root {
			groups = append(groups, chat)
		}
	}
	slices.SortFunc(groups[1:], func(a, b types.JID) int { return strings.Compare(a.String(), b.String()) })
	return groups
}

func (l *ledgerBook) link(ctx context.Context, db *sql.DB, chat, target types.JID) (root types.JID, followers []types.JID, err error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	root = l.rootOf(target)
	if root == l.rootOf(chat) {
		return root, nil, errSameLedger
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return root, nil, err
	}
	defer func() {
		if err != nil {
			err = errors.Join(err, tx.Rollback())
		}
	}()
	if _, err = tx.ExecContext(ctx, "UPDATE economia_vinculos SET economia = ? WHERE economia = ?", root.String(), chat.String()); err != nil {
		return root, nil, err
	}
	_, err = tx.ExecContext(ctx,
		`INSERT INTO economia_vinculos (chat, economia) VALUES (?, ?)
		 ON CONFLICT (chat) DO UPDATE SET economia = excluded.economia`,
		chat.String(), root.String(),
	)
	if err != nil {
		return root, nil, err
	}
	if err = tx.Commit(); err != nil {
		return root, nil, err
	}
	for member, r := range l.roots {
		if r == chat {
			l.roots[member] = root
			followers = append(followers, member)
		}
	}
	l.roots[chat] = root
	return root, followers, nil
}

func (l *ledgerBook) unlink(ctx context.Context, db *sql.DB, chat types.JID) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	res, err := db.ExecContext(ctx, "DELETE FROM economia_vinculos WHERE chat = ?", chat.String())
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	delete(l.roots, chat)
	return n > 0, nil
}

func (l *ledgerBook) arg(arg any) any {
	if jid, ok := arg.(types.JID); ok {
		return l.of(jid).String()
	}
	raw, ok := arg.(string)
	if !ok || !strings.HasSuffix(raw, "@"+types.GroupServer) {
		return arg
	}
	jid, err := types.ParseJID(raw)
	if err != nil {
		return arg
	}
	return l.of(jid).String()
}

func ledgerArgs(args []any) []any {
	mapped := make([]any, len(args))
	for i, arg := range args {
		mapped[i] = ledgers.arg(arg)
	}
	return mapped
}

type ledgerDB struct {
	db *sql.DB
}

func (l *ledgerDB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return l.db.ExecContext(ctx, query, ledgerArgs(args)...)
}

func (l *ledgerDB) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return l.db.QueryContext(ctx, query, ledgerArgs(args)...)
}

func (l *ledgerDB) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return l.db.QueryRowContext(ctx, query, ledgerArgs(args)...)
}

func (l *ledgerDB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*ledgerTx, error) {
	tx, err := l.db.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &ledgerTx{tx: tx}, nil
}

type ledgerTx struct {
	tx *sql.Tx
}

func (l *ledgerTx) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return l.tx.ExecContext(ctx, query, ledgerArgs(args)...)
}

func (l *ledgerTx) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return l.tx.QueryContext(ctx, query, ledgerArgs(args)...)
}

func (l *ledgerTx) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return l.tx.QueryRowContext(ctx, query, ledgerArgs(args)...)
}

func (l *ledgerTx) Commit() error {
	return l.tx.Commit()
}

func (l *ledgerTx) Rollback() error {
	return l.tx.Rollback()
}

func (b *bot) economyGroups(ctx context.Context, root types.JID) []types.JID {
	return slices.DeleteFunc(ledgers.members(root), func(group types.JID) bool { return !b.EconomyOn(ctx, group) })
}

func (b *bot) announce(ctx context.Context, root types.JID, text string, featured types.JID) error {
	groups := b.economyGroups(ctx, root)
	errs := make([]error, len(groups))
	var g errgroup.Group
	for i, group := range groups {
		g.Go(func() error {
			errs[i] = b.announceIn(ctx, group, text, featured)
			return nil
		})
	}
	_ = g.Wait()
	return errors.Join(errs...)
}

func (b *bot) announceIn(ctx context.Context, group types.JID, text string, featured types.JID) error {
	allowed, err := b.AllowedIn(ctx, group, []types.JID{featured})
	if err != nil || !allowed {
		return err
	}
	_, err = b.SendMentions(ctx, group, text, []types.JID{featured})
	return err
}

func (b *bot) groupNames(ctx context.Context, groups []types.JID) []string {
	names := make([]string, len(groups))
	var g errgroup.Group
	g.SetLimit(5)
	for i, group := range groups {
		g.Go(func() error {
			names[i] = b.groupName(ctx, group)
			return nil
		})
	}
	_ = g.Wait()
	return names
}

func (b *bot) groupName(ctx context.Context, group types.JID) string {
	info, err := b.Client.GetGroupInfo(ctx, group)
	if err != nil {
		b.Client.Log.Warnf("ler nome do grupo %s: %v", group, err)
		return group.User
	}
	return info.Name
}

func (b *bot) linkEconomy(ctx context.Context, evt *events.Message, args []string) error {
	chat := evt.Info.Chat.ToNonAD()
	if chat.Server != types.GroupServer {
		return b.SendText(ctx, chat, "Manda o !economia vincular dentro do grupo que vai entrar na economia do outro")
	}
	name := strings.Join(args, " ")
	if name == "" {
		return b.SendText(ctx, chat, "Use assim: !economia vincular <nome do grupo>. Este grupo passa a usar a economia daquele")
	}
	group, problem, err := b.GroupNamed(ctx, name)
	if err != nil {
		return err
	}
	if problem != "" {
		return b.SendText(ctx, chat, problem)
	}
	root, followers, err := ledgers.link(ctx, b.Messages.DB, chat, group.JID)
	if errors.Is(err, errSameLedger) {
		return b.SendText(ctx, chat, "Este grupo e o *"+group.Name+"* já usam a mesma economia 🤝")
	}
	if err != nil {
		return fmt.Errorf("vincular economia: %w", err)
	}
	names := b.groupNames(ctx, append([]types.JID{root}, followers...))
	lines := []string{fmt.Sprintf("🔗 Este grupo agora usa a economia do *%s*: carteira, mochila, presidente, ranking, loteria, cadeia, tudo junto", names[0])}
	if root != group.JID {
		lines = append(lines, fmt.Sprintf("_O *%s* já usava essa economia, então este grupo entrou nela também_", group.Name))
	}
	lines = append(lines, "", "📦 O que este grupo tinha antes fica guardado e volta se ele sair com !economia desvincular")
	if len(followers) > 0 {
		lines = append(lines, "", "👥 Os grupos que usavam a economia deste vieram junto: "+strings.Join(names[1:], ", "))
	}
	if !b.EconomyOn(ctx, chat) {
		lines = append(lines, "", "⚠️ O modo economia tá desligado aqui. Liga com !modo ativar economia pra usar")
	}
	return b.SendText(ctx, chat, strings.Join(lines, "\n"))
}

func (b *bot) unlinkEconomy(ctx context.Context, evt *events.Message, _ []string) error {
	chat := evt.Info.Chat.ToNonAD()
	if chat.Server != types.GroupServer {
		return b.SendText(ctx, chat, "Manda o !economia desvincular dentro do grupo que vai voltar pra economia própria")
	}
	unlinked, err := ledgers.unlink(ctx, b.Messages.DB, chat)
	if err != nil {
		return fmt.Errorf("desvincular economia: %w", err)
	}
	if unlinked {
		return b.SendText(ctx, chat, "✂️ Este grupo saiu da economia compartilhada e voltou pra economia própria, do jeito que tava antes de vincular")
	}
	followers := ledgers.members(chat)[1:]
	if len(followers) == 0 {
		return b.SendText(ctx, chat, "Este grupo não tá vinculado a nenhum outro, a economia dele já é só dele")
	}
	return b.SendText(ctx, chat, "Este grupo é a economia principal de: "+strings.Join(b.groupNames(ctx, followers), ", ")+". Pra separar, manda !economia desvincular em cada um deles")
}

func (b *bot) showEconomyLinks(ctx context.Context, evt *events.Message, _ []string) error {
	chat := evt.Info.Chat.ToNonAD()
	groups := ledgers.members(ledgers.of(chat))
	if len(groups) == 1 {
		return b.SendText(ctx, chat, "Esta conversa usa a economia própria, sem vínculo com nenhum grupo. Pra juntar com outro: !economia vincular <nome do grupo>")
	}
	names := b.groupNames(ctx, groups)
	lines := []string{"🔗 *Grupos com a mesma economia*", ""}
	for i, group := range groups {
		tags := []string{}
		if i == 0 {
			tags = append(tags, "principal")
		}
		if group == chat {
			tags = append(tags, "este grupo")
		}
		line := "• " + names[i]
		if len(tags) > 0 {
			line += " _(" + strings.Join(tags, ", ") + ")_"
		}
		lines = append(lines, line)
	}
	lines = append(lines, "", "_Pra separar um grupo, manda !economia desvincular nele_")
	return b.SendText(ctx, chat, strings.Join(lines, "\n"))
}
