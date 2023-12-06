package zendesk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/nukosuke/go-zendesk/zendesk"
	"github.com/spf13/viper"
)

// will be set by ldflags
var ClientID string
var ClientSecret string

type Client struct {
	*zendesk.Client
	subdomain string
}

func NewClientFromViper(v *viper.Viper) (*Client, error) {
	subdomain := v.GetString("zendesk.subdomain")
	token := v.GetString("zendesk.bearer-token")
	return NewClient(subdomain, token)
}

func NewClient(subdomain, token string) (*Client, error) {
	// You can set custom *http.Client here
	client, err := zendesk.NewClient(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create zendesk client: %w", err)
	}

	err = client.SetSubdomain(subdomain)
	if err != nil {
		return nil, fmt.Errorf("failed to set subdomain: %w", err)
	}
	if token != "" {
		client.SetCredential(zendesk.NewBearerTokenCredential(token))
	}

	return &Client{
		Client:    client,
		subdomain: subdomain,
	}, nil
}

func (c *Client) GetBearerToken(ctx context.Context, email, password string) (string, error) {
	b, err := json.Marshal(map[string]string{
		"grant_type":    "password",
		"client_id":     ClientID,
		"client_secret": ClientSecret,
		"scope":         "read",
		"username":      email,
		"password":      password,
	})
	if err != nil {
		return "", fmt.Errorf("Error marshalling body: %s", err)
	}
	url := "https://" + c.subdomain + ".zendesk.com/oauth/tokens"
	resp, err := http.Post(
		url,
		"application/json",
		bytes.NewReader(b),
	)
	if err != nil {
		return "", fmt.Errorf("Error posting to %s: %s", url, err)
	}
	defer resp.Body.Close()

	result := struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		Scope       string `json:"scope"`

		// error
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return "", fmt.Errorf("Error decoding response: %s", err)
	}

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("Error getting bearer token: %s - %s", result.Error, result.ErrorDescription)
	}

	return result.AccessToken, nil
}
