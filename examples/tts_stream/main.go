// Example: Streaming Text-to-Speech with Gradium SDK
package main

import (
	"context"
	"fmt"
	"log"

	gradium "github.com/confiture-ai/gradium-sdk-go"
)

func main() {
	client, err := gradium.NewClient() // uses GRADIUM_API_KEY env var
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	// Create streaming connection
	stream, err := client.TTS.Stream(ctx, gradium.TTSParams{
		VoiceID:      "YTpq7expH9539ERJ",
		OutputFormat: gradium.FormatPCM,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = stream.Close() }()

	// Wait for ready
	if err := stream.WaitReady(ctx); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Stream ready (Request ID: %s)\n", stream.RequestID())

	// Send text
	if err := stream.SendText("Hello, this is a streaming example."); err != nil {
		log.Fatal(err)
	}
	if err := stream.SendText(" You can send multiple text chunks."); err != nil {
		log.Fatal(err)
	}
	if err := stream.SendEndOfStream(); err != nil {
		log.Fatal(err)
	}

	// Receive audio chunks
	var totalBytes int
	for chunk := range stream.Audio() {
		totalBytes += len(chunk)
		fmt.Printf("Received %d bytes (total: %d)\n", len(chunk), totalBytes)
	}

	fmt.Printf("Stream complete. Total audio: %d bytes\n", totalBytes)
}
