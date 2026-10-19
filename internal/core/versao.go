package core

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"regexp"
	"slices"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/types"
	"golang.org/x/sync/errgroup"
)

//go:embed all:versoes
var versionFiles embed.FS

const (
	commitsFile       = "versoes/commits.txt"
	releaseNoteKey    = "Novidade:"
	updateConnectWait = time.Minute
)

type bump int

const (
	noBump bump = iota
	patchBump
	minorBump
	majorBump
)

type semver [3]int

func (v semver) String() string {
	return fmt.Sprintf("v%d.%d.%d", v[0], v[1], v[2])
}

func (v semver) bumped(by bump) semver {
	if by == noBump {
		return v
	}
	part := int(majorBump - by)
	v[part]++
	for i := part + 1; i < len(v); i++ {
		v[i] = 0
	}
	return v
}

type revision struct {
	hash    string
	version semver
	notes   []string
}

type release struct {
	version semver
	notes   []string
}

type botUpdate struct {
	head     revision
	releases []release
	chats    []types.JID
}

var commitHeader = regexp.MustCompile(`^(\w+)(?:\([^)]*\))?(!)?:`)

func commitBump(subject, body string) bump {
	header := commitHeader.FindStringSubmatch(subject)
	if header == nil {
		return patchBump
	}
	if header[2] == "!" || breaking(body) {
		return majorBump
	}
	byType := map[string]bump{
		"feat":     minorBump,
		"docs":     noBump,
		"ci":       noBump,
		"build":    noBump,
		"test":     noBump,
		"style":    noBump,
		"refactor": noBump,
	}
	if by, listed := byType[strings.ToLower(header[1])]; listed {
		return by
	}
	return patchBump
}

func breaking(body string) bool {
	return slices.ContainsFunc(strings.Split(body, "\n"), func(line string) bool {
		return strings.HasPrefix(line, "BREAKING CHANGE:") || strings.HasPrefix(line, "BREAKING-CHANGE:")
	})
}

func releaseNotes(body string) []string {
	var notes []string
	for line := range strings.SplitSeq(body, "\n") {
		note, found := strings.CutPrefix(strings.TrimSpace(line), releaseNoteKey)
		if note = strings.TrimSpace(note); found && note != "" {
			notes = append(notes, note)
		}
	}
	return notes
}

func parseRevisions(log string) []revision {
	var revisions []revision
	var version semver
	for record := range strings.SplitSeq(log, "\x1e") {
		fields := strings.SplitN(strings.TrimSpace(record), "\x1f", 3)
		if len(fields) < 2 {
			continue
		}
		body := ""
		if len(fields) == 3 {
			body = fields[2]
		}
		notes := releaseNotes(body)
		by := commitBump(fields[1], body)
		if len(notes) > 0 {
			by = max(by, patchBump)
		}
		version = version.bumped(by)
		revisions = append(revisions, revision{hash: fields[0], version: version, notes: notes})
	}
	return revisions
}

func embeddedRevisions() ([]revision, error) {
	log, err := versionFiles.ReadFile(commitsFile)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return parseRevisions(string(log)), nil
}

func pendingRevisions(revisions []revision, installed string) []revision {
	head := revisions[len(revisions)-1]
	if installed == "" {
		return slices.DeleteFunc(slices.Clone(revisions), func(r revision) bool { return r.version != head.version })
	}
	i := slices.IndexFunc(revisions, func(r revision) bool { return r.hash == installed })
	if i < 0 {
		return nil
	}
	return revisions[i+1:]
}

func releasesOf(pending []revision) []release {
	var releases []release
	for _, rev := range pending {
		if len(rev.notes) == 0 {
			continue
		}
		if last := len(releases) - 1; last >= 0 && releases[last].version == rev.version {
			releases[last].notes = append(releases[last].notes, rev.notes...)
			continue
		}
		releases = append(releases, release{version: rev.version, notes: slices.Clone(rev.notes)})
	}
	slices.Reverse(releases)
	return releases
}

func (u *botUpdate) text() string {
	lines := []string{fmt.Sprintf("🆕 *Saiu atualização do bot!* Agora tô na *%s*. Olha o que tem de novo:", u.head.version)}
	for _, r := range u.releases {
		lines = append(lines, "", "*"+r.version.String()+"*")
		for _, note := range r.notes {
			lines = append(lines, "• "+note)
		}
	}
	return strings.Join(lines, "\n")
}

type versionStore struct {
	db *sql.DB
}

func (s *versionStore) installed(ctx context.Context) (string, error) {
	var hash string
	err := s.db.QueryRowContext(ctx, "SELECT revisao FROM versoes ORDER BY instalada_em DESC LIMIT 1").Scan(&hash)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return hash, err
}

func (s *versionStore) install(ctx context.Context, rev revision, at time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO versoes (revisao, versao, instalada_em) VALUES (?, ?, ?)
		ON CONFLICT (revisao) DO UPDATE SET instalada_em = excluded.instalada_em`,
		rev.hash, rev.version.String(), at.Unix(),
	)
	return err
}

func (b *Bot) PrepareUpdate(ctx context.Context) {
	update, err := b.pendingUpdate(ctx)
	if err != nil {
		b.Client.Log.Errorf("preparar o aviso de atualização: %v", err)
		return
	}
	b.update = update
}

func (b *Bot) pendingUpdate(ctx context.Context) (*botUpdate, error) {
	revisions, err := embeddedRevisions()
	if err != nil || len(revisions) == 0 {
		return nil, err
	}
	head := revisions[len(revisions)-1]
	b.Client.Log.Infof("rodando a versão %s (%s)", head.version, head.hash)
	installed, err := b.versions.installed(ctx)
	if err != nil {
		return nil, err
	}
	pending := pendingRevisions(revisions, installed)
	if len(pending) == 0 {
		return nil, nil
	}
	update := &botUpdate{head: head, releases: releasesOf(pending)}
	if len(update.releases) == 0 {
		return update, nil
	}
	update.chats, err = b.announcementChats(ctx)
	return update, err
}

func (b *Bot) announcementChats(ctx context.Context) ([]types.JID, error) {
	chats, err := b.Modes.chatsWith(ctx, ModeAnnouncements)
	if err != nil {
		return nil, err
	}
	reached := make([]bool, len(chats))
	g, gctx := errgroup.WithContext(ctx)
	for i, chat := range chats {
		g.Go(func() error {
			on, err := b.ChatModeOn(gctx, chat, ModeAnnouncements)
			if err != nil || !on {
				return err
			}
			reached[i], err = b.AllowedIn(gctx, chat, nil)
			return err
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	var targets []types.JID
	for i, chat := range chats {
		if reached[i] {
			targets = append(targets, chat)
		}
	}
	return targets, nil
}

func (b *Bot) announceUpdate(ctx context.Context) {
	update := b.update
	if update == nil {
		return
	}
	for !b.Client.WaitForConnection(updateConnectWait) {
		if ctx.Err() != nil {
			return
		}
	}
	if err := b.sendUpdate(ctx, update); err != nil {
		b.Client.Log.Errorf("avisar a atualização: %v", err)
	}
	if err := b.versions.install(ctx, update.head, time.Now()); err != nil {
		b.Client.Log.Errorf("gravar a versão %s: %v", update.head.version, err)
	}
}

func (b *Bot) sendUpdate(ctx context.Context, update *botUpdate) error {
	text := update.text()
	errs := make([]error, len(update.chats))
	var g errgroup.Group
	for i, chat := range update.chats {
		g.Go(func() error {
			errs[i] = b.SendText(ctx, chat, text)
			return nil
		})
	}
	_ = g.Wait()
	return errors.Join(errs...)
}
