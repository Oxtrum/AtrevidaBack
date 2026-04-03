package services

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
)

type WhatsAppPayload struct {
	MessagingProduct string `json:"messaging_product"`
	To               string `json:"to"`
	Type             string `json:"type"`
	Text             struct {
		Body string `json:"body"`
	} `json:"text"`
}

func SendWhatsApp(to string, message string) (*http.Response, error) {
	url := "https://graph.facebook.com/v18.0/TU_PHONE_ID/messages"

	payload := WhatsAppPayload{
		MessagingProduct: "whatsapp",
		To:               to,
		Type:             "text",
	}
	payload.Text.Body = message

	jsonData, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer "+os.Getenv("TOKEN"))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	return client.Do(req)
}
