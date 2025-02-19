package main

import (
	"context"
	"github.com/mrz1836/postmark"
	"log"
	"os"
)

var emailClient *postmark.Client

func emailSetup() {
	emailClient = postmark.NewClient(os.Getenv("POSTMARK_SERVER_TOKEN"), os.Getenv("POSTMARK_ACCOUNT_TOKEN"))
	email := postmark.Email{
		From:     "0428079@radford.act.edu.au",
		To:       "0358632@radford.act.edu.au",
		Subject:  "OpenElect is Up",
		TextBody: "OpenElect is up and running!",
	}
	_, err := emailClient.SendEmail(context.Background(), email)
	if err != nil {
		log.Fatalf("Error sending email: %v", err)
	}
}

func sendEmail(from string, to string, subject string, body string) error {
	email := postmark.Email{
		From:     from,
		To:       to,
		Subject:  subject,
		TextBody: body + "\n\nThis is an automated message sent from OpenElect.",
	}
	_, err := emailClient.SendEmail(context.Background(), email)
	return err
}
