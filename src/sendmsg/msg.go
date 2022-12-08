package sendmsg

import (
	"log"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/m1guelpf/chatgpt-telegram/src/chatgpt"
	"github.com/m1guelpf/chatgpt-telegram/src/markdown"
	"github.com/m1guelpf/chatgpt-telegram/src/ratelimit"
)

type Conversation struct {
	ConversationID string
	LastMessageID  string
}

func Sendmsg_(
	id int64,
	msg string,
	userConversations map[int64]Conversation,
	chatGPT *chatgpt.ChatGPT,
) {
	log.Print("Init bb", id, msg)
	feed, err := chatGPT.SendMessage(msg,
		userConversations[id].ConversationID,
		userConversations[id].LastMessageID,
	)
	if err != nil {
		log.Print(err)
		return
	}
	for response := range feed {
		// log.Print(response.Message)
		userConversations[id] = Conversation{
			LastMessageID:  response.MessageId,
			ConversationID: response.ConversationId,
		}
	}
}

func debouncedType(chatId int64, bot *tgbotapi.BotAPI) func() {
	return ratelimit.Debounce((1 * time.Second), func() {
		bot.Request(tgbotapi.NewChatAction(chatId, "typing"))
	})
}

func debouncedEdit(
	messageId int,
	msgConfig tgbotapi.MessageConfig,
	bot *tgbotapi.BotAPI,
) func(interface{}, interface{}) {
	return ratelimit.DebounceWithArgs((2 * time.Second),
		func(
			messageId interface{},
			text interface{},
		) {
			_, err := bot.Request(tgbotapi.EditMessageTextConfig{
				BaseEdit: tgbotapi.BaseEdit{
					ChatID:    msgConfig.ChatID,
					MessageID: messageId.(int),
				},
				Text:      text.(string),
				ParseMode: msgConfig.ParseMode,
			})

			if err != nil {
				if err.Error() == "Bad Request: message is not modified: specified new message content and reply markup are exactly the same as a current content and reply markup of the message" {
					return
				}

				log.Printf("Couldn't edit message: %v", err)
			}
		})
}

func ReplyToChat(
	feed chan chatgpt.ChatResponse,
	msgConfig tgbotapi.MessageConfig,
	userConversations map[int64]Conversation,
	bot *tgbotapi.BotAPI,
) (tgbotapi.MessageConfig, error) {
	var lastResp string
	var message tgbotapi.Message

	debounceE := debouncedEdit(message.MessageID, msgConfig, bot)

	for response := range feed {
		log.Print(response.Message)
		userConversations[msgConfig.ChatID] = Conversation{
			LastMessageID:  response.MessageId,
			ConversationID: response.ConversationId,
		}
		lastResp = markdown.EnsureFormatting(response.Message)
		msgConfig.Text = lastResp
		if msgConfig.Text == "``" {
			msgConfig.Text = "`...`"
		}

		if message.MessageID == 0 {
			var err error

			message, err = bot.Send(msgConfig)
			log.Print(msgConfig)
			log.Print("Send first")
			if err != nil {
				log.Fatalf("Couldn't send message: %v", err)
			}
		} else {
			log.Print("send edit")
			debounceE(message.MessageID, lastResp)
		}

	}

	_, err := bot.Request(tgbotapi.EditMessageTextConfig{
		BaseEdit: tgbotapi.BaseEdit{
			ChatID:    msgConfig.ChatID,
			MessageID: message.MessageID,
		},
		Text:      lastResp,
		ParseMode: "Markdown",
	})

	if err != nil {
		if err.Error() == "Bad Request: message is not modified: "+
			"specified new message content and reply markup are exactly the same as a current content and reply markup of the message" {
			return msgConfig, err
		}

		log.Printf("Couldn't perform final edit on message: %v", err)
	}
	return msgConfig, nil
}

func ProcessOneInput(
	input string,
	chatGPT *chatgpt.ChatGPT,
	msgConfig tgbotapi.MessageConfig,
	userConversations map[int64]Conversation,
	bot *tgbotapi.BotAPI,
) (tgbotapi.MessageConfig, error) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	debounceT := debouncedType(msgConfig.ChatID, bot)

	//

	go func() {
		for range ticker.C {
			debounceT()
		}
	}()

	feed, err := chatGPT.SendMessage(input,
		userConversations[msgConfig.ChatID].ConversationID,
		userConversations[msgConfig.ChatID].LastMessageID,
	)

	if err != nil {
		log.Print("Couldn't send message to chatgpt: ", err)
		msgConfig.Text = "Couldn't send message to chatgpt: " + err.Error()
		_, err := bot.Send(msgConfig)
		return msgConfig, err
	}

	return ReplyToChat(
		feed,
		msgConfig,
		userConversations,
		bot,
	)
}
