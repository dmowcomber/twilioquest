package main

import (
	"fmt"
	"os"
	"time"

	"github.com/sfreiberg/gotwilio"
)

const (
	envKeyAccountSID = "TWILIO_ACCOUNT_SID"
	envKeyAuthToken  = "TWILIO_AUTH_TOKEN"
)

func main() {
	// initialize client
	accountSID := os.Getenv(envKeyAccountSID)
	authToken := os.Getenv(envKeyAuthToken)
	if accountSID == "" || authToken == "" {
		fmt.Printf("must set environment variables %q and %q\n", envKeyAccountSID, envKeyAuthToken)
		os.Exit(1)
	}
	twilioCli := gotwilio.NewTwilioClient(accountSID, authToken)

	// send message
	from := "+16572206115"
	to := "+12092104311"
	message := fmt.Sprintf("Greetings! The current time is: %s 831S4WQY5VYJHIR", time.Now().Format("2006-01-02 15:04:05"))
	resp, _, err := twilioCli.SendSMS(from, to, message, "", accountSID)
	if err != nil {
		fmt.Printf("failed to send sms: %q", err.Error())
		os.Exit(1)
	}
	fmt.Printf("successfully sent sms: %#v\n", resp)
}
