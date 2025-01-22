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
			respond += fmt.Sprintf("*%s*: %d\n\n", result.ExamName, result.ExamGrade)
		}
	case "/help":
		respond = "Bot komutları:\n\n" +
			"/start: Botu başlatır.\n" +
			"/login: Botu aktif hâle getirir.\n" +
			"/logout: Botu pasif hâle getirir.\n" +
			"/sinavlar: Sınav sonuçlarını gösterir.\n" +
			"/help: Yardım menüsünü gösterir."
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
