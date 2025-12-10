# Gradium Go SDK

Unofficial Go SDK for the [Gradium API](https://gradium.ai) — low-latency, high-quality Text-to-Speech and Speech-to-Text services.

## Features

- **Text-to-Speech (TTS)** — Convert text to natural-sounding speech
- **Speech-to-Text (STT)** — Transcribe audio with real-time streaming
- **Voice Cloning** — Create custom voices from audio samples
- **Multilingual** — Support for English, French, German, Spanish, and Portuguese
- **Low Latency** — Sub-300ms time-to-first-token
- **Streaming** — Real-time audio streaming via WebSockets

## Installation

```bash
go get github.com/confiture-ai/gradium-sdk-go
```

## Quick Start

```go
package main

import (
    "context"
    "os"

    gradium "github.com/confiture-ai/gradium-sdk-go"
)

func main() {
    // Create client (uses GRADIUM_API_KEY env var by default)
    client, _ := gradium.NewClient()

    // Text-to-Speech
    result, _ := client.TTS.Create(context.Background(), gradium.TTSParams{
        VoiceID:      "YTpq7expH9539ERJ",
        OutputFormat: gradium.FormatWAV,
        Text:         "Hello, world!",
    })

    os.WriteFile("output.wav", result.RawData, 0644)
}
```

## Configuration

### Creating a Client

```go
// Using environment variable (recommended)
client, err := gradium.NewClient() // reads GRADIUM_API_KEY

// Using API key directly
client, err := gradium.NewClient(
    gradium.WithAPIKey("gd_your_api_key"),
)

// With multiple options
client, err := gradium.NewClient(
    gradium.WithAPIKey("gd_your_api_key"), // or omit to use env var
    gradium.WithRegion(gradium.RegionUS),  // Use US region
    gradium.WithTimeout(60 * time.Second), // Custom timeout
)
```

### Environment Variables

```bash
# bash/zsh
export GRADIUM_API_KEY=gd_your_api_key_here

# fish
set -x GRADIUM_API_KEY gd_your_api_key_here
```

### Regions

```go
gradium.RegionEU  // Europe (default)
gradium.RegionUS  // United States
```

## Text-to-Speech (TTS)

### Simple TTS

```go
result, err := client.TTS.Create(ctx, gradium.TTSParams{
    VoiceID:      "YTpq7expH9539ERJ",
    OutputFormat: gradium.FormatWAV,
    Text:         "Hello, world!",
})

os.WriteFile("output.wav", result.RawData, 0644)
```

### Streaming TTS

```go
stream, err := client.TTS.Stream(ctx, gradium.TTSParams{
    VoiceID:      "YTpq7expH9539ERJ",
    OutputFormat: gradium.FormatPCM,
})
defer stream.Close()

stream.WaitReady(ctx)
stream.SendText("Hello, ")
stream.SendText("this is streaming!")
stream.SendEndOfStream()

// Process audio chunks as they arrive
for chunk := range stream.Audio() {
    // Play or save chunk
}
```

### Speed Control

```go
result, err := client.TTS.Create(ctx, gradium.TTSParams{
    VoiceID:      "YTpq7expH9539ERJ",
    OutputFormat: gradium.FormatWAV,
    Text:         "Hello, world!",
    JSONConfig: &gradium.TTSConfig{
        PaddingBonus: -2.0, // Faster (-4.0 to -0.1)
        // PaddingBonus: 2.0, // Slower (0.1 to 4.0)
    },
})
```

### Adding Breaks/Pauses

```go
result, err := client.TTS.Create(ctx, gradium.TTSParams{
    VoiceID:      "YTpq7expH9539ERJ",
    OutputFormat: gradium.FormatWAV,
    Text:         `First sentence. <break time="1.5s" /> Second sentence after a pause.`,
})
```

### Output Formats

| Format | Description |
|--------|-------------|
| `FormatWAV` | Standard WAV file |
| `FormatPCM` | Raw PCM (48kHz, 16-bit, mono) |
| `FormatOpus` | Opus codec |
| `FormatULaw8000` | μ-law 8kHz |
| `FormatALaw8000` | A-law 8kHz |
| `FormatPCM16000` | PCM 16kHz |
| `FormatPCM24000` | PCM 24kHz |

## Speech-to-Text (STT)

### Simple Transcription

```go
audioData, _ := os.ReadFile("audio.wav")

text, err := client.STT.Transcribe(ctx, gradium.STTParams{
    InputFormat: gradium.InputFormatWAV,
}, audioData)

fmt.Println(text)
```

### Streaming STT

```go
stream, err := client.STT.Stream(ctx, gradium.STTParams{
    InputFormat: gradium.InputFormatPCM,
})
defer stream.Close()

info, _ := stream.WaitReady(ctx)
fmt.Printf("Sample rate: %d\n", info.SampleRate)

// Send audio chunks
stream.SendAudio(chunk1)
stream.SendAudio(chunk2)
stream.SendEndOfStream()

// Receive transcriptions
for text := range stream.Text() {
    fmt.Printf("[%.2fs] %s\n", text.StartS, text.Text)
}
```

### Voice Activity Detection (VAD)

```go
stream, err := client.STT.Stream(ctx, gradium.STTParams{
    InputFormat: gradium.InputFormatPCM,
})
defer stream.Close()

stream.WaitReady(ctx)

// Monitor VAD for turn-taking
for vad := range stream.VAD() {
    prob := vad.VAD[2].InactivityProb
    if prob > 0.8 {
        fmt.Println("Speaker likely finished")
    }
}
```

### Audio Format Requirements (PCM)

- **Sample Rate**: 24000 Hz (24kHz)
- **Bit Depth**: 16-bit signed integer (little-endian)
- **Channels**: Mono
- **Chunk Size**: 1920 samples (80ms) recommended

### Input Formats

| Format | Description |
|--------|-------------|
| `InputFormatPCM` | Raw PCM (24kHz 16-bit mono) |
| `InputFormatWAV` | WAV format |
| `InputFormatOpus` | Opus format |

## Voices

### List Voices

```go
voices, err := client.Voices.List(ctx, &gradium.VoiceListParams{
    Limit:          100,
    IncludeCatalog: true,
})

for _, v := range voices {
    fmt.Printf("%s (%s)\n", v.Name, v.UID)
}
```

### Get Voice

```go
voice, err := client.Voices.Get(ctx, "voice_uid")
```

### Create Custom Voice

```go
audioFile, _ := os.Open("voice_sample.wav")
defer audioFile.Close()

result, err := client.Voices.Create(ctx, audioFile, "voice_sample.wav", gradium.VoiceCreateParams{
    Name:        "My Voice",
    Description: ptr("Custom voice description"),
    Language:    ptr("en"),
})
```

### Update Voice

```go
voice, err := client.Voices.Update(ctx, "voice_uid", gradium.VoiceUpdateParams{
    Name: ptr("New Name"),
})
```

### Delete Voice

```go
err := client.Voices.Delete(ctx, "voice_uid")
```

## Credits

```go
credits, err := client.Credits.Get(ctx)

fmt.Printf("Remaining: %d / %d\n", credits.RemainingCredits, credits.AllocatedCredits)
fmt.Printf("Plan: %s\n", credits.PlanName)
fmt.Printf("Next rollover: %s\n", credits.NextRolloverDate)
```

## Available Voices

### Flagship Voices

| Name | Voice ID | Language | Gender | Description |
|------|----------|----------|--------|-------------|
| Emma | `YTpq7expH9539ERJ` | en-US | Feminine | Pleasant and smooth, ready to assist |
| Kent | `LFZvm12tW_z0xfGo` | en-US | Masculine | Relaxed and authentic American |
| Sydney | `jtEKaLYNn6iif5PR` | en-US | Feminine | Joyful and airy |
| John | `KWJiFWu2O9nMPYcR` | en-US | Masculine | Warm, low-pitched broadcaster |
| Eva | `ubuXFxVQwVYnZQhy` | en-GB | Feminine | Joyful and dynamic British |
| Jack | `m86j6D7UZpGzHsNu` | en-GB | Masculine | Pleasant British |
| Elise | `b35yykvVppLXyw_l` | fr-FR | Feminine | Warm French |
| Leo | `axlOaUiFyOZhy4nv` | fr-FR | Masculine | Warm French |
| Mia | `-uP9MuGtBqAvEyxI` | de-DE | Feminine | Joyful German |
| Maximilian | `0y1VZjPabOBU3rWy` | de-DE | Masculine | Warm German |
| Valentina | `B36pbz5_UoWn4BDl` | es-MX | Feminine | Warm Mexican |
| Sergio | `xu7iJ_fn2ElcWp2s` | es-ES | Masculine | Warm Spanish |
| Alice | `pYcGZz9VOo4n2ynh` | pt-BR | Feminine | Warm Brazilian |
| Davi | `M-FvVo9c-jGR4PgP` | pt-BR | Masculine | Engaging Brazilian |

> You can also create your own [custom voices](#create-custom-voice).

## Error Handling

```go
result, err := client.TTS.Create(ctx, params)
if err != nil {
    switch e := err.(type) {
    case *gradium.AuthenticationError:
        // Invalid API key
    case *gradium.ValidationError:
        // Invalid parameters
    case *gradium.RateLimitError:
        // Rate limit exceeded, retry after e.RetryAfter seconds
    case *gradium.NotFoundError:
        // Resource not found
    case *gradium.WebSocketError:
        // WebSocket connection error
    default:
        // Other error
    }
}
```

## Testing

Run the test suite:

```bash
go test -v ./...

# With race detection
go test -race ./...

# With coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## License

MIT

---

Built with love in Paris by [Majdi Toumi](https://majdi.im)
