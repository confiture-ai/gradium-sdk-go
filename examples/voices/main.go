// Example: Voice Management with Gradium SDK
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

	// List voices
	voices, err := client.Voices.List(ctx, &gradium.VoiceListParams{
		Limit:          10,
		IncludeCatalog: true,
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d voices:\n", len(voices))
	for _, v := range voices {
		lang := "unknown"
		if v.Language != nil {
			lang = *v.Language
		}
		fmt.Printf("  - %s (%s) [%s]\n", v.Name, v.UID, lang)
	}

	// Create custom voice (if audio file provided)
	if len(os.Args) > 1 {
		audioFile, err := os.Open(os.Args[1])
		if err != nil {
			log.Fatal(err)
		}
		defer func() { _ = audioFile.Close() }()

		desc := "My custom voice"
		lang := "en"

		result, err := client.Voices.Create(ctx, audioFile, "voice_sample.wav", gradium.VoiceCreateParams{
			Name:        "My Custom Voice",
			Description: &desc,
			Language:    &lang,
		})
		if err != nil {
			log.Fatal(err)
		}

		if result.Error != nil {
			log.Fatal(*result.Error)
		}

		fmt.Printf("Created voice: %s\n", *result.UID)
	}
}
