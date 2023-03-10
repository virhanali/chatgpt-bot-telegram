package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	openai "github.com/sashabaranov/go-openai"
	"github.com/spf13/viper"
)

type Config struct {
	TelegramToken string `mapstructure:"tgToken"`
	GptToken      string `mapstructure:"gptToken"`
}

func LoadConfig(path string) (c Config, err error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(path)

	viper.AutomaticEnv()

	err = viper.ReadInConfig()

	if err != nil {
		return
	}

	err = viper.Unmarshal(&c)
	return
}

func sendChatGPT(sendText string) string {
	c := openai.NewClient(viper.GetString("gptToken"))

	resp, err := c.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: sendText,
				},
			},
		},
	)

	if err != nil {
		log.Panic(err)
	}

	return resp.Choices[0].Message.Content
}

func main() {
	config, err := LoadConfig(".")

	if err != nil {
		panic(fmt.Errorf("fatal error with config.yaml: %w", err))
	}

	bot, err := tgbotapi.NewBotAPI(config.TelegramToken)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true // set to false for suppress logs in stdout
	log.Printf("Authorized on account %s", bot.Self.UserName)

	// Start Telegram long polling update
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates, err := bot.GetUpdatesChan(u)

	//Check message in updates
	for update := range updates {
		if update.Message == nil {
			continue
		}

		if !strings.HasPrefix(update.Message.Text, "/c ") {
			continue
		}

		cutText, _ := strings.CutPrefix(update.Message.Text, "/c")
		update.Message.Text = sendChatGPT(cutText)

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text)
		msg.ReplyToMessageID = update.Message.MessageID

		_, err = bot.Send(msg)
		if err != nil {
			log.Println("Error:", err)
		}
	}
}
