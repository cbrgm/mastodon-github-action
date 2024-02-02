package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/alexflint/go-arg"
)

// Application metadata
var (
	Version   string
	Revision  string
	GoVersion = runtime.Version()
	StartTime = time.Now()
)

// VisibilityType defines Mastodon status visibility options
type VisibilityType string

const (
	VisibilityPublic   VisibilityType = "public"
	VisibilityUnlisted VisibilityType = "unlisted"
	VisibilityPrivate  VisibilityType = "private"
	VisibilityDirect   VisibilityType = "direct"
)

func (vt VisibilityType) IsValid() bool {
	return vt == VisibilityPublic || vt == VisibilityUnlisted || vt == VisibilityPrivate || vt == VisibilityDirect
}

type MastodonStatus struct {
	Status      string `json:"status"`
	Visibility  string `json:"visibility"`
	Sensitive   bool   `json:"sensitive,omitempty"`
	SpoilerText string `json:"spoiler_text,omitempty"`
	Language    string `json:"language,omitempty"`
	ScheduledAt string `json:"scheduled_at,omitempty"`
}

type ActionInputs struct {
	URL         string `arg:"--url,required"`
	AccessToken string `arg:"--access-token,required"`
	Message     string `arg:"--message,required"`
	Visibility  string `arg:"--visibility"`
	Sensitive   bool   `arg:"--sensitive"`
	SpoilerText string `arg:"--spoiler-text"`
	Language    string `arg:"--language"`
	ScheduledAt string `arg:"--scheduled-at"`
}

type StatusResponse struct {
	ID         string `json:"id"`
	URL        string `json:"url"`
	Content    string `json:"content"`
	CreatedAt  string `json:"created_at"`
	Visibility string `json:"visibility"`
}

type ScheduledStatusResponse struct {
	ID          string    `json:"id"`
	ScheduledAt time.Time `json:"scheduled_at"`
}

func main() {
	var args ActionInputs
	arg.MustParse(&args)

	log.Printf("Mastodon GitHub Action Version: %s, Build: %s", Version, Revision)

	if strings.TrimSpace(args.Message) == "" {
		log.Fatal("Status message cannot be empty")
	}

	if !VisibilityType(args.Visibility).IsValid() {
		log.Fatalf("Invalid visibility: %s", args.Visibility)
	}

	scheduledAt, err := parseScheduledAt(args.ScheduledAt)
	if err != nil {
		log.Fatalf("Scheduled at error: %v", err)
	}

	status := MastodonStatus{
		Status:      args.Message,
		Visibility:  args.Visibility,
		Sensitive:   args.Sensitive,
		SpoilerText: args.SpoilerText,
		Language:    args.Language,
		ScheduledAt: scheduledAt,
	}

	if err := postStatus(args.URL, args.AccessToken, status); err != nil {
		log.Fatalf("Error posting status: %v", err)
	}
}

func postStatus(url, accessToken string, status MastodonStatus) error {
	payload, err := json.Marshal(status)
	if err != nil {
		return fmt.Errorf("marshaling error: %w", err)
	}

	apiURL := fmt.Sprintf("%s/api/v1/statuses", url)
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("request creation error: %w", err)
	}

	req.Header.Add("Authorization", "Bearer "+accessToken)
	req.Header.Add("Content-Type", "application/json")

	client := http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("API call error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body) // Reading body for debug
		return fmt.Errorf("API response: %s, Body: %s", resp.Status, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response body error: %w", err)
	}

	// Determine response type (immediate or scheduled) and output accordingly
	if strings.Contains(status.ScheduledAt, "T") {
		var scheduledResponse ScheduledStatusResponse
		if err := json.Unmarshal(body, &scheduledResponse); err != nil {
			return fmt.Errorf("unmarshaling scheduled response error: %w", err)
		}
		setActionOutputs(map[string]string{
			"id":           scheduledResponse.ID,
			"scheduled_at": scheduledResponse.ScheduledAt.String(),
		})
		log.Printf("Scheduled Status ID: %s, Scheduled At: %s", scheduledResponse.ID, scheduledResponse.ScheduledAt)
	} else {
		var statusResponse StatusResponse
		if err := json.Unmarshal(body, &statusResponse); err != nil {
			return fmt.Errorf("unmarshaling status response error: %w", err)
		}
		setActionOutputs(map[string]string{
			"id":  statusResponse.ID,
			"url": statusResponse.URL,
		})
		log.Printf("Status Posted: %s, URL: %s", statusResponse.ID, statusResponse.URL)
	}

	return nil
}

func parseScheduledAt(input string) (string, error) {
	if input == "" {
		return "", nil // No scheduling requested
	}

	t, err := time.Parse("2006-01-02 15:04", input)
	if err != nil {
		return "", fmt.Errorf("invalid date format: %w", err)
	}

	if time.Until(t) < 5*time.Minute {
		return "", fmt.Errorf("scheduled time must be at least 5 minutes in the future")
	}

	return t.Format(time.RFC3339), nil
}
