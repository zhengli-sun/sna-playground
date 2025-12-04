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

	// Create verification check parameters
	params := &api.CreateVerificationCheckParams{}
	params.SetTo(phoneNumber)

	// Check the SNA verification
	fmt.Println("\n=== Checking SNA verification ===")
	fmt.Printf("Phone number: %s\n", phoneNumber)
	fmt.Printf("Region Config: %s\n", region)

	resp, err := client.VerifyV2.CreateVerificationCheck(serviceSid, params)
	if err != nil {
		log.Fatalf("Error checking verification: %v", err)
	}

	// Pretty print the response
	jsonResponse, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		log.Fatalf("Error marshaling response: %v", err)
	}

	fmt.Println("=== Verification Check Response ===")
	fmt.Println(string(jsonResponse))

	// Display verification status
	fmt.Println("\n=== Verification Status ===")
	if resp.Status != nil {
		status := *resp.Status
		fmt.Printf("Status: %s\n", status)

		if status == "approved" {
			fmt.Println("✅ SUCCESS: SNA successfully confirmed user possession of the mobile number!")
		} else {
			fmt.Printf("⚠️  Verification status: %s\n", status)
		}
	}

	// Display valid status
	if resp.Valid != nil {
		fmt.Printf("Valid: %v\n", *resp.Valid)
	}

	// Check for SNA error codes
	fmt.Println("\n=== SNA Attempt Error Codes ===")
	if resp.SnaAttemptsErrorCodes != nil {
		errorSlice := *resp.SnaAttemptsErrorCodes

		if len(errorSlice) == 0 {
			fmt.Println("✅ No errors - SNA verification completed successfully!")
		} else {
			fmt.Printf("⚠️  Found %d error(s):\n", len(errorSlice))
			for i, errItem := range errorSlice {
				if errMap, ok := errItem.(map[string]interface{}); ok {
					fmt.Printf("\nError %d:\n", i+1)
					if attemptSid, exists := errMap["attempt_sid"]; exists {
						fmt.Printf("  Attempt SID: %v\n", attemptSid)
					}
					if code, exists := errMap["code"]; exists {
						fmt.Printf("  Error Code: %v\n", code)
						fmt.Println("  Check the Error and Warning Dictionary for more information:")
						fmt.Println("  https://www.twilio.com/docs/verify/api/verification#sna-error-codes")
					}
				}
			}
		}
	} else {
		fmt.Println("No SNA error codes in response")
	}

	// Important notes
	fmt.Println("\n=== Important Notes ===")
	fmt.Println("• Verifications expire after 10 minutes")
	fmt.Println("• If you get a 404 error, the verification has expired")
	fmt.Println("• Status 'approved' means successful verification")
	fmt.Println("• Check sna_attempts_error_codes for any issues")
}
