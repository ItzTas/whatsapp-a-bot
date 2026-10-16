package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"unicode"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
)

const (
	filterGroups = "grupos"
	filterPeople = "pessoas"
	blacklist    = "blacklist"
	whitelist    = "whitelist"
	filterOff    = "desligado"
)

type filterStore struct {
	db *sql.DB
}

type filterEntry struct {
	target string
	ids    []types.JID
	name   string
}

func (e filterEntry) keys() (string, string) {
	return e.ids[0].String(), e.ids[len(e.ids)-1].String()
}

func (e filterEntry) label() string {
	if e.name != "" {
		return "O grupo *" + e.name + "*"
	}
	if e.target == filterGroups {
		return "Este grupo"
	}
	return "@" + e.ids[0].User
}

func (s *filterStore) allows(ctx context.Context, e filterEntry) (bool, error) {
	first, second := e.keys()
	var mode string
	var listed bool
	err := s.db.QueryRowContext(ctx,
		`SELECT m.modo, EXISTS (
		   SELECT 1 FROM filtro_listas l
		   WHERE l.alvo = m.alvo AND l.lista = m.modo
		     AND (l.id IN (?, ?) OR l.id_alt IN (?, ?))
		 ) FROM filtro_modos m WHERE m.alvo = ?`,
		first, second, first, second, e.target,
	).Scan(&mode, &listed)
	if errors.Is(err, sql.ErrNoRows) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	return listed == (mode == whitelist), nil
}

func (s *filterStore) add(ctx context.Context, list string, e filterEntry) (bool, error) {
	first, second := e.keys()
	alt := ""
	if len(e.ids) > 1 {
		alt = second
	}
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO filtro_listas (alvo, lista, id, id_alt)
		 SELECT ?, ?, ?, ?
		 WHERE NOT EXISTS (
		   SELECT 1 FROM filtro_listas
		   WHERE alvo = ? AND lista = ? AND (id IN (?, ?) OR id_alt IN (?, ?))
		 )`,
		e.target, list, first, alt, e.target, list, first, second, first, second,
	)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

func (s *filterStore) remove(ctx context.Context, list string, e filterEntry) (bool, error) {
	first, second := e.keys()
	res, err := s.db.ExecContext(ctx,
		"DELETE FROM filtro_listas WHERE alvo = ? AND lista = ? AND (id IN (?, ?) OR id_alt IN (?, ?))",
		e.target, list, first, second, first, second,
	)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

func (s *filterStore) setMode(ctx context.Context, target, mode string) error {
	if mode == filterOff {
		_, err := s.db.ExecContext(ctx, "DELETE FROM filtro_modos WHERE alvo = ?", target)
		return err
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO filtro_modos (alvo, modo) VALUES (?, ?)
		 ON CONFLICT (alvo) DO UPDATE SET modo = excluded.modo`,
		target, mode,
	)
	return err
}

func (s *filterStore) modes(ctx context.Context) (modes map[string]string, err error) {
	rows, err := s.db.QueryContext(ctx, "SELECT alvo, modo FROM filtro_modos")
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()

	modes = map[string]string{filterGroups: filterOff, filterPeople: filterOff}
	for rows.Next() {
		var target, mode string
		if err := rows.Scan(&target, &mode); err != nil {
			return nil, err
		}
		modes[target] = mode
	}
	return modes, rows.Err()
}

func (s *filterStore) counts(ctx context.Context) (counts map[[2]string]int, err error) {
	rows, err := s.db.QueryContext(ctx, "SELECT alvo, lista, COUNT(*) FROM filtro_listas GROUP BY alvo, lista")
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()

	counts = map[[2]string]int{}
	for rows.Next() {
		var target, list string
		var n int
		if err := rows.Scan(&target, &list, &n); err != nil {
			return nil, err
		}
		counts[[2]string{target, list}] = n
	}
	return counts, rows.Err()
}

func (s *filterStore) listed(ctx context.Context, list string) (entries map[string][]types.JID, err error) {
	rows, err := s.db.QueryContext(ctx, "SELECT alvo, id FROM filtro_listas WHERE lista = ? ORDER BY rowid", list)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()

	entries = map[string][]types.JID{}
	for rows.Next() {
		var target, id string
		if err := rows.Scan(&target, &id); err != nil {
			return nil, err
		}
		jid, err := types.ParseJID(id)
		if err != nil {
			return nil, fmt.Errorf("id %q inválido na %s: %w", id, list, err)
		}
		entries[target] = append(entries[target], jid)
	}
	return entries, rows.Err()
}

func ownersFromEnv() []string {
	var owners []string
	for _, field := range strings.Split(os.Getenv("BOT_DONO"), ",") {
		if number := digitsOnly(field); number != "" {
			owners = append(owners, number)
		}
	}
	return owners
}

func digitsOnly(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsDigit(r) {
			return r
		}
		return -1
	}, s)
}

func (b *bot) senderIdentities(ctx context.Context, evt *events.Message) []types.JID {
	if evt.Info.SenderAlt.IsEmpty() {
		return b.identities(ctx, evt.Info.Sender.ToNonAD())
	}
	return []types.JID{evt.Info.Sender.ToNonAD(), evt.Info.SenderAlt.ToNonAD()}
}

func (b *bot) isOwner(ctx context.Context, evt *events.Message) bool {
	return slices.ContainsFunc(b.senderIdentities(ctx, evt), func(id types.JID) bool {
		return id.Server == types.DefaultUserServer && slices.Contains(b.owners, id.User)
	})
}

func (b *bot) allowed(ctx context.Context, evt *events.Message) (bool, error) {
	checks := []filterEntry{{target: filterPeople, ids: b.senderIdentities(ctx, evt)}}
	if chat := evt.Info.Chat.ToNonAD(); chat.Server == types.GroupServer {
		checks = append(checks, filterEntry{target: filterGroups, ids: []types.JID{chat}})
	}
	for _, e := range checks {
		ok, err := b.filters.allows(ctx, e)
		if err != nil || !ok {
			return false, err
		}
	}
	return true, nil
}

func (b *bot) runOwnerCommand(ctx context.Context, evt *events.Message, name string) bool {
	commands := b.ownerCommands()
	i := slices.IndexFunc(commands, func(c botCommand) bool { return c.matches(name) })
	if i < 0 {
		return false
	}
	if !b.isOwner(ctx, evt) {
		b.client.Log.Warnf("!%s ignorado: %v não tá no BOT_DONO (%v)", name, b.senderIdentities(ctx, evt), b.owners)
		return false
	}
	if err := commands[i].run(ctx, evt); err != nil {
		b.client.Log.Errorf("comando !%s: %v", name, err)
	}
	return true
}

func (b *bot) ownerCommands() []botCommand {
	dono := category{title: "🔒 Dono"}
	return []botCommand{
		{
			name:     "modo",
			aliases:  []string{"filtro"},
			category: dono,
			help: []usage{
				{"!modo", "mostra o modo dos grupos e das pessoas e quantos tem em cada lista"},
				{"!modo grupos blacklist", "o bot ignora os grupos da blacklist"},
				{"!modo grupos whitelist", "o bot só funciona nos grupos da whitelist"},
				{"!modo pessoas blacklist", "o bot ignora as pessoas da blacklist (whitelist funciona igual)"},
				{"!modo grupos desligado", "desliga o filtro dos grupos, ou das pessoas com !modo pessoas desligado"},
				{"!modo ativar diversao", "nesta conversa, libera de uma vez romance, família, jogos, medidores, estatísticas e zoeira"},
				{"!modo ativar jogos", "libera só uma categoria (romance, familia, jogos, medidores, estatisticas, zoeira ou ia)"},
				{"!modo ativar jogos zoeira ia", "liga vários modos numa mensagem só (desativar também aceita vários)"},
				{"!modo ativar palavras", "nesta conversa, o bot responde algumas palavras mesmo sem o !, tipo arrogante ou nojo"},
				{"!modo ativar risada", "nesta conversa, o bot manda uma figurinha de risada quando alguém ri"},
				{"!modo ativar utilidades", "nesta conversa, o bot baixa os vídeos, imagens e gifs dos links que mandarem e libera o !baixar e o !figurinha"},
				{"!modo desativar jogos", "desliga o modo de novo (funciona com qualquer modo)"},
				{"!modo ativar jogos -- <nome do grupo>", "ativa (ou desativa) os modos no grupo com esse nome, mandando de qualquer conversa"},
			},
			run: b.modo,
		},
		{
			name:     "blacklist",
			category: dono,
			help: []usage{
				{"!blacklist", "coloca a conversa atual na blacklist: o grupo, ou a pessoa se for no privado"},
				{"!blacklist @pessoa", "coloca a pessoa marcada na blacklist (pode ser um número também, tipo 5511999999999)"},
				{"!blacklist <nome do grupo>", "coloca o grupo com esse nome na blacklist, mandando de qualquer conversa"},
				{"!blacklist -- <nome do grupo>", "tira o grupo da blacklist (-- @pessoa tira a pessoa, só -- tira a conversa atual; tirar também funciona)"},
				{"!blacklist ver", "mostra quem tá na blacklist"},
			},
			run: b.blacklist,
		},
		{
			name:     "whitelist",
			category: dono,
			help: []usage{
				{"!whitelist", "coloca a conversa atual na whitelist: o grupo, ou a pessoa se for no privado"},
				{"!whitelist @pessoa", "coloca a pessoa marcada na whitelist (pode ser um número também)"},
				{"!whitelist <nome do grupo>", "coloca o grupo com esse nome na whitelist, mandando de qualquer conversa"},
				{"!whitelist -- <nome do grupo>", "tira o grupo da whitelist (-- @pessoa tira a pessoa, só -- tira a conversa atual; tirar também funciona)"},
				{"!whitelist ver", "mostra quem tá na whitelist"},
			},
			run: b.whitelist,
		},
	}
}

func (b *bot) modo(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat
	args := strings.Fields(strings.ToLower(messageText(evt.Message)))[1:]
	if len(args) == 0 {
		return b.showFilterModes(ctx, chat)
	}
	toggles := map[string]bool{"ativar": true, "ligar": true, "desativar": false, "desligar": false}
	if on, isToggle := toggles[args[0]]; isToggle {
		return b.toggleChatMode(ctx, evt, on, args[1:])
	}

	targets := map[string]string{"grupos": filterGroups, "grupo": filterGroups, "pessoas": filterPeople, "pessoa": filterPeople}
	modes := map[string]string{blacklist: blacklist, whitelist: whitelist, filterOff: filterOff, "desligar": filterOff, "off": filterOff}
	usageText := "Use assim: !modo grupos blacklist (ou whitelist, ou desligado). Pra mexer nas pessoas, troque grupos por pessoas."
	if len(args) < 2 {
		return b.sendText(ctx, chat, usageText)
	}
	target, knownTarget := targets[args[0]]
	mode, knownMode := modes[args[1]]
	if !knownTarget || !knownMode {
		return b.sendText(ctx, chat, usageText)
	}

	var counts map[[2]string]int
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error { return b.filters.setMode(gctx, target, mode) })
	g.Go(func() (err error) { counts, err = b.filters.counts(gctx); return err })
	if err := g.Wait(); err != nil {
		return err
	}

	confirmations := map[string]string{
		filterOff: "✅ Filtro de %s desligado. O bot responde todo mundo de novo.",
		blacklist: "✅ %s em modo blacklist. O bot ignora quem tá na blacklist.",
		whitelist: "✅ %s em modo whitelist. O bot só responde quem tá na whitelist.",
	}
	text := fmt.Sprintf(confirmations[mode], target)
	if mode == whitelist && counts[[2]string{target, whitelist}] == 0 {
		text += fmt.Sprintf("\n\n⚠️ A whitelist de %s tá vazia, então o bot vai ignorar todos os %s até você colocar alguém nela.", target, target)
	}
	return b.sendText(ctx, chat, text)
}

func (b *bot) showFilterModes(ctx context.Context, chat types.JID) error {
	var (
		modes      map[string]string
		counts     map[[2]string]int
		chatActive []string
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { modes, err = b.filters.modes(gctx); return err })
	g.Go(func() (err error) { counts, err = b.filters.counts(gctx); return err })
	g.Go(func() (err error) { chatActive, err = b.modes.active(gctx, chat.ToNonAD()); return err })
	if err := g.Wait(); err != nil {
		return err
	}

	titles := map[string]string{filterGroups: "👥 Grupos", filterPeople: "🙋 Pessoas"}
	lines := []string{"⚙️ *Filtros do bot*", ""}
	for _, target := range []string{filterGroups, filterPeople} {
		lines = append(lines, fmt.Sprintf("%s: *%s* (%d na blacklist, %d na whitelist)",
			titles[target], modes[target], counts[[2]string{target, blacklist}], counts[[2]string{target, whitelist}]))
	}
	lines = append(lines, "", "_Pra mudar: !modo grupos whitelist, !modo pessoas desligado..._", "")
	lines = append(lines, chatModeLines(chatActive)...)
	lines = append(lines, "", "_Pra mudar: !modo ativar diversao, !modo desativar jogos..._")
	return b.sendText(ctx, chat, strings.Join(lines, "\n"))
}

func (b *bot) blacklist(ctx context.Context, evt *events.Message) error {
	return b.editFilterList(ctx, evt, blacklist)
}

func (b *bot) whitelist(ctx context.Context, evt *events.Message) error {
	return b.editFilterList(ctx, evt, whitelist)
}

func (b *bot) editFilterList(ctx context.Context, evt *events.Message, list string) error {
	args := strings.Fields(messageText(evt.Message))[1:]
	if len(args) > 0 && strings.EqualFold(args[0], "ver") {
		return b.showFilterList(ctx, evt.Info.Chat, list)
	}
	removing, args := removalArgs(args)

	entries, problem, err := b.filterEntries(ctx, evt, args)
	if err != nil {
		return err
	}
	if problem != "" {
		return b.sendText(ctx, evt.Info.Chat, problem)
	}
	results := map[[2]bool]string{
		{false, true}:  "✅ %s entrou na %s",
		{false, false}: "%s já tava na %s",
		{true, true}:   "✅ %s saiu da %s",
		{true, false}:  "%s não tava na %s",
	}
	lines := make([]string, len(entries))
	var mentions []types.JID
	for i, e := range entries {
		change := b.filters.add
		if removing {
			change = b.filters.remove
		}
		changed, err := change(ctx, list, e)
		if err != nil {
			return err
		}
		lines[i] = fmt.Sprintf(results[[2]bool{removing, changed}], e.label(), list)
		if e.target == filterPeople {
			mentions = append(mentions, e.ids[0])
		}
	}

	modes, err := b.filters.modes(ctx)
	if err != nil {
		return err
	}
	if target := entries[0].target; modes[target] != list {
		lines = append(lines, "", fmt.Sprintf("_A %s de %s só vale com !modo %s %s (agora tá em %s)_", list, target, target, list, modes[target]))
	}
	_, err = b.sendMentions(ctx, evt.Info.Chat, strings.Join(lines, "\n"), mentions)
	return err
}

func removalArgs(args []string) (bool, []string) {
	if len(args) == 0 {
		return false, args
	}
	if first := strings.ToLower(args[0]); first == "--" || first == "tirar" || first == "remover" {
		return true, args[1:]
	}
	rest, removing := strings.CutPrefix(args[0], "--")
	if !removing {
		return false, args
	}
	return true, append([]string{rest}, args[1:]...)
}

func (b *bot) filterEntries(ctx context.Context, evt *events.Message, args []string) ([]filterEntry, string, error) {
	people := mentionedJIDs(evt.Message)
	var nameWords []string
	for _, arg := range args {
		if strings.HasPrefix(arg, "@") {
			continue
		}
		if number := digitsOnly(arg); len(number) >= 8 && number == strings.TrimPrefix(arg, "+") {
			people = append(people, types.NewJID(number, types.DefaultUserServer))
			continue
		}
		nameWords = append(nameWords, arg)
	}
	if len(people) > 0 {
		entries := make([]filterEntry, len(people))
		for i, person := range people {
			entries[i] = filterEntry{target: filterPeople, ids: b.identities(ctx, person.ToNonAD())}
		}
		return entries, "", nil
	}
	if len(nameWords) > 0 {
		return b.groupByName(ctx, strings.Join(nameWords, " "))
	}

	chat := evt.Info.Chat.ToNonAD()
	if chat.Server == types.GroupServer {
		return []filterEntry{{target: filterGroups, ids: []types.JID{chat}}}, "", nil
	}
	return []filterEntry{{target: filterPeople, ids: b.identities(ctx, chat)}}, "", nil
}

func (b *bot) groupByName(ctx context.Context, name string) ([]filterEntry, string, error) {
	groups, err := b.client.GetJoinedGroups(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("listar grupos: %w", err)
	}
	wanted := foldLetters(name)
	var exact, partial []*types.GroupInfo
	for _, group := range groups {
		folded := foldLetters(group.Name)
		if folded == wanted {
			exact = append(exact, group)
		}
		if strings.Contains(folded, wanted) {
			partial = append(partial, group)
		}
	}

	matches := exact
	if len(matches) == 0 {
		matches = partial
	}
	if len(matches) == 1 {
		return []filterEntry{{target: filterGroups, ids: []types.JID{matches[0].JID}, name: matches[0].Name}}, "", nil
	}
	if len(matches) == 0 {
		return nil, fmt.Sprintf("Não achei nenhum grupo com \"%s\" no nome.", name), nil
	}
	lines := []string{"Achei mais de um grupo com esse nome:"}
	for _, group := range matches {
		lines = append(lines, "• "+group.Name)
	}
	lines = append(lines, "", "Manda o nome mais completo.")
	return nil, strings.Join(lines, "\n"), nil
}

func (b *bot) showFilterList(ctx context.Context, chat types.JID, list string) error {
	var (
		entries map[string][]types.JID
		modes   map[string]string
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { entries, err = b.filters.listed(gctx, list); return err })
	g.Go(func() (err error) { modes, err = b.filters.modes(gctx); return err })
	if err := g.Wait(); err != nil {
		return err
	}

	groupNames, err := b.groupNames(ctx, entries[filterGroups])
	if err != nil {
		return err
	}
	people := entries[filterPeople]
	personTags := make([]string, len(people))
	for i, person := range people {
		personTags[i] = "@" + person.User
	}

	titles := map[string]string{blacklist: "🚫 *Blacklist*", whitelist: "✅ *Whitelist*"}
	lines := []string{titles[list]}
	sections := []struct {
		target string
		title  string
		items  []string
	}{
		{filterGroups, "👥 *Grupos*", groupNames},
		{filterPeople, "🙋 *Pessoas*", personTags},
	}
	for _, s := range sections {
		status := "valendo"
		if modes[s.target] != list {
			status = fmt.Sprintf("não tá valendo, o modo de %s tá em %s", s.target, modes[s.target])
		}
		lines = append(lines, "", fmt.Sprintf("%s _(%s)_", s.title, status))
		if len(s.items) == 0 {
			lines = append(lines, "_ninguém_")
		}
		for _, item := range s.items {
			lines = append(lines, "• "+item)
		}
	}
	_, err = b.sendMentions(ctx, chat, strings.Join(lines, "\n"), people)
	return err
}

func (b *bot) groupNames(ctx context.Context, groups []types.JID) ([]string, error) {
	names := make([]string, len(groups))
	var g errgroup.Group
	g.SetLimit(5)
	for i, group := range groups {
		g.Go(func() error {
			info, err := b.client.GetGroupInfo(ctx, group)
			if err != nil {
				b.client.Log.Warnf("ler nome do grupo %s: %v", group, err)
				names[i] = group.User
				return nil
			}
			names[i] = info.Name
			return nil
		})
	}
	return names, g.Wait()
}
