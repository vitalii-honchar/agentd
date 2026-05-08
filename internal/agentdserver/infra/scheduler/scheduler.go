package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	appscheduling "github.com/vitalii-honchar/agentd/internal/agentdserver/app/scheduling"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"

	"github.com/robfig/cron/v3"
)

type Scheduler struct {
	cron   *cron.Cron
	parser cron.Parser

	mu      sync.Mutex
	entries map[string]cron.EntryID
}

var _ appscheduling.Scheduler = (*Scheduler)(nil)

func New() *Scheduler {
	return &Scheduler{
		cron:    cron.New(),
		parser:  cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor),
		entries: make(map[string]cron.EntryID),
	}
}

func (s *Scheduler) Start(context.Context) error {
	s.cron.Start()

	return nil
}

func (s *Scheduler) Stop(context.Context) error {
	stopCtx := s.cron.Stop()
	<-stopCtx.Done()

	return nil
}

func (s *Scheduler) Reconcile(
	ctx context.Context,
	agents []domain.Agent,
	handler appscheduling.Handler,
) error {
	seen := make(map[string]struct{}, len(agents))
	for _, agent := range agents {
		seen[agent.Name] = struct{}{}
		if !agent.Enabled || agent.Status != domain.AgentStatusActive ||
			agent.Schedule.Type == domain.ScheduleTypeManual {
			s.Unschedule(agent.Name)

			continue
		}
		if agent.Schedule.Type != domain.ScheduleTypeCron {
			return fmt.Errorf("%w: schedule.type %q", domain.ErrInvalidDefinition, agent.Schedule.Type)
		}
		if err := s.schedule(ctx, agent, handler); err != nil {
			return err
		}
	}

	s.mu.Lock()
	for agentName := range s.entries {
		if _, ok := seen[agentName]; !ok {
			s.removeLocked(agentName)
		}
	}
	s.mu.Unlock()

	return nil
}

func (s *Scheduler) Unschedule(agentName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.removeLocked(agentName)

	return nil
}

func (s *Scheduler) NextRun(schedule domain.Schedule, from time.Time) (*time.Time, error) {
	if schedule.Type == domain.ScheduleTypeManual {
		return nil, nil
	}
	if schedule.Type != domain.ScheduleTypeCron {
		return nil, fmt.Errorf("%w: schedule.type %q", domain.ErrInvalidDefinition, schedule.Type)
	}

	parsed, err := s.parser.Parse(schedule.Expression)
	if err != nil {
		return nil, fmt.Errorf("%w: schedule.expression: %v", domain.ErrInvalidDefinition, err)
	}
	next := parsed.Next(from)

	return &next, nil
}

func (s *Scheduler) schedule(
	ctx context.Context,
	agent domain.Agent,
	handler appscheduling.Handler,
) error {
	s.mu.Lock()
	s.removeLocked(agent.Name)
	s.mu.Unlock()

	entryID, err := s.cron.AddFunc(agent.Schedule.Expression, func() {
		_ = handler(ctx, appscheduling.Trigger{
			AgentName: agent.Name,
			DueAt:     time.Now().UTC(),
			Source:    domain.RunTriggerSchedule,
		})
	})
	if err != nil {
		return fmt.Errorf("%w: schedule.expression: %v", domain.ErrInvalidDefinition, err)
	}

	s.mu.Lock()
	s.entries[agent.Name] = entryID
	s.mu.Unlock()

	return nil
}

func (s *Scheduler) removeLocked(agentName string) {
	if entryID, ok := s.entries[agentName]; ok {
		s.cron.Remove(entryID)
		delete(s.entries, agentName)
	}
}
