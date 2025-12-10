// Example: Basic usage of the Gradium SDK
//
// This example demonstrates all main features of the SDK in one file.
// Run with: go run examples/basic/main.go
package main

import (
	"context"
	"fmt"
	"os"

	gradium "github.com/confiture-ai/gradium-sdk-go"
)

func main() {
	fmt.Println("=== Gradium SDK Examples ===")
	fmt.Println()

	// Create client (reads GRADIUM_API_KEY from environment by default)
	client, err := gradium.NewClient()
	if err != nil {
		fmt.Printf("Failed to create client: %v\n", err)
		fmt.Println("Make sure GRADIUM_API_KEY is set")
		os.Exit(1)
	}

	ctx := context.Background()

	// Example 1: Check credits
	fmt.Println("1. Checking credits...")
	credits, err := client.Credits.Get(ctx)
	if err != nil {
		fmt.Printf("   (Skipped - %v)\n", err)
	} else {
		fmt.Printf("   Credits: %d/%d\n", credits.RemainingCredits, credits.AllocatedCredits)
		fmt.Printf("   Plan: %s\n", credits.PlanName)
	}

	// Example 2: List voices
	fmt.Println("\n2. Listing voices...")
	voices, err := client.Voices.List(ctx, &gradium.VoiceListParams{
		IncludeCatalog: true,
		Limit:          5,
	})
	if err != nil {
		fmt.Printf("   (Skipped - %v)\n", err)
	} else {
		fmt.Printf("   Found %d voices\n", len(voices))
		for i, v := range voices {
			if i >= 3 {
				break
			}
			fmt.Printf("   - %s (%s)\n", v.Name, v.UID)
		}
	}

	// Example 3: Text-to-Speech
	fmt.Println("\n3. Text-to-Speech...")
	result, err := client.TTS.Create(ctx, gradium.TTSParams{
		VoiceID:      "YTpq7expH9539ERJ", // Emma voice
		OutputFormat: gradium.FormatWAV,
		Text:         "Hello! Welcome to Gradium. This is a test of the text to speech system.",
	})
	if err != nil {
		fmt.Printf("   (Skipped - %v)\n", err)
	} else {
		fmt.Printf("   Generated %d bytes of audio\n", len(result.RawData))
		fmt.Printf("   Sample rate: %dHz\n", result.SampleRate)
		fmt.Printf("   Request ID: %s\n", result.RequestID)

		// Save to file
		err = os.WriteFile("output.wav", result.RawData, 0o644)
		if err != nil {
			fmt.Printf("   Failed to save: %v\n", err)
		} else {
			fmt.Println("   Saved to output.wav")
		}
	}

	// Example 4: Streaming TTS
	fmt.Println("\n4. Streaming TTS...")
	stream, err := client.TTS.Stream(ctx, gradium.TTSParams{
		VoiceID:      "YTpq7expH9539ERJ",
		OutputFormat: gradium.FormatPCM,
	})
	if err != nil {
		fmt.Printf("   (Skipped - %v)\n", err)
	} else {
		defer func() { _ = stream.Close() }()

		if err := stream.WaitReady(ctx); err != nil {
			fmt.Printf("   (Skipped - %v)\n", err)
		} else {
			fmt.Printf("   Stream ready, request ID: %s\n", stream.RequestID())

			_ = stream.SendText("This is a streaming example. The audio is generated in real-time.")
			_ = stream.SendEndOfStream()

			var totalBytes int
			for chunk := range stream.Audio() {
				totalBytes += len(chunk)
			}
			fmt.Printf("   Received %d bytes of streaming audio\n", totalBytes)
		}
	}

	fmt.Println("\n=== Examples Complete ===")
}
