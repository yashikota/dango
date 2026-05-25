package reminder

import (
	"context"
	"fmt"
	"log"
	"time"
)

type Sender interface {
	SendReminder(ctx context.Context, rem Reminder) error
}

type Scheduler struct {
	repo     *Repository
	sender   Sender
	interval time.Duration
	now      func() time.Time
}

func NewScheduler(repo *Repository, sender Sender) *Scheduler {
	return &Scheduler{
		repo:     repo,
		sender:   sender,
		interval: 15 * time.Second,
		now:      time.Now,
	}
}

func (s *Scheduler) Start(ctx context.Context) {
	go s.run(ctx)
}

func (s *Scheduler) run(ctx context.Context) {
	if err := s.Tick(ctx); err != nil {
		log.Printf("reminder scheduler tick failed: %v", err)
	}

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.Tick(ctx); err != nil {
				log.Printf("reminder scheduler tick failed: %v", err)
			}
		}
	}
}

func (s *Scheduler) Tick(ctx context.Context) error {
	due, err := s.repo.Due(ctx, s.now().UTC(), 100)
	if err != nil {
		return err
	}

	for _, rem := range due {
		if err := s.sender.SendReminder(ctx, rem); err != nil {
			log.Printf("send reminder %d failed: %v", rem.ID, err)
			continue
		}
		if rem.Recurrence == nil {
			if err := s.repo.MarkCompleted(ctx, rem.ID); err != nil {
				return fmt.Errorf("mark reminder %d completed: %w", rem.ID, err)
			}
			continue
		}
		next, err := rem.Recurrence.NextAfter(s.now())
		if err != nil {
			return fmt.Errorf("compute next recurrence for reminder %d: %w", rem.ID, err)
		}
		if err := s.repo.Reschedule(ctx, rem.ID, next.UTC()); err != nil {
			return fmt.Errorf("reschedule reminder %d: %w", rem.ID, err)
		}
	}
	return nil
}

func FormatReminderMessage(rem Reminder) string {
	return fmt.Sprintf("⏰ **リマインダー**\n%s", rem.Message)
}
