package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/twilio/twilio-go"
	api "github.com/twilio/twilio-go/rest/verify/v2"
)

func main() {
	// Get credentials from environment variables
	accountSid := os.Getenv("TWILIO_ACCOUNT_SID")
	authToken := os.Getenv("TWILIO_AUTH_TOKEN")
	serviceSid := os.Getenv("TWILIO_VERIFY_SERVICE_SID")
	phoneNumber := os.Getenv("PHONE_NUMBER")
	region := os.Getenv("TWILIO_REGION")

	// Keep region empty for US (default), or use specific names for other regions
	// Don't default to "us1" as the SDK might interpret that as a subdomain

	// Validate required environment variables
	if accountSid == "" || authToken == "" || serviceSid == "" || phoneNumber == "" {
		log.Fatal("Please set TWILIO_ACCOUNT_SID, TWILIO_AUTH_TOKEN, TWILIO_VERIFY_SERVICE_SID, and PHONE_NUMBER environment variables")
	}

	// Initialize Twilio client
	client := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: accountSid,
		Password: authToken,
	})

	// IMPORTANT: Only set edge for non-US regions
	// For US (default), don't set any edge/region - it will use verify.twilio.com
	if region == "ie1" || region == "dublin" || region == "ireland" {
		fmt.Println("→ Setting edge to: dublin")
		client.SetEdge("dublin")
		client.SetRegion("ie1")
	} else if region == "au1" || region == "sydney" {
		fmt.Println("→ Setting edge to: sydney")
		client.SetEdge("sydney")
	} else {
		// For US or unknown, use default endpoint (no edge)
		fmt.Println("→ Using default US endpoint (no edge set)")
	}

	// Create SNA verification parameters
	params := &api.CreateVerificationParams{}
	params.SetTo(phoneNumber)
	params.SetChannel("sna")

	// Start new SNA verification
	fmt.Println("\n=== Starting new SNA verification ===")
	fmt.Printf("Phone: %s\n", phoneNumber)
	fmt.Printf("Region Config: %s\n", region)
	resp, err := client.VerifyV2.CreateVerification(serviceSid, params)
	if err != nil {
		log.Fatalf("Error creating verification: %v", err)
	}

	// Pretty print the response
	jsonResponse, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		log.Fatalf("Error marshaling response: %v", err)
	}

	fmt.Println("\n=== Verification Response ===")
	fmt.Println(string(jsonResponse))

	// Extract and display the SNA URL
	if resp.Sna != nil {
		// Type assert the Sna field to a map to access the URL
		if snaMap, ok := (*resp.Sna).(map[string]interface{}); ok {
			if url, exists := snaMap["url"]; exists {
				fmt.Println("\n=== SNA URL (use this in Step 2) ===")
				fmt.Println(url)
				fmt.Println("\nNote: This URL is unique and can only be used once. It expires in 10 minutes.")
			} else {
				fmt.Println("\nWarning: No URL found in SNA response")
			}
		} else {
			fmt.Println("\nWarning: Could not parse SNA response")
		}
	} else {
		fmt.Println("\nWarning: No SNA data found in response")
	}
}
