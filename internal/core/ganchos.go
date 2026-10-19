package core

import (
	"context"
	"slices"
	"sync"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

type observer struct {
	name string
	run  Handler
}

type modeStop struct {
	mode string
	run  func(ctx context.Context, chat types.JID)
}

type passiveAnswer struct {
	mode string
	run  Handler
}

type hooks struct {
	commands       []func() []Command
	ownerCommands  []func() []Command
	interceptors   []func(ctx context.Context, evt *events.Message) bool
	afterMessage   []func(ctx context.Context, chat types.JID)
	reactors       []func(ctx context.Context, evt *events.Message) bool
	observers      []observer
	passives       []passiveAnswer
	commandFilters []func(ctx context.Context, evt *events.Message, name string) bool
	unknown        []func(ctx context.Context, evt *events.Message, name string) error
	guards         []func(ctx context.Context, evt *events.Message, cmd Command) bool
	afterCommand   []func(ctx context.Context, evt *events.Message, cmd Command)
	restorers      map[string]GameRestorer
	jobs           []func(ctx context.Context)
	groupInfo      []func(ctx context.Context, evt *events.GroupInfo)
	modeStops      []modeStop
}

func (b *Bot) registerCore() {
	b.AddOwnerCommands(b.filterCommands)
	b.AddCommands(b.helpCommands)
}

func (b *Bot) AddCommands(provide func() []Command) {
	b.hooks.commands = append(b.hooks.commands, provide)
}

func (b *Bot) AddOwnerCommands(provide func() []Command) {
	b.hooks.ownerCommands = append(b.hooks.ownerCommands, provide)
}

func (b *Bot) AddInterceptor(intercept func(ctx context.Context, evt *events.Message) bool) {
	b.hooks.interceptors = append(b.hooks.interceptors, intercept)
}

func (b *Bot) AddAfterMessage(after func(ctx context.Context, chat types.JID)) {
	b.hooks.afterMessage = append(b.hooks.afterMessage, after)
}

func (b *Bot) AddReactor(react func(ctx context.Context, evt *events.Message) bool) {
	b.hooks.reactors = append(b.hooks.reactors, react)
}

func (b *Bot) AddObserver(name string, run Handler) {
	b.hooks.observers = append(b.hooks.observers, observer{name: name, run: run})
}

func (b *Bot) AddPassive(mode string, run Handler) {
	b.hooks.passives = append(b.hooks.passives, passiveAnswer{mode: mode, run: run})
}

func (b *Bot) AddCommandFilter(filter func(ctx context.Context, evt *events.Message, name string) bool) {
	b.hooks.commandFilters = append(b.hooks.commandFilters, filter)
}

func (b *Bot) AddUnknownCommand(answer func(ctx context.Context, evt *events.Message, name string) error) {
	b.hooks.unknown = append(b.hooks.unknown, answer)
}

func (b *Bot) AddGuard(guard func(ctx context.Context, evt *events.Message, cmd Command) bool) {
	b.hooks.guards = append(b.hooks.guards, guard)
}

func (b *Bot) AddAfterCommand(after func(ctx context.Context, evt *events.Message, cmd Command)) {
	b.hooks.afterCommand = append(b.hooks.afterCommand, after)
}

func (b *Bot) AddRestorers(restorers map[string]GameRestorer) {
	if b.hooks.restorers == nil {
		b.hooks.restorers = map[string]GameRestorer{}
	}
	for kind, restore := range restorers {
		b.hooks.restorers[kind] = restore
	}
}

func (b *Bot) AddJob(job func(ctx context.Context)) {
	b.hooks.jobs = append(b.hooks.jobs, job)
}

func (b *Bot) AddGroupInfo(handle func(ctx context.Context, evt *events.GroupInfo)) {
	b.hooks.groupInfo = append(b.hooks.groupInfo, handle)
}

func (b *Bot) OnModeOff(mode string, stop func(ctx context.Context, chat types.JID)) {
	b.hooks.modeStops = append(b.hooks.modeStops, modeStop{mode: mode, run: stop})
}

func (b *Bot) modesTurnedOff(ctx context.Context, chat types.JID, off []string) {
	for _, stop := range b.hooks.modeStops {
		if slices.Contains(off, stop.mode) {
			stop.run(ctx, chat)
		}
	}
}

func (b *Bot) commands() []Command {
	var commands []Command
	for _, provide := range b.hooks.commands {
		commands = append(commands, provide()...)
	}
	return commands
}

func (b *Bot) OwnerCommands() []Command {
	var commands []Command
	for _, provide := range b.hooks.ownerCommands {
		commands = append(commands, provide()...)
	}
	return commands
}

func (b *Bot) intercepted(ctx context.Context, evt *events.Message) bool {
	for _, intercept := range b.hooks.interceptors {
		if intercept(ctx, evt) {
			return true
		}
	}
	return false
}

func (b *Bot) afterMessage(ctx context.Context, chat types.JID) {
	for _, after := range b.hooks.afterMessage {
		after(ctx, chat)
	}
}

func (b *Bot) reactToMessage(ctx context.Context, evt *events.Message) {
	for _, react := range b.hooks.reactors {
		if react(ctx, evt) {
			return
		}
	}
}

func (b *Bot) observeMessage(ctx context.Context, evt *events.Message) {
	for _, o := range b.hooks.observers {
		if err := o.run(ctx, evt); err != nil {
			b.Client.Log.Errorf("%s: %v", o.name, err)
		}
	}
}

func (b *Bot) answerModes(ctx context.Context, evt *events.Message) {
	var wg sync.WaitGroup
	for _, p := range b.hooks.passives {
		wg.Go(func() {
			if err := p.run(ctx, evt); err != nil {
				b.Client.Log.Errorf("responder modo %s: %v", p.mode, err)
			}
		})
	}
	wg.Wait()
}

func (b *Bot) filteredCommand(ctx context.Context, evt *events.Message, name string) bool {
	for _, filter := range b.hooks.commandFilters {
		if filter(ctx, evt, name) {
			return true
		}
	}
	return false
}

func (b *Bot) answerMissingCommand(ctx context.Context, evt *events.Message, name string) {
	for _, answer := range b.hooks.unknown {
		if err := answer(ctx, evt, name); err != nil {
			b.Client.Log.Errorf("responder comando inexistente !%s: %v", name, err)
		}
	}
}

func (b *Bot) guarded(ctx context.Context, evt *events.Message, cmd Command) bool {
	for _, guard := range b.hooks.guards {
		if guard(ctx, evt, cmd) {
			return true
		}
	}
	return false
}

func (b *Bot) commandRan(ctx context.Context, evt *events.Message, cmd Command) {
	for _, after := range b.hooks.afterCommand {
		after(ctx, evt, cmd)
	}
}

func (b *Bot) StartJobs(ctx context.Context) {
	for _, job := range b.hooks.jobs {
		go job(ctx)
	}
}

func (b *Bot) HandleGroupInfo(ctx context.Context, evt *events.GroupInfo) {
	for _, handle := range b.hooks.groupInfo {
		handle(ctx, evt)
	}
}
