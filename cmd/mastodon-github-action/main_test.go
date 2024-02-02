package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPostStatus(t *testing.T) {
	tests := []struct {
		name           string
		status         MastodonStatus
		mockResponse   string
		mockStatusCode int
		wantErr        bool
	}{
		{
			name: "successful post",
			status: MastodonStatus{
				Status:     "Test status",
				Visibility: string(VisibilityPublic),
			},
			mockResponse:   `{"id":"123","url":"http://example.com/status/123","content":"Test status","created_at":"2020-01-01T00:00:00Z","visibility":"public"}`,
			mockStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name: "error from server",
			status: MastodonStatus{
				Status:     "Test status",
				Visibility: string(VisibilityPublic),
			},
			mockResponse:   `{"error":"Internal server error"}`,
			mockStatusCode: http.StatusInternalServerError,
			wantErr:        true,
		},
		{
			name: "successful unlisted post",
			status: MastodonStatus{
				Status:     "Unlisted status",
				Visibility: string(VisibilityUnlisted),
			},
			mockResponse:   `{"id":"124","url":"http://example.com/status/124","content":"Unlisted status","created_at":"2020-01-02T00:00:00Z","visibility":"unlisted"}`,
			mockStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name: "successful private post",
			status: MastodonStatus{
				Status:     "Private status",
				Visibility: string(VisibilityPrivate),
			},
			mockResponse:   `{"id":"125","url":"http://example.com/status/125","content":"Private status","created_at":"2020-01-03T00:00:00Z","visibility":"private"}`,
			mockStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name: "successful direct post",
			status: MastodonStatus{
				Status:     "Direct message",
				Visibility: string(VisibilityDirect),
			},
			mockResponse:   `{"id":"126","url":"http://example.com/status/126","content":"Direct message","created_at":"2020-01-04T00:00:00Z","visibility":"direct"}`,
			mockStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name: "successful post with sensitive content",
			status: MastodonStatus{
				Status:     "Sensitive status",
				Visibility: string(VisibilityPublic),
				Sensitive:  true,
			},
			mockResponse:   `{"id":"127","url":"http://example.com/status/127","content":"Sensitive status","created_at":"2020-01-05T00:00:00Z","visibility":"public"}`,
			mockStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name: "successful post with spoiler text",
			status: MastodonStatus{
				Status:      "Spoiler status",
				Visibility:  string(VisibilityPublic),
				SpoilerText: "Spoiler alert!",
			},
			mockResponse:   `{"id":"128","url":"http://example.com/status/128","content":"Spoiler status","created_at":"2020-01-06T00:00:00Z","visibility":"public"}`,
			mockStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name: "successful scheduled post",
			status: MastodonStatus{
				Status:      "Scheduled status",
				Visibility:  string(VisibilityPublic),
				ScheduledAt: "2020-01-07T00:00:00Z",
			},
			mockResponse:   `{"id":"129","scheduled_at":"2020-01-07T00:00:00Z"}`,
			mockStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name: "post with invalid visibility",
			status: MastodonStatus{
				Status:     "Bad visibility status",
				Visibility: "invalid",
			},
			mockResponse:   `{"error":"Invalid visibility"}`,
			mockStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
		{
			name: "API rate limit exceeded",
			status: MastodonStatus{
				Status:     "Rate limit status",
				Visibility: string(VisibilityPublic),
			},
			mockResponse:   `{"error":"Rate limit exceeded"}`,
			mockStatusCode: http.StatusTooManyRequests,
			wantErr:        true,
		},
		{
			name: "API unauthorized access",
			status: MastodonStatus{
				Status:     "Unauthorized status",
				Visibility: string(VisibilityPublic),
			},
			mockResponse:   `{"error":"Unauthorized"}`,
			mockStatusCode: http.StatusUnauthorized,
			wantErr:        true,
		},
		{
			name: "invalid scheduled_at format",
			status: MastodonStatus{
				Status:      "Invalid scheduled_at format",
				Visibility:  string(VisibilityPublic),
				ScheduledAt: "invalid-date",
			},
			mockResponse:   ``, // No mock response needed as the error should occur before the API call
			mockStatusCode: 0,  // Status code is irrelevant in this case
			wantErr:        true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.mockStatusCode)
				fmt.Fprintln(w, tc.mockResponse)
			}))
			defer mockServer.Close()

			args := ActionInputs{
				URL:         mockServer.URL, // Use mock server URL
				AccessToken: "fake-token",
				Message:     tc.status.Status,
				Visibility:  tc.status.Visibility,
				Sensitive:   tc.status.Sensitive,
				SpoilerText: tc.status.SpoilerText,
				Language:    tc.status.Language,
				ScheduledAt: tc.status.ScheduledAt,
			}

			err := postStatus(args.URL, args.AccessToken, tc.status)

			if (err != nil) != tc.wantErr {
				t.Errorf("postStatus() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}
