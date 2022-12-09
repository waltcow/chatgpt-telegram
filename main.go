package main

import (
	"fmt"
	"github.com/joho/godotenv"
	"github.com/m1guelpf/chatgpt-telegram/src/chatgpt"
	"github.com/m1guelpf/chatgpt-telegram/src/config"
	"github.com/m1guelpf/chatgpt-telegram/src/tgbot"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
)

func main() {
	conf, err := config.Init()
	if err != nil {
		log.Fatalf("Couldn't load config: %v", err)
	}

	chatGPT := chatgpt.Init(conf)
	log.Println("Started ChatGPT")

	err = godotenv.Load()
	if err != nil {
		log.Fatalf("Couldn't load .env file: %v", err)
	}

	bot, err := tgbot.New(os.Getenv("TELEGRAM_TOKEN"))

	if err != nil {
		log.Fatalf("Couldn't start Telegram bot: %v", err)
	}

	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		bot.Stop()
		os.Exit(0)
	}()

	log.Printf("Started Telegram bot! Message @%s to start.", bot.Username)

	for update := range bot.GetUpdatesChan() {
		if update.Message == nil {
			continue
		}

		var (
			updateText      = update.Message.Text
			updateChatID    = update.Message.Chat.ID
			updateMessageID = update.Message.MessageID
		)

		userId := strconv.FormatInt(update.Message.Chat.ID, 10)

		if os.Getenv("TELEGRAM_ID") != "" && userId != os.Getenv("TELEGRAM_ID") {
			bot.Send(updateChatID, updateMessageID, "You are not authorized to use this bot.")
			continue
		}

		isAtChatBotInGroup := (update.Message.Chat.Type == "group" || update.Message.Chat.Type == "supergroup") && strings.HasPrefix(update.Message.Text, "@"+bot.Username)

		if !update.Message.IsCommand() && (update.Message.Chat.IsPrivate() || isAtChatBotInGroup) {
			bot.SendTyping(updateChatID)

			feed, err := chatGPT.SendMessage(updateText, updateChatID)
			if err != nil {
				bot.Send(updateChatID, updateMessageID, fmt.Sprintf("Error: %v", err))
			} else {
				bot.SendAsLiveOutput(updateChatID, updateMessageID, feed)
			}
			continue
		}

		var text string
		switch update.Message.Command() {
		case "help":
			text = "Send a message to start talking with ChatGPT. You can use /reload at any point to clear the conversation history and start from scratch (don't worry, it won't delete the Telegram messages)."
		case "start":
			text = "Send a message to start talking with ChatGPT. You can use /reload at any point to clear the conversation history and start from scratch (don't worry, it won't delete the Telegram messages)."
		case "reload":
			chatGPT.ResetConversation(updateChatID)
			text = "Started a new conversation. Enjoy!"
		default:
			text = "Unknown command. Send /help to see a list of commands."
		}

		if _, err := bot.Send(updateChatID, updateMessageID, text); err != nil {
			log.Printf("Error sending message: %v", err)
		}
	}
}
