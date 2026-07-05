package keycloak

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"expense-splitter/types"
)

var (
	ErrInvalidCredentials = errors.New("keycloak: invalid credentials")
	ErrUserExists         = errors.New("keycloak: user already exists")
	ErrUnavailable        = errors.New("keycloak: provider unavailable")
	ErrNotConfigured      = errors.New("keycloak: server-side calls not configured")
)

type Client struct {
	cfg  types.KeycloakConfig
	http *http.Client
}

func New(cfg types.KeycloakConfig) *Client {
	return &Client{cfg: cfg, http: &http.Client{Timeout: 10 * time.Second}}
}

type Token struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
}

// Login exchanges username/password for tokens via the realm's public client
// (OAuth2 password grant / Keycloak "direct access grant").
func (c *Client) Login(ctx context.Context, username, password string) (*Token, error) {
	if !c.cfg.LoginEnabled() {
		return nil, ErrNotConfigured
	}
	form := url.Values{
		"grant_type": {"password"},
		"client_id":  {c.cfg.ClientID},
		"username":   {username},
		"password":   {password},
	}
	tok, status, err := c.token(ctx, c.cfg.Realm, form)
	switch {
	case err != nil:
		return nil, ErrUnavailable
	case status == http.StatusUnauthorized || status == http.StatusBadRequest:
		return nil, ErrInvalidCredentials
	case status != http.StatusOK:
		return nil, ErrUnavailable
	}
	return tok, nil
}

// CreateUser provisions a new enabled user with a permanent password via the
// admin REST API and returns the Keycloak user id (the token "sub").
func (c *Client) CreateUser(ctx context.Context, email, password, name string) (string, error) {
	if !c.cfg.AdminEnabled() {
		return "", ErrNotConfigured
	}
	admin, err := c.adminToken(ctx)
	if err != nil {
		return "", err
	}

	first, last := splitName(name)
	// Keycloak's default user profile requires BOTH names; a missing one fails
	// later logins with "Account is not fully set up". Fall back rather than
	// send empties.
	if first == "" {
		if i := strings.Index(email, "@"); i > 0 {
			first = email[:i]
		} else {
			first = email
		}
	}
	if last == "" {
		last = first
	}
	body, err := json.Marshal(map[string]any{
		"username":      email,
		"email":         email,
		"firstName":     first,
		"lastName":      last,
		"enabled":       true,
		"emailVerified": true,
		"credentials": []map[string]any{{
			"type":      "password",
			"value":     password,
			"temporary": false,
		}},
	})
	if err != nil {
		return "", err
	}

	endpoint := fmt.Sprintf("%s/admin/realms/%s/users", c.cfg.BaseURL, url.PathEscape(c.cfg.Realm))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(string(body)))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+admin.AccessToken)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", ErrUnavailable
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusCreated:
		return userIDFromLocation(resp.Header.Get("Location")), nil
	case http.StatusConflict:
		return "", ErrUserExists
	default:
		return "", ErrUnavailable
	}
}

// adminToken authenticates against the master realm's admin-cli client.
func (c *Client) adminToken(ctx context.Context) (*Token, error) {
	form := url.Values{
		"grant_type": {"password"},
		"client_id":  {"admin-cli"},
		"username":   {c.cfg.AdminUser},
		"password":   {c.cfg.AdminPassword},
	}
	tok, status, err := c.token(ctx, "master", form)
	if err != nil || status != http.StatusOK {
		return nil, ErrUnavailable
	}
	return tok, nil
}

func (c *Client) token(ctx context.Context, realm string, form url.Values) (*Token, int, error) {
	endpoint := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/token", c.cfg.BaseURL, url.PathEscape(realm))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, resp.StatusCode, nil
	}
	var tok Token
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return nil, resp.StatusCode, err
	}
	return &tok, resp.StatusCode, nil
}

func userIDFromLocation(loc string) string {
	if i := strings.LastIndex(loc, "/"); i >= 0 && i < len(loc)-1 {
		return loc[i+1:]
	}
	return ""
}

func splitName(name string) (first, last string) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", ""
	}
	parts := strings.SplitN(name, " ", 2)
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], strings.TrimSpace(parts[1])
}
