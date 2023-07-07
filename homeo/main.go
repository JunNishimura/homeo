package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/line/line-bot-sdk-go/v7/linebot"
	openai "github.com/sashabaranov/go-openai"
)

type Webhook struct {
	Events []*linebot.Event `json:"events"`
}

func isSignatureValid(channelSecret, signature string, body []byte) bool {
	decodedSignature, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return false
	}

	hash := hmac.New(sha256.New, []byte(channelSecret))
	_, err = hash.Write(body)
	if err != nil {
		return false
	}

	return hmac.Equal(decodedSignature, hash.Sum(nil))
}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	bot, err := linebot.New(
		os.Getenv("LINE_CHANNEL_SECRET"),
		os.Getenv("LINE_CHANNEL_TOKEN"),
	)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
		}, err
	}

	if isSignatureValid(os.Getenv("LINE_CHANNEL_SECRET"), request.Headers["X-Line-Signature"], []byte(request.Body)) {
		log.Println("signature is not valid")
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
		}, nil
	}

	var webhook Webhook
	if err := json.Unmarshal([]byte(request.Body), &webhook); err != nil {
		log.Println("json unmarshal error")
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
		}, err
	}

	client := openai.NewClient(os.Getenv("OPENAI_API_KEY"))

	for _, event := range webhook.Events {
		if event.Type == linebot.EventTypeMessage {
			switch message := event.Message.(type) {
			case *linebot.TextMessage:
				if request.Path == "/chat" {
					compReq := openai.ChatCompletionRequest{
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
						compReq,
					)
					if err != nil {
						return events.APIGatewayProxyResponse{
							StatusCode: http.StatusInternalServerError,
						}, err
					}

					if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(resp.Choices[0].Message.Content)).Do(); err != nil {
						return events.APIGatewayProxyResponse{
							StatusCode: http.StatusInternalServerError,
						}, err
					}
				}
			}
		}
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
	}, nil
}

func main() {
	lambda.Start(handler)
}
