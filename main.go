package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/m1guelpf/chatgpt-telegram/src/chatgpt"
	"github.com/m1guelpf/chatgpt-telegram/src/config"
	"github.com/m1guelpf/chatgpt-telegram/src/tgbot"
)

func main() {
	envConfig, err := config.LoadEnvConfig(".env")
	if err != nil {
		log.Fatalf("Couldn't load .env config: %v", err)
	}

	chatGPT := chatgpt.Init(envConfig.OpenAISession)
	log.Println("Started ChatGPT")

	if err := envConfig.ValidateWithDefaults(); err != nil {
		log.Fatalf("Invalid .env config: %v", err)
	}

	bot, err := tgbot.New(envConfig.TelegramToken, time.Duration(envConfig.EditWaitSeconds))
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
			updateText       = update.Message.Text
			updateChatID     = update.Message.Chat.ID
			updateMessageID  = update.Message.MessageID
			updateUserID     = update.Message.From.ID
			isChatBotInGroup = update.Message.Chat.Type == "group" || update.Message.Chat.Type == "supergroup"
			isMentionChatBot = strings.HasPrefix(update.Message.Text, "@"+bot.Username) || strings.HasSuffix(update.Message.Text, "@"+bot.Username)
			isPrivateChat    = update.Message.Chat.IsPrivate()
		)

		if len(envConfig.TelegramID) != 0 && !envConfig.HasTelegramID(updateUserID) {
			log.Printf("User %d is not allowed to use this bot", updateUserID)
			bot.Send(updateChatID, updateMessageID, "You are not authorized to use this bot.")
			continue
		}

		if !update.Message.IsCommand() && (isChatBotInGroup && isMentionChatBot || isPrivateChat) {
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
		case "renew":
			sessionToken := update.Message.CommandArguments()
			resp, err := chatGPT.UpdateSessionToken(sessionToken)
			if err != nil {
				text = fmt.Sprintf("Error renewing auth: %v", err)
			} else {
				text = "Renewed auth successfully! " + resp
				_, err = config.UpdateEnvConfig(".env", "OPENAI_SESSION", sessionToken)
				if err != nil {
					log.Printf("Error updating .env file: %v", err)
					text += " (but couldn't update .env file)"
				}
				text += " (updated .env file)"
			}
		default:
			continue
		}

		if _, err := bot.Send(updateChatID, updateMessageID, text); err != nil {
			log.Printf("Error sending message: %v", err)
		}
	}
}
