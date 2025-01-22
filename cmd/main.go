package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"os"
	"os/signal"
	"slices"
	"strconv"
	"strings"
	"time"
	"uludag/database"
	"uludag/otomasyon"
	"uludag/telegram"

	"github.com/go-co-op/gocron/v2"
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
		panic(err)
	}

	// Create data fetcher
	fetcher := otomasyon.NewUludagFetcher()

	// Create telegram bot client
	bot := telegram.NewTelegramBot(botToken)

	// Create webhook server
	server := telegram.NewServer(botToken, port, bot, fetcher, database, botID)

	s, err := gocron.NewScheduler()
	if err != nil {
		panic(err)
	}

	_, err = s.NewJob(
		gocron.DurationJob(15*time.Second),
		gocron.NewTask(newExamNotifier, database, fetcher, bot),
	)
	if err != nil {
		panic(err)
	}

	// Start task scheduler
	s.Start()

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
		panic(err)
	}

	server.Stop(context.Background())
}

func newExamNotifier(database *database.Database, fetcher *otomasyon.UludagFetcher, bot *telegram.TelegramBot) {
	var old_exams *os.File
	var err error

	old_exams, err = os.OpenFile("./data/old_exams", os.O_RDWR, 0666)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			old_exams, err = os.Create("./data/old_exams")
			if err != nil {
				panic(err)
			}
		} else {
			panic(err)
		}
	}

	// Fetch old exams
	oldExamContents, err := io.ReadAll(old_exams)
	if err != nil {
		panic(err)
	}

	oldExamsMap := make(map[int]struct{})
	for _, examID := range strings.Split(string(oldExamContents), "\n") {
		examIDInt, err := strconv.Atoi(strings.Trim(examID, " "))
		if err != nil {
			continue
		}

		oldExamsMap[examIDInt] = struct{}{}
	}

	newExamsMap := maps.Clone(oldExamsMap)

	// Fetch new exam results
	users, err := database.AllUsers()
	if err != nil {
		panic(err)
	}

	for _, user := range users {
		results, err := fetcher.GetExamResults(otomasyon.Student{
			StudentID:           user.StudentID,
			StudentSessionToken: user.StudentSessionToken,
		})
		if err != nil {
			panic(err)
		}

		var newExamContents string
		for _, result := range results {
			if _, ok := oldExamsMap[result.ExamID]; !ok {
				newExamsMap[result.ExamID] = struct{}{}
				bot.SendMessage(telegram.MessageOptions{
					Text:   "Yeni sınav sonuçları açıklanmış!. Açıklanan sınav: " + result.ExamName,
					ChatID: user.ChatID,
				})
			}

			newExamContents += fmt.Sprintf("%d\n", result.ExamID)
		}
	}

	// Update old exams
	newExamsKeys := slices.Collect(maps.Keys(newExamsMap))
	var newExamsContent string

	for _, examID := range newExamsKeys {
		newExamsContent += fmt.Sprintf("%d\n", examID)
	}

	old_exams.Truncate(0)
	old_exams.Seek(0, 0)
	old_exams.Write([]byte(newExamsContent))
}
