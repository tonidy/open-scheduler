package main

import (
	"context"
	"fmt"
	"log"

	"github.com/robfig/cron/v3"
)

type Command interface {
	Execute(ctx context.Context, token string, nodeID string) error
	IntervalSeconds() int
	Name() string
}

type CommandExecutor struct {
	commands []Command
	token    string
	nodeID   string
}

func NewCommandExecutor() *CommandExecutor {
	return &CommandExecutor{
		commands: make([]Command, 0),
	}
}

func (ce *CommandExecutor) SetToken(token, nodeID string) {
	ce.token = token
	ce.nodeID = nodeID
}

func (ce *CommandExecutor) Register(cmd Command) {
	ce.commands = append(ce.commands, cmd)
}

func (ce *CommandExecutor) ExecuteAll(ctx context.Context) error {
	for _, cmd := range ce.commands {
		if err := cmd.Execute(ctx, ce.token, ce.nodeID); err != nil {
			return fmt.Errorf("command %s failed: %w", cmd.Name(), err)
		}
	}
	return nil
}

func (ce *CommandExecutor) ExecuteCommand(ctx context.Context, name string) error {
	for _, cmd := range ce.commands {
		if cmd.Name() == name {
			return cmd.Execute(ctx, ce.token, ce.nodeID)
		}
	}
	return fmt.Errorf("command %s not found", name)
}

func (ce *CommandExecutor) GetCommands() []Command {
	return ce.commands
}

func (ce *CommandExecutor) StartScheduler(ctx context.Context) {
	c := cron.New()
	for _, cmd := range ce.commands {
		interval := cmd.IntervalSeconds()
		schedule := fmt.Sprintf("@every %ds", interval)
		cmdCopy := cmd
		c.AddFunc(schedule, func() {
			if err := cmdCopy.Execute(ctx, ce.nodeID, ce.token); err != nil {
				log.Printf("Error executing command %s: %v", cmdCopy.Name(), err)
			}
		})
	}
	c.Start()

	<-ctx.Done()
	c.Stop()
}
