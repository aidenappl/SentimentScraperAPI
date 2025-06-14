package gpt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/aidenappl/SentimentScraperAPI/env"
	"github.com/aidenappl/SentimentScraperAPI/structs"
)

type OpenAIRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func FetchSentimentFromChatGPT(article structs.News) (*structs.Sentiment, error) {

	prompt := buildPrompt(article)

	requestBody := OpenAIRequest{
		Model: "gpt-4", // or "gpt-3.5-turbo"
		Messages: []ChatMessage{
			{Role: "system", Content: "You are a JSON-based sentiment analyzer."},
			{Role: "user", Content: prompt},
		},
	}

	payload, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+env.OpenAIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)

	var raw struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		log.Println("Failed to parse OpenAI response, retrying!:", err)
		return FetchSentimentFromChatGPT(article) // Retry in case of transient error
	}

	if len(raw.Choices) == 0 {
		return nil, fmt.Errorf("no choices returned from OpenAI")
	}

	// Unmarshal the response content into Sentiment struct
	var sentiment structs.Sentiment
	if err := json.Unmarshal([]byte(raw.Choices[0].Message.Content), &sentiment); err != nil {
		log.Println("Failed to parse sentiment response, retrying!:", err)
		return FetchSentimentFromChatGPT(article) // Retry in case of parsing error
	}

	return &sentiment, nil
}
