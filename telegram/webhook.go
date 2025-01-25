package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"uludag/database"
	"uludag/otomasyon"
)

type Server struct {
	TelegramToken string `json:"telegram_token"`
	Port          string `json:"port"`
	botID         string
	server        *http.Server
	bot           *TelegramBot
	fetcher       *otomasyon.UludagFetcher
	database      *database.Database
}

type Chat struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
}

type MessageEntity struct {
	Type   string `json:"type"`
	Offset int    `json:"offset"`
	Length int    `json:"length"`
}

type User struct {
	ID        int    `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
	IsBot     bool   `json:"is_bot"`
}

type Message struct {
	Chat           Chat            `json:"chat"`
	From           User            `json:"from"`
	ReplyToMessage *Message        `json:"reply_to_message"`
	Entities       []MessageEntity `json:"entities"`
	Text           string          `json:"text"`
}

type Update struct {
	Message Message `json:"message"`
}

const msg1 = "Lütfen bu mesajı yanıtlayarak öğrenci numaranızı ve şifrenizi boşluk bırakarak girin."

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

	bodyContent := make([]byte, r.ContentLength)
	r.Body.Read(bodyContent)

	if err := json.Unmarshal(bodyContent, &update); err != nil {
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
		respond = msg1
		isForceReply = true
	case "/logout":
		err := s.database.DeleteUser(chatID)
		if err != nil {
			respond = "Çıkış yapılırken bir hata oluştu."
			break
		}

		respond = "Başarıyla çıkış yapıldı\\!"
	case "/sinavlar":
		user, err := s.database.GetUser(chatID)
		if err != nil {
			respond = "Önce giriş yapmalısınız. /login komutunu kullanarak giriş yapabilirsiniz."
			break
		}

		student := otomasyon.Student{
			StudentID:           user.StudentID,
			StudentSessionToken: user.StudentSessionToken,
		}

		// Check token
		ok, err := s.fetcher.CheckStudentToken(student)
		if !ok || err != nil {
			respond = "Token geçersiz. Giriş yapmalısınız. /login komutunu kullanarak giriş yapabilirsiniz."
			break
		}

		results, err := s.fetcher.GetExamResults(student)

		respond = "*Sınav Sonuçları*\n"
		for _, result := range results {
			respond += fmt.Sprintf("*%s*: %.2f\n\n", result.ExamName, result.ExamGrade)
		}
	case "/help":
		respond = "Bot komutları:\n\n" +
			"/start: Botu başlatır.\n" +
			"/login: Botu aktif hâle getirir.\n" +
			"/logout: Botu pasif hâle getirir.\n" +
			"/sinavlar: Sınav sonuçlarını gösterir.\n" +
			"/yemekhane: Günün Yemekhane menüsünü gösterir.\n" +
			"/profil: Öğrenci bilgilerini gösterir.\n" +
			"/notkarti: Not kartını gösterir.\n" +
			"/dersprogrami: Ders programını gösterir.\n" +
			"/help: Yardım menüsünü gösterir."
	case "/yemekhane":
		respond = s.GetTodaysRefactoryMenu()
	case "/profil":
		respond = s.GetStudentInfo(chatID)
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
	default:
		respond = s.handleReplies(update.Message)
	}

	// Send the response
	err := s.bot.SendMessage(MessageOptions{
		ChatID:    chatID,
		Text:      respond,
		ParseMode: "markdown",
		ReplyMarkup: ReplyMarkup{
			ForceReply: isForceReply,
			Selective:  false,
		},
	})

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}

	w.WriteHeader(http.StatusOK)
	return
}

func (s *Server) Start() {
	http.HandleFunc("/webhook", s.webhookHandler)
	s.server.Addr = ":" + s.Port
	s.server.ListenAndServe()
}

func (s *Server) Stop(ctx context.Context) {
	s.server.Shutdown(ctx)
}

func (s *Server) getStudent(chatID string) (*otomasyon.Student, string) {
	user, err := s.database.GetUser(chatID)
	if err != nil {
		return nil, "Önce giriş yapmalısınız. /login komutunu kullanarak giriş yapabilirsiniz."
	}

	student := otomasyon.Student{
		StudentID:           user.StudentID,
		StudentSessionToken: user.StudentSessionToken,
	}

	// Check token
	ok, err := s.fetcher.CheckStudentToken(student)
	if !ok || err != nil {
		return nil, "Token geçersiz. Giriş yapmalısınız. /login komutunu kullanarak giriş yapabilirsiniz."
	}

	return &student, ""
}

func (s *Server) getSyllabus(student otomasyon.Student) string {
	var respond string
	entries, err := s.fetcher.GetSyllabus(student)
	if err != nil {
		return "Ders programı alınırken bir hata oluştu."

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
		return "Öğrencinin bölünleri getirilirken bilinmeyen bir hata meydana geldi!"
	}

	for i, result := range results {
		student.Branch = result.DepartmentID
		respond += "*" + result.DepartmentName + ":*\n\n"
		semesters, err := s.fetcher.GetGradeCard(student)
		if err != nil {
			return "Not kartı çekilirken bilinmeyen bir hata oluştu!"
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

func (s *Server) GetStudentInfo(chatID string) string {
	var respond string
	user, err := s.database.GetUser(chatID)
	if err != nil {
		return "Önce giriş yapmalısınız. /login komutunu kullanarak giriş yapabilirsiniz."

	}

	student := otomasyon.Student{
		StudentID:           user.StudentID,
		StudentSessionToken: user.StudentSessionToken,
	}

	profile, err := s.fetcher.GetStudentInfo(student)
	if err != nil {
		return "Öğrenci bilgileri alınırken bir hata oluştu."
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
		respond = "Yemekhane menüsü alınırken bir hata oluştu."
		return respond
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

		if repliedTo.From.ID != id && repliedTo.Text != msg1 {
			return ""
		}

		// Check if already logged in
		_, err = s.database.GetUser(chatID)
		if err == nil {
			return "Zaten giriş yapmışsınız. Çıkış yapmak için /logout komutunu kullanabilirsiniz."
		}

		username, password, found := strings.Cut(message.Text, " ")
		if !found {
			return ""
		}

		token, ok, err := s.fetcher.StudentLogin(username, password)
		if !ok || err != nil {
			return "Giriş başarısız. Lütfen /login komutunu girerek tekrar deneyin."
		}

		err = s.database.DeleteUser(chatID)
		if err != nil {
			return "Giriş başarısız. Lütfen /login komutunu girerek tekrar deneyin."
		}

		err = s.database.SaveUser(database.User{
			StudentID:           username,
			ChatID:              chatID,
			StudentSessionToken: token,
		})
		if err != nil {
			return "Giriş başarısız. Lütfen /login komutunu girerek tekrar deneyin."
		}

		return "Başarıyla giriş yaptınız. Artık sınavlarınızı görebilirsiniz. Çıkış yapmak için /logout komutunu kullanabilirsiniz."
	}

	return "Bilinmeyen komut. Komutları listelemek için /help kullanabilirsin."
}
