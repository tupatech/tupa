package tupa

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var (
	googleOauthInitOnce sync.Once
	GoogleOauthConfig   = &oauth2.Config{
		ClientID:     "",
		ClientSecret: "",
		RedirectURL:  "",
		Scopes:       []string{""},
		Endpoint:     google.Endpoint,
	}
)

type GoogleDefaultResponse struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Picture       string `json:"picture"`
	HostedDomain  string `json:"hd"`
}

type GoogleAuthResponse struct {
	UserInfo GoogleDefaultResponse
	Token    *oauth2.Token
}

func UseGoogleOauth(clientID, clientSecret, redirectURL string, scopes []string) {
	googleOauthInitOnce.Do(func() {
		GoogleOauthConfig.ClientID = clientID
		GoogleOauthConfig.ClientSecret = clientSecret
		GoogleOauthConfig.RedirectURL = redirectURL
		GoogleOauthConfig.Scopes = scopes
	})
}

func AuthGoogleHandler(tc *TupaContext) error {
	URL, err := url.Parse(GoogleOauthConfig.Endpoint.AuthURL)
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	parameters := url.Values{}
	parameters.Add("client_id", GoogleOauthConfig.ClientID)
	parameters.Add("scope", strings.Join(GoogleOauthConfig.Scopes, " "))
	parameters.Add("redirect_uri", GoogleOauthConfig.RedirectURL)
	parameters.Add("response_type", "code")

	URL.RawQuery = parameters.Encode()
	url := URL.String()

	http.Redirect((*tc.Response()), tc.Request(), url, http.StatusTemporaryRedirect)
	return nil
}

func AuthGoogleCallback(w http.ResponseWriter, r *http.Request) (*GoogleAuthResponse, error) {
	code := r.FormValue("code")
	if code == "" {
		w.Write([]byte("Usuário não aceitou a autenticação...\n"))
		reason := r.FormValue("error")
		if reason == "user_denied" {
			w.Write([]byte("Usuário negou a permissão..."))
		}

		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return nil, nil
	}

	token, err := GoogleOauthConfig.Exchange(context.Background(), code)
	if err != nil {
		fmt.Printf("Exchange do código falhou '%s'\n", err)
		return nil, err
	}

	resp, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + url.QueryEscape(token.AccessToken))
	if err != nil {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return nil, nil
	}
	defer resp.Body.Close()

	response, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return nil, nil
	}

	var userInfo GoogleDefaultResponse
	err = json.Unmarshal(response, &userInfo)
	if err != nil {
		return nil, err
	}

	return &GoogleAuthResponse{
		UserInfo: userInfo,
		Token:    token,
	}, nil
}
