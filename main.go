package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/line/line-bot-sdk-go/v7/linebot"
	openai "github.com/sashabaranov/go-openai"
)

func main() {
	if err := godotenv.Load(".env"); err != nil {
		log.Printf("fail to load env file: %v", err)
	}

	bot, err := linebot.New(
		os.Getenv("LINE_CHANNEL_SECRET"),
		os.Getenv("LINE_CHANNEL_TOKEN"),
	)
	if err != nil {
		log.Printf("fail to init line bot: %v", err)
	}

	client := openai.NewClient(os.Getenv("OPENAI_API_KEY"))

	http.HandleFunc("/callback", func(w http.ResponseWriter, req *http.Request) {
		events, err := bot.ParseRequest(req)
		if err != nil {
			if err == linebot.ErrInvalidSignature {
				w.WriteHeader(400)
			} else {
				w.WriteHeader(500)
			}
			return
		}
		for _, event := range events {
			if event.Type == linebot.EventTypeMessage {
				switch message := event.Message.(type) {
				case *linebot.TextMessage:
					req := openai.ChatCompletionRequest{
						Model: openai.GPT3Dot5Turbo,
						Messages: []openai.ChatCompletionMessage{
							{
								Role: openai.ChatMessageRoleSystem,
								Content: `以下はAIチャットボット(以下ボット)との会話です。
ボットは、150文字以下の文章で相手の事を褒めます。

ボットは
1. 相手の発話内容を繰り返す
2. 相手の行動は凡人にはできないことだと述べる。たとえ、「食べる」「飲む」といった人間が当たり前にできることであったとしても、わざと「普通の人には到底できない」ことだと言う
3. 相手の存在が特別であることを述べる。「100年に一度の逸材」という言葉は、よく使われる言葉で面白くないので、人がくすっと笑えるようなユーモアがあり、何かの比喩を含んだ表現に言い換える
4. 相手の行動が偉人の功績と同等であることを述べる。参照する偉人は任意で良いから、とにかく具体例を出す
という構成で相手の事を褒めます。`,
							},
							{
								Role:    openai.ChatMessageRoleUser,
								Content: message.Text,
							},
						},
					}

					resp, err := client.CreateChatCompletion(
						context.Background(),
						req,
					)
					if err != nil {
						log.Printf("fail to get a response from OpenAI API: %v", err)
					}

					if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(resp.Choices[0].Message.Content)).Do(); err != nil {
						log.Printf("fail to reply message from bot: %v", err)
					}
				}
			}
		}
	})

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
