package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"time"
	"uludag/database"
	"uludag/otomasyon"
	"uludag/task"
	"uludag/telegram"

	"github.com/go-co-op/gocron/v2"
	"github.com/rs/zerolog/log"
)

var port string
var botToken string
var botID string

func init() {
	// Parse environment variables
	port = os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	botToken = os.Getenv("BOT_TOKEN")
	if botToken == "" {
		panic("BOT_TOKEN must be set")
	}

	botID = os.Getenv("BOT_ID")
	if botID == "" {
		panic("BOT_ID must be set")
	}
}

func main() {
	var err error

	database, err := database.NewDatabase("./data/users.db")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create database connection")
	}

	// Create data fetcher
	fetcher := otomasyon.NewUludagFetcher()

	// Create telegram bot client
	bot := telegram.NewTelegramBot(botToken)

	// Create webhook server
	server := telegram.NewServer(botToken, port, bot, fetcher, database, botID)

	s, err := gocron.NewScheduler()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create task scheduler")
	}

	// Create exam notifier task
	oldExamsPath, err := filepath.Abs("./data/old_exams.txt")
	if err != nil {
		panic(err)
	}
	notifier := task.NewExamNotifier(database, fetcher, bot, oldExamsPath)

	_, err = s.NewJob(
		gocron.DurationJob(15*time.Second),
		gocron.NewTask(notifier.Notifier),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create task")
	}

	// Start task scheduler
	s.Start()
	log.Info().Msg("Task scheduler started")

	// Start webserver (non-blocking)
	go func() {
		server.Start()
	}()

	// Graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	<-ctx.Done()

	slog.Info("Server is shutdown!")

	if err := s.Shutdown(); err != nil {
		log.Error().Err(err).Msg("Failed to shutdown scheduler")
	}

	if err := database.Close(); err != nil {
		log.Error().Err(err).Msg("Failed to close database connection")
	}

	server.Stop(context.Background())
}
