// Example: Streaming Speech-to-Text with Gradium SDK
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

	// Create streaming connection
	stream, err := client.STT.Stream(ctx, gradium.STTParams{
		InputFormat: gradium.InputFormatPCM,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = stream.Close() }()

	// Wait for ready
	info, err := stream.WaitReady(ctx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Stream ready (Sample rate: %d, Frame size: %d)\n", info.SampleRate, info.FrameSize)

	// Read and send audio in chunks
	audioData, err := os.ReadFile("audio.pcm")
	if err != nil {
		log.Fatal("Please provide an audio.pcm file (24kHz 16-bit mono):", err)
	}

	// Send audio in chunks
	chunkSize := info.FrameSize * 2 // 2 bytes per sample
	for i := 0; i < len(audioData); i += chunkSize {
		end := i + chunkSize
		if end > len(audioData) {
			end = len(audioData)
		}
		if err := stream.SendAudio(audioData[i:end]); err != nil {
			log.Fatal(err)
		}
	}

	if err := stream.SendEndOfStream(); err != nil {
		log.Fatal(err)
	}

	// Receive transcriptions
	for text := range stream.Text() {
		fmt.Printf("[%.2fs] %s\n", text.StartS, text.Text)
	}
}
