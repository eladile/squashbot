package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"squashbot/squash"
	"squashbot/telegram"
)

var (
	listCourtRegex = mustCompile(".*courts.*available.*|.*available.*courts.*|.*any.*courts.*")
	aliveRegex     = mustCompile(".*are you alive.*")
	bookRegex      = mustCompile(".*book.*court.*at.*")
)

func mustCompile(s string) *regexp.Regexp {
	r, err := regexp.Compile(s)
	if err != nil {
		panic(err)
	}
	return r
}

type Server struct {
	RegToHandler   map[*regexp.Regexp]func(string, string) error
	TelegramClient telegram.Client
	Loginurl       string
	Username       string
	Password       string
	Username2      string
}

func NewServer(telegram telegram.Client, loginurl, username, password, username2 string) *Server {
	s := Server{
		TelegramClient: telegram,
		Loginurl:       loginurl,
		Username:       username,
		Password:       password,
		Username2:      username2,
	}
	s.RegToHandler = map[*regexp.Regexp]func(string, string) error{
		listCourtRegex: s.handleAvailableCourts,
		aliveRegex:     s.handleAliveRequest,
		bookRegex:      s.handleBookRequest,
	}
	return &s
}

func (s *Server) HandleUpdates(updates []telegram.Update) error {
	for _, update := range updates {
		if update.Message == nil || update.Message.Text == nil {
			continue
		}
		for regx, handler := range s.RegToHandler {
			if text := strings.ToLower(*update.Message.Text); regx.MatchString(text) {
				if err := handler(fmt.Sprintf("%d", update.Message.Chat.Id), text); err != nil {
					log.Printf("Can't handle text %s that matched %s, got error:%s\n",
						text, regx.String(), err.Error())
				}
			}
		}
	}
	return nil
}

func (s *Server) handleAliveRequest(id, text string) error {
	return s.TelegramClient.SendMessage(id, "yeah I'm fine, thanks!")
}

func (s *Server) handleAvailableCourts(id, text string) error {
	day, month, year, err := s.getDate(text)
	if err != nil {
		return fmt.Errorf("failed to fetch date: %w", err)
	}

	c, err := squash.Login(s.Loginurl, s.Username, s.Password)
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}
	res, err := c.GetAvailableCourts(day, month, year)
	if err != nil {
		return fmt.Errorf("getAvailableCourts failed: %w", err)
	}
	return s.TelegramClient.SendMessage(id, res.Telegram())
}

func (s *Server) getDate(text string) (day, month, year int, err error) {
	i := strings.Index(text, "/")
	if i < 0 || len(text)-i-2 < 0 || len(text) < i+6 {
		return 0, 0, 0, errors.New("no date or bad date format")
	}
	text = text[i-2 : i+6]
	_, err = fmt.Sscanf(text, "%02d/%02d/%02d", &day, &month, &year)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("scan failed: %w", err)
	}
	return day, month, year, err
}

func (s *Server) getTime(text string) (hour, min int, err error) {
	i := strings.Index(text, ":")
	if i < 0 || len(text)-i-2 < 0 || len(text) < i+3 {
		return 0, 0, errors.New("no time or bad time format")
	}
	text = text[i-2 : i+3]
	_, err = fmt.Sscanf(text, "%02d:%02d", &hour, &min)
	if err != nil {
		return 0, 0, fmt.Errorf("scan failed: %w", err)
	}
	return hour, min, err
}

func (s *Server) GetCourts(w http.ResponseWriter, r *http.Request) {
	var err error
	grabInt := func(key string) int {
		if err != nil {
			return 0
		}
		v := r.FormValue(key)
		if v == "" {
			err = errors.New("missing " + key)
			return 0
		}
		var ret int
		ret, err = strconv.Atoi(v)
		return ret
	}
	day := grabInt("day")
	month := grabInt("month")
	year := grabInt("year")
	if err != nil {
		http.Error(w, "Bad user input "+err.Error(), http.StatusBadRequest)
		return
	}

	c, err := squash.Login(s.Loginurl, s.Username, s.Password)
	if err != nil {
		http.Error(w, "Login failed "+err.Error(), http.StatusInternalServerError)
		return
	}
	res, err := c.GetAvailableCourts(day, month, year)
	if err != nil {
		http.Error(w, "Failed to fetch courts "+err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, res)

}

func jsonResponse(w http.ResponseWriter, obj interface{}) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	err := enc.Encode(obj)
	if err != nil {
		log.Println("Failed encoding json response", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleBookRequest(id string, text string) (err error) {
	defer func() {
		if err != nil {
			_ = s.TelegramClient.SendMessage(id, fmt.Sprintf("erorr occured while booking: %s", err.Error()))
		}
	}()

	day, month, year, err := s.getDate(text)
	if err != nil {
		return err
	}

	c, err := squash.Login(s.Loginurl, s.Username, s.Password)
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}
	res, err := c.GetAvailableCourts(day, month, year)
	if err != nil {
		return fmt.Errorf("failed to fetch available courts: %w", err)
	}

	hour, min, err := s.getTime(text)
	if err != nil {
		return fmt.Errorf("failed to get time: %w", err)
	}

	// the biggest the court number the better court it is
	chosenCourt := -1
	for court, records := range res {
		for _, record := range records {
			h, m, err := s.getTime(record.Hour)
			if err != nil {
				continue
			}
			if h == hour && m == min {
				if court > chosenCourt {
					chosenCourt = court
				}
			}
		}
	}

	err = c.Book(s.Username, s.Username2, day, month, year, hour, min, chosenCourt)
	if err != nil {
		return err
	}
	_ = s.TelegramClient.SendMessage(id, fmt.Sprintf("Booked court #%d at %2d/%2d/%2d %2d:%2d, other user (%s) needs to approve", chosenCourt, day, month, year, min, hour, s.Username2))
	return nil
}
