package main

import (
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	"github.com/m1guelpf/chatgpt-telegram/src/chatgpt"
	"github.com/m1guelpf/chatgpt-telegram/src/config"
	"github.com/m1guelpf/chatgpt-telegram/src/prompts"
	"github.com/m1guelpf/chatgpt-telegram/src/sendmsg"
	"github.com/m1guelpf/chatgpt-telegram/src/session"
)

func main() {
	config, err := config.Init()
	if err != nil {
		log.Fatalf("Couldn't load config: %v", err)
	}

	if config.OpenAISession == "" {
		session, err := session.GetSession()
		if err != nil {
			log.Fatalf("Couldn't get OpenAI session: %v", err)
		}

		err = config.Set("OpenAISession", session)
		if err != nil {
			log.Fatalf("Couldn't save OpenAI session: %v", err)
		}
	}

	chatGPT := chatgpt.Init(config)
	log.Println("Started ChatGPT")

	err = godotenv.Load()
	if err != nil {
		log.Fatalf("Couldn't load .env file: %v", err)
	}

	const proxyUrl = "http://172.31.176.1:10809"
	proxyUri, err := url.Parse(proxyUrl)
	if err != nil {
		log.Panic(err)
	}

	const APIEndpoint = "https://api.telegram.org/bot%s/%s"
	bot, err := tgbotapi.NewBotAPIWithClient(os.Getenv("TELEGRAM_TOKEN"),
		APIEndpoint,
		&http.Client{Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyUri),
		}})

	if err != nil {
		log.Panic(err)
	}

	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		bot.StopReceivingUpdates()
		os.Exit(0)
	}()

	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 30
	updates := bot.GetUpdatesChan(updateConfig)

	log.Printf("Started Telegram bot! Message @%s to start. %s, %s",
		bot.Self.UserName, bot.Self.FirstName, bot.Self.LastName)

	userConversations := make(map[int64]sendmsg.Conversation)

	for update := range updates {
		if update.Message == nil {
			continue
		}
		userInput := update.Message.Text
		if !update.Message.IsCommand() &&
			update.Message.Chat.IsGroup() {
			if !strings.HasPrefix(userInput,
				bot.Self.FirstName) {
				continue
			}
			userInput = userInput[2:]
		}

		log.Print(userInput)

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
		msg.ReplyToMessageID = update.Message.MessageID
		msg.ParseMode = "Markdown"

		userId := strconv.FormatInt(update.Message.Chat.ID, 10)
		if os.Getenv("TELEGRAM_ID") != "" && userId != os.Getenv("TELEGRAM_ID") {
			msg.Text = "You are not authorized to use this bot."
			bot.Send(msg)
			continue
		}

		if !update.Message.IsCommand() {
			msg, err = sendmsg.ProcessOneInput(
				userInput,
				&chatGPT,
				msg,
				userConversations,
				bot,
			)
			if err != nil {
				log.Print(err)
			}
			continue
		}
		args := update.Message.CommandArguments()
		log.Print(args)
		log.Print(update.Message.Command())

		switch update.Message.Command() {
		case "help":
			msg.Text = "Send a message to start talking with ChatGPT. You can use /reload at any point to clear the conversation history and start from scratch (don't worry, it won't delete the Telegram messages)."
		case "start":
			msg.Text = "Send a message to start talking with ChatGPT. You can use /reload at any point to clear the conversation history and start from scratch (don't worry, it won't delete the Telegram messages)."
		case "reload":
			userConversations[update.Message.Chat.ID] = sendmsg.Conversation{}
			msg.Text = "Started a new conversation. Enjoy!"
		case "travel":
			_, err := sendmsg.ProcessOneInput(
				prompts.TravelGuide,
				&chatGPT,
				msg,
				userConversations,
				bot,
			)
			if err != nil {
				log.Print(err)
			}
			continue
		case "terminal":
			_, err := sendmsg.ProcessOneInput(
				prompts.LinuxTerminal(args),
				&chatGPT,
				msg,
				userConversations,
				bot,
			)
			if err != nil {
				log.Print(err)
			}
			continue
		case "xjp":
			_, err := sendmsg.ProcessOneInput(
				prompts.XiJinPing,
				&chatGPT,
				msg,
				userConversations,
				bot,
			)
			if err != nil {
				log.Print(err)
			}
			continue
		case "jzm":
			_, err := sendmsg.ProcessOneInput(
				prompts.JiangZeMing,
				&chatGPT,
				msg,
				userConversations,
				bot,
			)
			if err != nil {
				log.Print(err)
			}
			continue
		case "catgirl":
			_, err := sendmsg.ProcessOneInput(
				prompts.CatGirl,
				&chatGPT,
				msg,
				userConversations,
				bot,
			)
			if err != nil {
				log.Print(err)
			}
			continue
		case "act":
			log.Print("act")
			_, err := sendmsg.ProcessOneInput(
				prompts.Charactor(args),
				&chatGPT,
				msg,
				userConversations,
				bot,
			)
			if err != nil {
				log.Print(err)
			}
			continue
		case "animal":
			log.Print("animal")
			_, err := sendmsg.ProcessOneInput(
				prompts.Animal(args),
				&chatGPT,
				msg,
				userConversations,
				bot,
			)
			if err != nil {
				log.Print(err)
			}
			continue
		case "turing":
			log.Print("turing")
			_, err := sendmsg.ProcessOneInput(
				prompts.TuringTest,
				&chatGPT,
				msg,
				userConversations,
				bot,
			)
			if err != nil {
				log.Print(err)
			}
			continue
		case "doctor":
			_, err := sendmsg.ProcessOneInput(
				prompts.Doctor,
				&chatGPT,
				msg,
				userConversations,
				bot,
			)
			if err != nil {
				log.Print(err)
			}
			continue

		case "rap":
			_, err := sendmsg.ProcessOneInput(
				prompts.Rapper,
				&chatGPT,
				msg,
				userConversations,
				bot,
			)
			if err != nil {
				log.Print(err)
			}
			continue
		case "baba":
			_, err := sendmsg.ProcessOneInput(
				prompts.Baba,
				&chatGPT,
				msg,
				userConversations,
				bot,
			)
			if err != nil {
				log.Print(err)
			}
			continue
		case "role":
			_, err := sendmsg.ProcessOneInput(
				prompts.Role(args),
				&chatGPT,
				msg,
				userConversations,
				bot,
			)
			if err != nil {
				log.Print(err)
			}
			continue
		case "pokemon":
			_, err := sendmsg.ProcessOneInput(
				prompts.Pokemon,
				&chatGPT,
				msg,
				userConversations,
				bot,
			)
			if err != nil {
				log.Print(err)
			}
			continue

		default:
			log.Print("bb Done")
			continue
		}

		if _, err := bot.Send(msg); err != nil {
			log.Printf("Couldn't send message: %v", err)
			continue
		}
	}
}
