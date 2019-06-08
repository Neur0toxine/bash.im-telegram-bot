package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/joho/godotenv"
)

type updateFunc func(tgbotapi.Update, *tgbotapi.BotAPI)

const envToken = "TG_BOT_TOKEN"
const envPollTimeout = "POLL_TIMEOUT"
const envWebhook = "WEBHOOK"
const envWebhookPort = "WEBHOOK_PORT"
const envCert = "CERT"
const envKey = "CERT_KEY"
const envDebug = "DEBUG"

const defaultPollTimeout = 30
const defaultWebhookPort = 8000

func main() {
	var (
		pollTimeout, webhookPort int
		err                      error
	)

	err = godotenv.Load()

	if err != nil {
		log.Printf("WARNING: Error while loading `.env` file `%s`", err.Error())
	}

	token := os.Getenv(envToken)
	webhookUrl := os.Getenv(envWebhook)
	webhookPortStr := os.Getenv(envWebhookPort)
	certFile := os.Getenv(envCert)
	certKey := os.Getenv(envKey)
	pollTimeoutStr := os.Getenv(envPollTimeout)
	debug, err := strconv.ParseBool(os.Getenv(envDebug))

	if err != nil {
		debug = false
	}

	if pollTimeout, err = strconv.Atoi(pollTimeoutStr); err != nil {
		log.Printf("Using default poll timeout - %d seconds...", defaultPollTimeout)
		pollTimeout = defaultPollTimeout
	}

	if webhookPort, err = strconv.Atoi(webhookPortStr); err != nil && webhookUrl != "" {
		log.Printf("Using default webhook port %d ...", defaultWebhookPort)
		webhookPort = defaultWebhookPort
	}

	if token == "" {
		log.Fatalf(
			"`%s` is not found in environment - specify it in `.env` file or pass it while launching",
			envToken,
		)
	}

	bot, err := tgbotapi.NewBotAPI(token)

	if err != nil {
		log.Fatal(err)
	}

	bot.Debug = debug

	log.Printf("Authorized on account: @%s (id: %d)", bot.Self.UserName, bot.Self.ID)
	log.Printf("Debug mode: %t", debug)

	if webhookUrl == "" {
		initWithPolling(bot, pollTimeout, processUpdate)
	} else {
		initWithWebhook(bot, webhookUrl, webhookPort, certFile, certKey, processUpdate)
	}
}

func processUpdate(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
	if update.InlineQuery != nil {
		results := make([]interface{}, 1)
		results[0] = tgbotapi.NewInlineQueryResultArticleMarkdown(
			update.InlineQuery.ID,
			"",
			"",
		)

		bot.AnswerInlineQuery(
			tgbotapi.InlineConfig{
				InlineQueryID:     update.InlineQuery.ID,
				Results:           results,
				CacheTime:         30,
				IsPersonal:        false,
				NextOffset:        "",
				SwitchPMText:      "",
				SwitchPMParameter: "",
			},
		)
	}

	msgs := make([]tgbotapi.MessageConfig, 1)
	msgs[0] = tgbotapi.NewMessage(update.Message.Chat.ID, "")
	msgs[0].ParseMode = "markdown"

	if update.Message.IsCommand() {
		switch update.Message.Command() {
		case "latest":
			items, err := GetLatestQuotes()

			if err != nil {
				msgs[len(msgs)-1].Text = "Не удалось получить последние цитаты :("
			} else {
				for _, item := range items {
					text := fmt.Sprintf(
						"*Цитата:* [#%d](%s)  \n"+
							"*Создано:* %s  \n"+
							"*Рейтинг:* %s  \n"+
							"*Текст:*\n  %s    \n\n",
						item.ID,
						item.Permalink,
						item.Created,
						item.Rating,
						item.Text,
					)

					if len(msgs[len(msgs)-1].Text+text) > 4096 {
						msgs = append(msgs, tgbotapi.NewMessage(update.Message.Chat.ID, text))
						msgs[len(msgs)-1].ParseMode = "markdown"
					} else {
						msgs[len(msgs)-1].Text += text
					}
				}
			}
		default:
			msgs[len(msgs)-1].Text = "Как насчёт последних цитат? Используйте /latest"
		}
	} else {
		text := fmt.Sprintf("Что вы пытаетесь здесь найти? Тут ничего нет...\n"+
			"Бот работает не так. Зайдите в любой чат, вызовите бота вот так: @%s <id>, где ID - "+
			"это идентификатор цитаты на bash.im. И бот перешлёт её!", bot.Self.UserName)
		msgs[len(msgs)-1] = tgbotapi.NewMessage(update.Message.Chat.ID, text)
		msgs[len(msgs)-1].ReplyToMessageID = update.Message.MessageID
	}

	for _, msg := range msgs {
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Error while trying to send message to chat `%d`: %s", update.Message.Chat.ID, err.Error())
		}
	}
}

func initWithPolling(bot *tgbotapi.BotAPI, updateTimeout int, updateCallback updateFunc) {
	var (
		updates tgbotapi.UpdatesChannel
		err     error
	)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = updateTimeout

	if updates, err = bot.GetUpdatesChan(u); err != nil {
		log.Fatalf("Error while trying to get updates: %s", err.Error())
	}

	for update := range updates {
		updateCallback(update, bot)
	}
}

func initWithWebhook(
	bot *tgbotapi.BotAPI,
	webhookUrl string,
	webhookPort int,
	certFile string,
	certKey string,
	updateCallback updateFunc,
) {
	var err error

	if webhookPort == 0 {
		webhookPort = 80
	}

	if webhookUrl == "" {
		log.Fatalf("Empty webhook URL provided (env %s)", envWebhook)
	}

	webhookLink := fmt.Sprintf("%s:%d/%s", webhookUrl, webhookPort, bot.Token)

	if certFile != "" && certKey != "" {
		_, err = bot.SetWebhook(tgbotapi.NewWebhookWithCert(webhookLink, "cert.pem"))
	} else {
		_, err = bot.SetWebhook(tgbotapi.NewWebhook(webhookLink))
	}

	if err != nil {
		log.Fatal(err)
	}

	info, err := bot.GetWebhookInfo()

	if err != nil {
		log.Fatal(err)
	}

	if info.LastErrorDate != 0 {
		log.Printf("Telegram callback failed: %s", info.LastErrorMessage)
	}

	updates := bot.ListenForWebhook("/" + bot.Token)
	serverUrl := fmt.Sprintf("0.0.0.0:%d", webhookPort)

	if certFile != "" && certKey != "" {
		go http.ListenAndServeTLS(serverUrl, certFile, certKey, nil)
	} else {
		go http.ListenAndServe(serverUrl, nil)
	}

	for update := range updates {
		updateCallback(update, bot)
	}
}
