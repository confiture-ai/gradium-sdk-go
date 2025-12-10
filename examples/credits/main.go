// Example: Check Credits with Gradium SDK
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

	credits, err := client.Credits.Get(ctx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Plan: %s\n", credits.PlanName)
	fmt.Printf("Credits: %d / %d\n", credits.RemainingCredits, credits.AllocatedCredits)
	fmt.Printf("Billing Period: %s\n", credits.BillingPeriod)
	if credits.NextRolloverDate != nil {
		fmt.Printf("Next Rollover: %s\n", *credits.NextRolloverDate)
	}
}
