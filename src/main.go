package main

import (
	"log"
	"math/rand"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

func main() {
	cfg, err := LoadConfig()

	if err != nil {
		log.Printf("WARNING: Error while loading `.env` file `%s`", err.Error())
	}

	bot, err := tgbotapi.NewBotAPI(cfg.Token)

	if err != nil {
		log.Fatal(err)
	}

	bot.Debug = cfg.Debug

	log.Printf("Authorized on account: @%s (id: %d)", bot.Self.UserName, bot.Self.ID)
	log.Printf("Debug mode: %t", cfg.Debug)
	rand.Seed(time.Now().UTC().UnixNano())

	if cfg.WebhookURL == "" {
		initWithPolling(bot, cfg.PollingTimeout, processUpdate)
	} else {
		initWithWebhook(bot, cfg.WebhookURL, cfg.ListenAddr, cfg.CertificateFile, cfg.CertificateKey, processUpdate)
	}
}
