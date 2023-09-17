package squash

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"
)

type Client struct {
	client    http.Client
	cookieJar http.CookieJar
}

type LoginRequest struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	RememberMe bool   `json:"rememberme"`
}

type LoginResponse struct {
	Role struct {
		BitMask int    `json:"bitMask"`
		Title   string `json:"title"`
	} `json:"role"`
	FirstName string `json:"firstName"`
	UserName  string `json:"userName"`
	Id        uint64 `json:"id"`
}

func Login(loginUrl, username, password string) (*Client, error) {
	sclient := Client{}
	var err error
	sclient.cookieJar, err = cookiejar.New(&cookiejar.Options{})
	if err != nil {
		return nil, err
	}
	loginReq := LoginRequest{
		Username:   username,
		Password:   password,
		RememberMe: true,
	}
	binaryRequest, err := json.Marshal(loginReq)
	if err != nil {
		return nil, err
	}

	transport := http.Transport{
		DisableCompression: false, //only diff from http.DefaultTransport
		TLSClientConfig:    &tls.Config{InsecureSkipVerify: true},
	}
	sclient.client = http.Client{
		Transport: &transport,
		Jar:       sclient.cookieJar,
		Timeout:   20 * time.Second,
	}
	header := http.Header{}
	header.Add("Content-Type", "application/json;charset=UTF-8")
	httpReq, err := http.NewRequest(http.MethodPost, loginUrl, bytes.NewBuffer(binaryRequest))
	if err != nil {
		return nil, err
	}
	httpReq.Header = header
	res, err := sclient.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	loginRes := LoginResponse{}
	if err := json.Unmarshal(body, &loginRes); err != nil {
		return nil, err
	}
	log.Println("Logged in successfully", loginRes)

	return &sclient, nil
}

type Record struct {
	Closed       string  `json:"closed"`
	User1        *string `json:"user1"`
	User2        *string `json:"user2"`
	CourtNumber  int     `json:"id"`
	Hour         string  `json:"hour"`
	User1Id      int     `json:"u1_id"`
	User2Id      int     `json:"u2_id"`
	User1Confirm *int    `json:"user1Confirm"`
	User2Confirm *int    `json:"user2Confirm"`
}

func (r *Record) Available() bool {
	return strings.ToLower(r.Closed) == "false" && r.User1 == nil && r.User2 == nil
}

type AvailableCourts map[int][]Record

func (a AvailableCourts) Telegram() string {
	if len(a) == 0 {
		return "No courts available"
	}
	av := map[string]struct{}{}
	hours := make([]string, 0, 10)
	for _, records := range a {
		for _, record := range records {
			if _, ok := av[record.Hour]; ok {
				continue
			}
			av[record.Hour] = struct{}{}
			hours = append(hours, record.Hour)
		}
	}
	sort.Strings(hours)
	res := strings.Join(append([]string{"Courts are available at:"}, hours...), "\n")
	return res
}

func (c *Client) GetAvailableCourts(day, month, year int) (AvailableCourts, error) {
	log.Printf("fetching courts for %d-%02d-%02d\n", year, month, day)
	url := fmt.Sprintf("http://www.bamigrash.com/tlv/api/courts/getCourts/0%d-%02d-%02d", year, month, day)
	httpReq, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	res, err := c.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	courtsRecords := struct {
		Res []struct {
			Records []Record `json:"records"`
		} `json:"res"`
	}{}
	err = json.Unmarshal(body, &courtsRecords)
	if err != nil {
		return nil, err
	}
	if len(courtsRecords.Res) > 4 {
		return nil, errors.New("Unexpected courts result (expected 4 courts)")
	}
	courts := make(map[int][]Record, 4)
	for i, _ := range courtsRecords.Res {
		for _, record := range courtsRecords.Res[i].Records {
			if !record.Available() {
				continue
			}
			c, ok := courts[record.CourtNumber]
			if !ok {
				c = make([]Record, 0, 10)
				courts[record.CourtNumber] = c
			}
			courts[record.CourtNumber] = append(c, record)
		}
	}
	return courts, nil
}

type InnerReservation struct {
	Player1     string `json:"player1"`
	Player2     string `json:"player2"`
	Hour        string `json:"hour"`
	Date        string `json:"date"`
	CourtNumber int    `json:"id"`
}

type Reservation struct {
	Res      string  `json:"reservation"`
	Override *string `json:"overrideDecision"`
}

func (c *Client) Book(user, otherUser string, day, month, year, hour, min, court int) error {
	url := "http://www.bamigrash.com/tlv/api/reservation/addReservation"
	inner := InnerReservation{
		Player1:     otherUser,
		Player2:     user,
		Hour:        fmt.Sprintf("%02d:%02d", hour, min),
		Date:        fmt.Sprintf("%d-%02d-%02d", year, month, day),
		CourtNumber: court,
	}
	innerTxt, err := json.Marshal(inner)
	if err != nil {
		return err
	}
	reqBody := Reservation{
		Res: string(innerTxt),
	}
	data, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}
	fmt.Println("EEEEEEEELAD", string(data))
	header := http.Header{}
	header.Add("Content-Type", "application/json;charset=UTF-8")
	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	httpReq.Header = header
	res, err := c.client.Do(httpReq)
	if err != nil {
		return err
	}
	log.Println("Booked successfully, details:", res)
	if res.StatusCode >= 300 {
		return errors.New(fmt.Sprintf("Failed booking: %s", res.Status))
	}
	return nil
}
