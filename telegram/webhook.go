package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"uludag/database"
	"uludag/otomasyon"

	"github.com/rs/zerolog/log"
)

type Server struct {
	server        *http.Server
	bot           *TelegramBot
	fetcher       *otomasyon.UludagFetcher
	database      *database.Database
	TelegramToken string
	Port          string
	botID         string
}

// Telegram types
type Chat struct {
	Username string `json:"username"`
	ID       int    `json:"id"`
}

type MessageEntity struct {
	Type   string `json:"type"`
	Offset int    `json:"offset"`
	Length int    `json:"length"`
}

type User struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
	ID        int    `json:"id"`
	IsBot     bool   `json:"is_bot"`
}

type Message struct {
	ReplyToMessage *Message        `json:"reply_to_message"`
	Chat           Chat            `json:"chat"`
	Text           string          `json:"text"`
	From           User            `json:"from"`
	Entities       []MessageEntity `json:"entities"`
}

type Update struct {
	Message Message `json:"message"`
}

func NewServer(token string, port string, bot *TelegramBot, fetcher *otomasyon.UludagFetcher, database *database.Database, botID string) *Server {
	return &Server{
		TelegramToken: token,
		Port:          port,
		server:        &http.Server{},
		bot:           bot,
		fetcher:       fetcher,
		database:      database,
		botID:         botID,
	}
}

func (s *Server) webhookHandler(w http.ResponseWriter, r *http.Request) {
	// Parse the request body
	var update Update

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error().Err(err).Msg("Failed to read request body")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := json.Unmarshal(body, &update); err != nil {
		log.Error().Err(err).Msg("Failed to unmarshal request body")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Extract the message from the update
	chatID := strconv.Itoa(update.Message.Chat.ID)
	message := update.Message.Text
	username := update.Message.Chat.Username

	var respond string
	var isForceReply bool

	// Skip if message is from bot
	if update.Message.From.IsBot {
		w.WriteHeader(http.StatusOK)
		return
	}

	// start, login logout, sinavlar
	switch message {
	case "/start":
		respond = "Merhaba, " + username + "! Bot'a hoşgeldin. Botu aktif hâle getirmek için /login komutunu kullanabilirsin."
	case "/login":
		respond = LoginReplyMessage
		isForceReply = true
	case "/logout":
		err := s.database.DeleteUser(chatID)
		if err != nil {
			log.Error().Err(err).Msg("Failed to delete user")
			respond = LogoutErrorMessage
			break
		}

		respond = LogoutSuccessMessage
	case "/sinavlar":
		student, output := s.getStudent(chatID)
		if student == nil && output != "" {
			respond = output
			break
		}

		results, err := s.fetcher.GetExamResults(*student)
		if err != nil {
			respond = ExamResultsErrorMessage
			break
		}

		respond = "*Sınav Sonuçları*\n"
		for _, result := range results {
			respond += fmt.Sprintf("*%s*: %.2f\n\n", result.ExamName, result.ExamGrade)
		}
	case "/yemekhane":
		respond = s.GetTodaysRefactoryMenu()
	case "/profil":
		student, output := s.getStudent(chatID)
		if student == nil && output != "" {
			respond = output
			break
		}

		respond = s.GetStudentInfo(*student)
	case "/notkarti":
		student, output := s.getStudent(chatID)
		if student == nil && output != "" {
			respond = output
			break
		}

		respond = s.getGradeCard(*student)
	case "/dersprogrami":
		student, output := s.getStudent(chatID)
		if student == nil && output != "" {
			respond = output
			break
		}

		respond = s.getSyllabus(*student)
	case "/sinavprogrami":
		student, output := s.getStudent(chatID)
		if student == nil && output != "" {
			respond = output
			break
		}

		respond = s.GetExamSchedule(*student)
	case "/help":
		respond = HelpMessage
	default:
		respond = s.handleReplies(update.Message)
	}

	// Check empty response
	if respond == "" {
		respond = UnknownErrorMessage
	}

	// Send the response
	if err = s.bot.SendMessage(MessageOptions{
		ChatID:    chatID,
		Text:      respond,
		ParseMode: "markdown",
		ReplyMarkup: ReplyMarkup{
			ForceReply: isForceReply,
			Selective:  false,
		},
	}); err != nil {
		log.Error().Err(err).Msg("Failed to send message")
		w.WriteHeader(http.StatusInternalServerError)
	}

	w.WriteHeader(http.StatusOK)
	return
}

func (s *Server) Start() {
	http.HandleFunc("/webhook", s.webhookHandler)
	s.server.Addr = ":" + s.Port

	log.Info().Msg("Starting server on port " + s.Port)
	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal().Err(err).Msg("Failed to start server")
	}
}

func (s *Server) Stop(ctx context.Context) {
	if err := s.server.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("Failed to shutdown server")
	}
}

func (s *Server) GetExamSchedule(student otomasyon.Student) string {
	var respond string

	exams, err := s.fetcher.GetExamSchedule(student)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch exam schedule")
		return ExamScheduleErrorMessage
	}

	examIDs := []int{2, 3, 4, 10}
	examNames := []string{"Vize", "Final", "Büt", "Ödev"}

	respond = "*Sınav Programı*\n\n"
	for i, name := range examNames {
		examEntries := make([]otomasyon.Exam, 0, len(exams))
		for _, entry := range exams {
			if entry.ExamTypeID == examIDs[i] {
				examEntries = append(examEntries, entry)
			}
		}

		// Skip if there is no entry for that exam type
		if len(examEntries) == 0 {
			continue
		}

		respond += "*" + name + "*\n"
		for _, entry := range examEntries {
			respond += "- *" + entry.ExamName + "* - " + entry.ExamDate + " " + entry.ExamTime + ", " + entry.ExamDuration + " dakika\n"
		}
		respond += "\n"
	}

	return respond
}

func (s *Server) getStudent(chatID string) (*otomasyon.Student, string) {
	user, err := s.database.GetUser(chatID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch user")
		return nil, NotLoggedInMessage
	}

	student := otomasyon.Student{
		StudentID:           user.StudentID,
		StudentSessionToken: user.StudentSessionToken,
	}

	// Check token
	ok, err := s.fetcher.CheckStudentToken(student)
	if !ok || err != nil {
		return nil, TokenErrorMessage
	}

	return &student, ""
}

func (s *Server) getSyllabus(student otomasyon.Student) string {
	var respond string
	entries, err := s.fetcher.GetSyllabus(student)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch syllabus")
		return SyllabusErrorMessage

	}

	respond = "*Ders Programı*\n\n"

	days := []string{"Pazartesi", "Salı", "Çarşamba", "Perşembe", "Cuma"}
	for i, day := range days {
		dayEntries := make([]otomasyon.SyllabusEntry, 0, len(entries))
		for _, entry := range entries {
			if entry.Day == i+1 && entry.Exists == 1 {
				dayEntries = append(dayEntries, entry)
			}
		}

		// Skip if there is no entry for that day
		if len(dayEntries) == 0 {
			continue
		}

		respond += "*" + day + "*\n"
		for _, entry := range dayEntries {
			respond += entry.ClassCode + " - " + entry.Hours + "\n"
		}
		respond += "\n"
	}

	return respond
}

func (s *Server) getGradeCard(student otomasyon.Student) string {
	var respond string

	results, err := s.fetcher.GetStudentBranches(student)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch student branches")
		return StudentBranchesErrorMessage
	}

	for i, result := range results {
		student.Branch = result.DepartmentID
		respond += "*" + result.DepartmentName + ":*\n\n"
		semesters, err := s.fetcher.GetGradeCard(student)
		if err != nil {
			return GradeCardErrorMessage
		}

		for _, semester := range semesters {
			respond += "*Dönem: " + semester.SemesterName + "\nDönem Kredisi: " + semester.SemesterECTS + " - Toplam Kredi: " + semester.TotalECTS + "*\n"
			respond += "ANO: " + semester.SemesterANO + " GANO: " + semester.GANO + "\n"

			respond += "*Harf Notları:*\n"
			for _, grade := range semester.Grades {
				respond += "- " + grade.CourseName + ": " + grade.Grade + " (" + grade.ECTS + " kredi)\n"
			}
			respond += "\n"
		}

		if i != len(results)-1 {
			respond += "\n"
		}
	}

	return respond
}

func (s *Server) GetStudentInfo(student otomasyon.Student) string {
	var respond string

	profile, err := s.fetcher.GetStudentInfo(student)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch student info")
		return StudentInfoErrorMessage
	}

	respond = "*Kişisel Bilgiler*\n\n"
	respond += "*Öğrenci Ad - Soyad:* " + profile.Name + " " + profile.Surname + "\n"
	respond += "*Öğrenci Uyruğu:* " + profile.Nationality + "\n"
	respond += "*Bölümler:*\n"

	for _, department := range profile.Departments {
		respond += "  - Numara: *" + department.StudentID + "*, Bölüm: " + department.DepartmentName + ", " + department.DepartmentYear + ". yıl " + department.DepartmentSemester + ". dönem" + "\n"
	}

	return respond
}

func (s *Server) GetTodaysRefactoryMenu() string {
	var respond string

	refactory, err := s.fetcher.GetRefactoryList()
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch refactory list")
		return RefactoryMenuErrorMessage
	}

	respond = "*Günün Yemekhane Menüsü*\n\n"
	respond += "*Öğle Yemeği:*\n"

	lunch_calories := strings.Split(refactory.Okalori, "\n")
	lunch_menu := strings.Split(refactory.Ogle, "\n")
	for i, menu := range lunch_menu {
		if menu == "" {
			continue
		}

		calory := "-"
		if i < len(lunch_calories) {
			calory = lunch_calories[i]
		}
		respond += menu + ": _" + calory + " kalori_" + "\n"
	}

	respond += "\n*Akşam Yemeği:*\n"

	dinner_calories := strings.Split(refactory.Akalori, "\n")
	dinner_menu := strings.Split(refactory.Aksam, "\n")

	for i, menu := range dinner_menu {
		if menu == "" {
			continue
		}

		calory := "-"
		if i < len(dinner_calories) {
			calory = dinner_calories[i]
		}
		respond += menu + ": _" + calory + " kalori_" + "\n"
	}

	return respond
}

func (s *Server) handleReplies(message Message) string {
	chatID := strconv.Itoa(message.Chat.ID)
	repliedTo := message.ReplyToMessage

	if repliedTo != nil {
		// Check if it is login message
		id, err := strconv.Atoi(s.botID)
		if err != nil {
			return ""
		}

		if repliedTo.From.ID != id || repliedTo.Text != LoginReplyMessage {
			return ""
		}

		// Check if already logged in
		_, err = s.database.GetUser(chatID)
		if err == nil {
			log.Error().Err(err).Msg("User already logged in")
			return AlreadyLoggedInMessage
		}

		username, password, found := strings.Cut(message.Text, " ")
		if !found {
			return ""
		}

		token, ok, err := s.fetcher.StudentLogin(username, password)
		if !ok || err != nil {
			log.Error().Err(err).Msg("Failed to login")
			return LoginErrorMessage
		}

		err = s.database.DeleteUser(chatID)
		if err != nil {
			return LoginErrorMessage
		}

		err = s.database.SaveUser(database.User{
			StudentID:           username,
			ChatID:              chatID,
			StudentSessionToken: token,
		})
		if err != nil {
			log.Error().Err(err).Msg("Failed to save user")
			return LoginErrorMessage
		}

		return LoginSuccessMessage
	}

	return UnknownCommandMessage
}
