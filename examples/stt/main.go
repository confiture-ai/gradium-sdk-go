// Example: Speech-to-Text with Gradium SDK
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	gradium "github.com/confiture-ai/gradium-sdk-go"
)

func main() {
	client, err := gradium.NewClient() // uses GRADIUM_API_KEY env var
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	// Read audio file
	audioData, err := os.ReadFile("audio.wav")
	if err != nil {
		log.Fatal("Please provide an audio.wav file:", err)
	}

	// Transcribe
	text, err := client.STT.Transcribe(ctx, gradium.STTParams{
		InputFormat: gradium.InputFormatWAV,
	}, audioData)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Transcription: %s\n", text)
}
