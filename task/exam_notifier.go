package task

import (
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"slices"
	"strconv"
	"strings"
	"uludag/database"
	"uludag/otomasyon"
	"uludag/telegram"

	"github.com/rs/zerolog/log"
)

type ExamNotifier struct {
	database *database.Database
	fetcher  Fetcher
	bot      Bot
	location string
}

type Fetcher interface {
	GetExamResults(student otomasyon.Student) ([]otomasyon.ExamResult, error)
}

type Bot interface {
	SendMessage(options telegram.MessageOptions) error
}

func NewExamNotifier(database *database.Database, fetcher Fetcher, bot Bot, location string) *ExamNotifier {
	return &ExamNotifier{
		database: database,
		fetcher:  fetcher,
		bot:      bot,
		location: location,
	}
}

func (n *ExamNotifier) Notifier() {
	old_exams, err := os.OpenFile(n.location, os.O_RDWR, 0666)
	if errors.Is(err, os.ErrNotExist) {
		old_exams, err = os.Create(n.location)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to create old exams file")
		}
	} else if err != nil {
		log.Fatal().Err(err).Msg("Failed to open old exams file")
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
	users, err := n.database.AllUsers()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to fetch users")
	}

	for _, user := range users {
		results, err := n.fetcher.GetExamResults(otomasyon.Student{
			StudentID:           user.StudentID,
			StudentSessionToken: user.StudentSessionToken,
		})
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to fetch users")
		}

		var newExamContents string
		for _, result := range results {
			if _, ok := oldExamsMap[result.ExamID]; !ok {
				newExamsMap[result.ExamID] = struct{}{}
				n.bot.SendMessage(telegram.MessageOptions{
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

	if err := old_exams.Close(); err != nil {
		log.Fatal().Err(err).Msg("Failed to close old exams file")
	}
}
