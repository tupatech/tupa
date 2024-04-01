package tupa

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/linkedin"
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
	GoogleWentWrongRedirUrl string

	/// LINKEDIN
	linkedinOauthInitOnce sync.Once
	LinkedinOauthConfig   = &oauth2.Config{
		ClientID:     "",
		ClientSecret: "",
		RedirectURL:  "",
		Scopes:       []string{""},
		Endpoint:     linkedin.Endpoint,
	}
	LinkedingWentWrongRedirUrl string
)

type GoogleDefaultResponse struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	Locale        string `json:"locale"`
	HostedDomain  string `json:"hd"`
}

type GoogleAuthResponse struct {
	UserInfo GoogleDefaultResponse
	Token    *oauth2.Token
}

type LinkedinAuthResponse struct {
	UserInfo LinkedinUserInfo
	Token    *oauth2.Token
}

type LinkedinUserInfo struct {
	Name          string             `json:"name"`
	Email         string             `json:"email"`
	EmailVerified bool               `json:"email_verified"`
	FamilyName    string             `json:"family_name"`
	GivenName     string             `json:"given_name"`
	Locale        LinkedinUserLocale `json:"locale"`
}

type LinkedinUserLocale struct {
	Country  string `json:"country"`
	Language string `json:"language"`
}

func UseGoogleOauth(clientID, clientSecret, redirectURL, googleWentWrongdRedirectURL string, scopes []string) {
	googleOauthInitOnce.Do(func() {
		GoogleOauthConfig.ClientID = clientID
		GoogleOauthConfig.ClientSecret = clientSecret
		GoogleOauthConfig.RedirectURL = redirectURL
		GoogleOauthConfig.Scopes = scopes
		GoogleWentWrongRedirUrl = googleWentWrongdRedirectURL
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

func AuthGoogleCallback(tc *TupaContext) (*GoogleAuthResponse, error) {
	code := tc.Request().FormValue("code")
	if code == "" {
		log.Println("Usuário não aceitou a autenticação...")
		reason := tc.Request().FormValue("error")
		if reason == "user_denied" {
			log.Println("Usuário negou a permissão...")
		}

		http.Redirect(*tc.Response(), tc.Request(), GoogleWentWrongRedirUrl, http.StatusTemporaryRedirect)
		return nil, nil
	}

	token, err := GoogleOauthConfig.Exchange(context.Background(), code)
	if err != nil {
		fmt.Printf("Exchange do código falhou '%s'\n", err)
		return nil, err
	}

	resp, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + url.QueryEscape(token.AccessToken))
	if err != nil {
		http.Redirect(*tc.Response(), tc.Request(), GoogleWentWrongRedirUrl, http.StatusTemporaryRedirect)
		return nil, err
	}
	defer resp.Body.Close()

	response, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Redirect(*tc.Response(), tc.Request(), GoogleWentWrongRedirUrl, http.StatusTemporaryRedirect)
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

func UseLinkedinOauth(clientID, clientSecret, redirectURL, linkedinWentWrongRedirUrl string, scopes []string) {
	linkedinOauthInitOnce.Do(func() {
		LinkedinOauthConfig.ClientID = clientID
		LinkedinOauthConfig.ClientSecret = clientSecret
		LinkedinOauthConfig.RedirectURL = redirectURL
		LinkedinOauthConfig.Scopes = scopes
		LinkedingWentWrongRedirUrl = linkedinWentWrongRedirUrl
	})
}

func AuthLinkedinHandler(tc *TupaContext) error {
	URL, err := url.Parse(LinkedinOauthConfig.Endpoint.AuthURL)
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	parameters := url.Values{}
	parameters.Add("client_id", LinkedinOauthConfig.ClientID)
	parameters.Add("scope", strings.Join(LinkedinOauthConfig.Scopes, " "))
	parameters.Add("redirect_uri", LinkedinOauthConfig.RedirectURL)
	parameters.Add("response_type", "code")

	URL.RawQuery = parameters.Encode()
	url := URL.String()

	http.Redirect((*tc.Response()), tc.Request(), url, http.StatusTemporaryRedirect)
	return nil
}

func AuthLinkedinCallback(w http.ResponseWriter, r *http.Request) (map[string]interface{}, error) {
	code := r.FormValue("code")
	if code == "" {
		log.Println("Usuário não aceitou a autenticação...")

		http.Redirect(w, r, LinkedingWentWrongRedirUrl, http.StatusPermanentRedirect)
		return nil, nil
	}

	token, err := LinkedinOauthConfig.Exchange(context.Background(), code)
	if err != nil {
		fmt.Printf("Exchange do código falhou '%s'\n", err)
		return nil, err
	}

	req, err := http.NewRequest("GET", "https://api.linkedin.com/v2/userinfo", nil)
	if err != nil {
		http.Redirect(w, r, LinkedingWentWrongRedirUrl, http.StatusTemporaryRedirect)
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil
	}
	defer resp.Body.Close()

	var userInfo map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&userInfo)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"UserInfo": userInfo,
		"Token":    token,
	}, nil
}
