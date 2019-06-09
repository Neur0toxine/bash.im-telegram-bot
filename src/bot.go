package main

import (
	"strings"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"unicode/utf16"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type updateFunc func(tgbotapi.Update, *tgbotapi.BotAPI)

func processUpdate(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
	if update.InlineQuery != nil {
		var (
			results    []interface{}
			bashQuotes []BashQuote
			quote      BashQuote
			err        error
		)

		if update.InlineQuery.Query == "" {
			return
		}

		if quoteId, errConv := strconv.Atoi(update.InlineQuery.Query); errConv == nil {
			quote, err = GetQuote(quoteId)
			bashQuotes = append(bashQuotes, quote)
		} else {
			log.Print(errConv)
			bashQuotes, err = SearchQuotes(update.InlineQuery.Query, 3)
		}

		if err == nil {
			for _, quote := range bashQuotes {
				utfEncodedString := utf16.Encode([]rune(quote.Text))
				runeString := utf16.Decode(utfEncodedString[:50])

				title := fmt.Sprintf(
					"[#%d]: %s\n",
					quote.ID,
					string(runeString)+"...\n",
				)

				text := fmt.Sprintf(
					"*Цитата:* [#%d](%s), %s  \n"+
						"*Рейтинг:* %s  \n"+
						"%s    \n\n",
					quote.ID,
					quote.Permalink,
					quote.Created,
					quote.Rating,
					quote.Text,
				)

				results = append(results, tgbotapi.NewInlineQueryResultArticleMarkdown(
					strconv.Itoa(rand.Int()),
					title,
					text,
				))
			}
		} else {
			errMsg := "Не удалось произвести поиск"
			results = append(results, tgbotapi.NewInlineQueryResultArticleMarkdown(
				strconv.Itoa(rand.Int()),
				errMsg,
				errMsg,
			))
		}

		if len(results) == 0 {
			errMsg := "Ничего не найдено..."
			results = append(results, tgbotapi.NewInlineQueryResultArticleMarkdown(
				strconv.Itoa(rand.Int()),
				errMsg,
				errMsg,
			))
		}

		response, err := bot.AnswerInlineQuery(
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

		if err != nil {
			log.Print(err)
		}

		if !response.Ok {
			log.Printf("Error %d while trying to send inline update", response.ErrorCode)
		}
	} else {
		msgs := make([]tgbotapi.MessageConfig, 1)
		msgs[0] = NewMessage(update.Message.Chat.ID, 0, "", "markdown")

		if update.Message.IsCommand() {
			switch update.Message.Command() {
			case "latest":
				SendMessages(bot, []tgbotapi.MessageConfig{
					NewMessage(update.Message.Chat.ID, update.Message.MessageID, "_Получаю свежие цитаты..._", "markdown"),
				})

				items, err := GetLatestQuotes()

				if err != nil {
					msgs[len(msgs)-1].Text = "Не удалось получить последние цитаты :("
				} else {
					for _, item := range items {
						text := fmt.Sprintf(
							"*Цитата:* [#%d](%s), %s  \n"+
								"*Рейтинг:* %s  \n"+
								"%s    \n\n",
							item.ID,
							item.Permalink,
							item.Created,
							item.Rating,
							item.Text,
						)

						if len(msgs[len(msgs)-1].Text+text) > 4096 {
							msgs = append(msgs, NewMessage(update.Message.Chat.ID, 0, text, "markdown"))
						} else {
							msgs[len(msgs)-1].Text += text
						}
					}
				}
			default:
				msgs[len(msgs)-1].Text = "Как насчёт последних цитат? Используйте /latest"
			}
		} else {
			text := fmt.Sprintf("Зайдите в любой чат, вызовите бота вот так:\n `@%s <id>`, где ID - "+
				"это идентификатор цитаты на bash.im. И бот перешлёт её!\n"+
				"Ещё вместо идентификатора можно указать текст, по которому бот попытается найти цитаты.", bot.Self.UserName)
			msgs[len(msgs)-1] = NewMessage(update.Message.Chat.ID, update.Message.MessageID, text, "markdown")
		}

		SendMessages(bot, msgs)
	}
}

func NewMessage(chatID int64, replyTo int, text string, parse string) tgbotapi.MessageConfig {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = parse

	if replyTo != 0 {
		msg.ReplyToMessageID = replyTo
	}

	return msg
}

func SendMessages(bot *tgbotapi.BotAPI, msgs []tgbotapi.MessageConfig) {
	for _, msg := range msgs {
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Error while trying to send message to chat `%d`: %s", msg.BaseChat.ChatID, err.Error())
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
		go updateCallback(update, bot)
	}
}

func initWithWebhook(
	bot *tgbotapi.BotAPI,
	webhookUrl string,
	listenAddr string,
	certFile string,
	certKey string,
	updateCallback updateFunc,
) {
	var err error

	if webhookUrl == "" {
		log.Fatalf("Empty webhook URL provided (env %s)", envWebhook)
	}

	if certFile != "" && certKey != "" {
		_, err = bot.SetWebhook(tgbotapi.NewWebhookWithCert(webhookUrl, certFile))
	} else {
		_, err = bot.SetWebhook(tgbotapi.NewWebhook(webhookUrl))
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

	updates := bot.ListenForWebhook(webhookUrl[strings.LastIndex(webhookUrl, "/"):])

	if certFile != "" && certKey != "" {
		go http.ListenAndServeTLS(listenAddr, certFile, certKey, nil)
	} else {
		go http.ListenAndServe(listenAddr, nil)
	}

	for update := range updates {
		go updateCallback(update, bot)
	}
}
