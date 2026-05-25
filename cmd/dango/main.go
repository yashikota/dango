package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/yashikota/dango/internal/config"
	"github.com/yashikota/dango/internal/discord"
	"github.com/yashikota/dango/internal/reminder"
)

const version = "2.0.0"

func main() {
	cfg, err := config.Load(version)
	if err != nil {
		log.Fatal(err)
	}

	repo, err := reminder.OpenRepository(cfg.DatabasePath)
	if err != nil {
		log.Fatal("open reminder database: ", err)
	}
	defer repo.Close()

	reminderService := reminder.NewService(repo, cfg.Location)
	bot, err := discord.NewBot(cfg, reminderService)
	if err != nil {
		log.Fatal(err)
	}
	if err := bot.Open(); err != nil {
		log.Fatal(err)
	}
	defer bot.Close()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	scheduler := reminder.NewScheduler(repo, bot)
	scheduler.Start(ctx)

	log.Println("dango is running. Press Ctrl+C to exit.")
	<-ctx.Done()
	log.Println("Shutting down...")
}
