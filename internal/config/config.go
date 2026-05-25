package config

import (
	"fmt"
	"os"
	"time"
)

const (
	defaultDatabasePath = "dango.sqlite3"
	defaultTimezone     = "Asia/Tokyo"
)

type Config struct {
	DiscordToken string
	DatabasePath string
	TimezoneName string
	Location     *time.Location
	Version      string
}

func Load(version string) (Config, error) {
	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		return Config{}, fmt.Errorf("DISCORD_TOKEN is not set")
	}

	timezoneName := os.Getenv("DANGO_TIMEZONE")
	if timezoneName == "" {
		timezoneName = defaultTimezone
	}
	location, err := time.LoadLocation(timezoneName)
	if err != nil {
		return Config{}, fmt.Errorf("load DANGO_TIMEZONE %q: %w", timezoneName, err)
	}

	databasePath := os.Getenv("DANGO_DATABASE_PATH")
	if databasePath == "" {
		databasePath = defaultDatabasePath
	}

	return Config{
		DiscordToken: token,
		DatabasePath: databasePath,
		TimezoneName: timezoneName,
		Location:     location,
		Version:      version,
	}, nil
}
