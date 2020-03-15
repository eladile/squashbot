package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"squashbot/squash"
	"squashbot/telegram"
	"strconv"
	"strings"
)

var (
	listCourtRegex = mustCompile(".*courts.*available.*|.*available.*courts.*|.*any.*courts.*")
	aliveRegex     = mustCompile(".*are you alive.*")
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
	var (
		day, month, year int
	)
	i := strings.Index(text, "/")
	if i < 0 || len(text)-i-2 < 0 || len(text) < i+6 {
		return nil
	}
	text = text[i-2 : i+6]
	n, err := fmt.Sscanf(text, "%02d/%02d/%02d", &day, &month, &year)
	if err != nil {
		log.Println("Scan failed ", n, err.Error())
		return err
	}

	c, err := squash.Login(s.Loginurl, s.Username, s.Password)
	if err != nil {
		return err
	}
	res, err := c.GetAvailableCourts(day, month, year)
	if err != nil {
		return err
	}
	return s.TelegramClient.SendMessage(id, res.Telegram())
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
