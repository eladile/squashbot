package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"squashbot/telegram"
	"squashbot/webserver/server"
)

type Config struct {
	TelegramBotToken string `json:"telegram_bot_token"`
	LoginUrl         string `json:"login_url"`
	Username         string `json:"username"`
	Password         string `json:"password"`
	Username2        string `json:"username2"`
}

func main() {
	configFile := flag.String("config", "./config.json", "the config file")

	flag.Parse()

	data, err := ioutil.ReadFile(*configFile)
	if err != nil {
		log.Panic(err)
	}

	config := Config{}
	err = json.Unmarshal(data, &config)
	if err != nil {
		log.Panic(err)
	}
	tclient, err := telegram.NewClient(config.TelegramBotToken)

	if err != nil {
		panic(err)
	}
	s := server.NewServer(tclient, config.LoginUrl, config.Username, config.Password, config.Username2)
	go func() {
		for {
			updates, err := s.TelegramClient.GetUpdates(60)
			if err != nil {
				log.Println("Something went wrong getting updates: " + err.Error())
			}
			err = s.HandleUpdates(updates)
			if err != nil {
				log.Println("Something went wrong handling updates: " + err.Error())
			}
			log.Println("Finished handling interval")
		}
	}()

	select {}
}
