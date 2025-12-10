// Example: Text-to-Speech with Gradium SDK
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	gradium "github.com/confiture-ai/gradium-sdk-go"
)

func main() {
	// Create client (uses GRADIUM_API_KEY env var by default)
	client, err := gradium.NewClient()
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	// Simple TTS - get complete audio
	result, err := client.TTS.Create(ctx, gradium.TTSParams{
		VoiceID:      "YTpq7expH9539ERJ", // Emma voice
		OutputFormat: gradium.FormatWAV,
		Text:         "Hello! Welcome to Gradium. This is a text to speech example.",
	})
	if err != nil {
		log.Fatal(err)
	}

	// Save to file
	err = os.WriteFile("output.wav", result.RawData, 0o644)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Audio saved to output.wav (Request ID: %s)\n", result.RequestID)
}
