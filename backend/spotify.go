package backend

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Spotify API integration for track search/matching only
// Uses Client Credentials flow (no user login required)

const (
	spotifyTokenURL = "https://accounts.spotify.com/api/token"
	spotifyAPIBase  = "https://api.spotify.com/v1"

	// Internal credentials for search (Client Credentials flow)
	spotifyInternalClientID     = "5f573c9620494bae87890c0f08a60293"
	spotifyInternalClientSecret = "212476d9b0f3472eaa762d90b19b0ba8"
)

// spotifyTokens stores access token for API requests (internal use only)
type spotifyTokens struct {
	AccessToken string
	ExpiresAt   int64
}

// SpotifyClient handles Spotify API search requests
type SpotifyClient struct {
	clientID     string
	clientSecret string
	tokens       *spotifyTokens
	httpClient   *http.Client
	mu           sync.Mutex
}

// SpotifyTrack represents a track from Spotify search results
type SpotifyTrack struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Artists  string `json:"artists"`
	Album    string `json:"album"`
	Duration int    `json:"durationMs"`
	URI      string `json:"uri"`
	ISRC     string `json:"isrc,omitempty"`
}

// NewSpotifyClientForSearch creates a search-only client using Client Credentials
func NewSpotifyClientForSearch() *SpotifyClient {
	client := &SpotifyClient{
		clientID:     spotifyInternalClientID,
		clientSecret: spotifyInternalClientSecret,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// Get access token using Client Credentials flow
	if err := client.authenticateClientCredentials(); err != nil {
		fmt.Printf("Warning: Could not authenticate Spotify: %v\n", err)
	}

	return client
}

// authenticateClientCredentials gets an access token using Client Credentials flow
func (c *SpotifyClient) authenticateClientCredentials() error {
	data := url.Values{}
	data.Set("grant_type", "client_credentials")

	req, err := http.NewRequest("POST", spotifyTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(c.clientID, c.clientSecret)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("token request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		TokenType   string `json:"token_type"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return fmt.Errorf("failed to decode token response: %w", err)
	}

	c.mu.Lock()
	c.tokens = &spotifyTokens{
		AccessToken: tokenResp.AccessToken,
		ExpiresAt:   time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second).Unix(),
	}
	c.mu.Unlock()

	return nil
}

// ensureValidToken ensures we have a valid access token
func (c *SpotifyClient) ensureValidToken() error {
	c.mu.Lock()
	tokens := c.tokens
	c.mu.Unlock()

	if tokens == nil {
		return c.authenticateClientCredentials()
	}

	// Refresh if token expires in next 60 seconds
	if time.Now().Unix() >= tokens.ExpiresAt-60 {
		return c.authenticateClientCredentials()
	}

	return nil
}

// doRequest makes an authenticated request to Spotify API
func (c *SpotifyClient) doRequest(method, endpoint string, body io.Reader) ([]byte, error) {
	if err := c.ensureValidToken(); err != nil {
		return nil, err
	}

	req, err := http.NewRequest(method, spotifyAPIBase+endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.mu.Lock()
	token := c.tokens.AccessToken
	c.mu.Unlock()

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Handle rate limiting
	if resp.StatusCode == 429 {
		retryAfter := resp.Header.Get("Retry-After")
		return nil, fmt.Errorf("rate limited, retry after %s seconds", retryAfter)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// SearchByISRC searches for a track by ISRC code
func (c *SpotifyClient) SearchByISRC(isrc string) (*SpotifyTrack, error) {
	if isrc == "" {
		return nil, fmt.Errorf("ISRC is empty")
	}

	query := url.QueryEscape(fmt.Sprintf("isrc:%s", isrc))
	endpoint := fmt.Sprintf("/search?q=%s&type=track&limit=1", query)

	data, err := c.doRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var searchResp struct {
		Tracks struct {
			Items []struct {
				ID      string `json:"id"`
				Name    string `json:"name"`
				URI     string `json:"uri"`
				Artists []struct {
					Name string `json:"name"`
				} `json:"artists"`
				Album struct {
					Name string `json:"name"`
				} `json:"album"`
				DurationMs  int `json:"duration_ms"`
				ExternalIDs struct {
					ISRC string `json:"isrc"`
				} `json:"external_ids"`
			} `json:"items"`
		} `json:"tracks"`
	}

	if err := json.Unmarshal(data, &searchResp); err != nil {
		return nil, fmt.Errorf("failed to parse search results: %w", err)
	}

	if len(searchResp.Tracks.Items) == 0 {
		return nil, nil // No match found
	}

	item := searchResp.Tracks.Items[0]

	var artists []string
	for _, a := range item.Artists {
		artists = append(artists, a.Name)
	}

	return &SpotifyTrack{
		ID:       item.ID,
		Name:     item.Name,
		Artists:  strings.Join(artists, ", "),
		Album:    item.Album.Name,
		Duration: item.DurationMs,
		URI:      item.URI,
		ISRC:     item.ExternalIDs.ISRC,
	}, nil
}

// SearchByQuery searches for tracks by text query
func (c *SpotifyClient) SearchByQuery(query string, limit int) ([]SpotifyTrack, error) {
	if limit <= 0 {
		limit = 10
	}

	endpoint := fmt.Sprintf("/search?q=%s&type=track&limit=%d", url.QueryEscape(query), limit)

	data, err := c.doRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var searchResp struct {
		Tracks struct {
			Items []struct {
				ID      string `json:"id"`
				Name    string `json:"name"`
				URI     string `json:"uri"`
				Artists []struct {
					Name string `json:"name"`
				} `json:"artists"`
				Album struct {
					Name string `json:"name"`
				} `json:"album"`
				DurationMs  int `json:"duration_ms"`
				ExternalIDs struct {
					ISRC string `json:"isrc"`
				} `json:"external_ids"`
			} `json:"items"`
		} `json:"tracks"`
	}

	if err := json.Unmarshal(data, &searchResp); err != nil {
		return nil, fmt.Errorf("failed to parse search results: %w", err)
	}

	var tracks []SpotifyTrack
	for _, item := range searchResp.Tracks.Items {
		var artists []string
		for _, a := range item.Artists {
			artists = append(artists, a.Name)
		}

		tracks = append(tracks, SpotifyTrack{
			ID:       item.ID,
			Name:     item.Name,
			Artists:  strings.Join(artists, ", "),
			Album:    item.Album.Name,
			Duration: item.DurationMs,
			URI:      item.URI,
			ISRC:     item.ExternalIDs.ISRC,
		})
	}

	return tracks, nil
}
