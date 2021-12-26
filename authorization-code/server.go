package main

import (
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/sessions"
)

type Client struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scope        []string
	AuthzURL     string
	TokenURL     string
	GrantType    string
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	IDToken      string `json:"id_token"`
}

var templates = make(map[string]*template.Template)

var store = sessions.NewCookieStore([]byte("SESSION_KEY"))

var client = &Client{
	AuthzURL:     "https://demo.identityserver.io/connect/authorize",
	TokenURL:     "https://demo.identityserver.io/connect/token",
	ClientID:     "interactive.confidential",
	ClientSecret: "secret",
	RedirectURL:  "http://localhost:8080/callback",
}

func authzCodeHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")
	if session.Values["status"] == true {
		err := templates["index"].Execute(w, struct {
			result interface{}
		}{
			result: session.Values["tokenData"],
		})
		if err != nil {
			log.Printf("failed to execute template: %v", err)
		}
		return
	}
	u, err := url.Parse(client.AuthzURL)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	state := "abc"
	v := u.Query()
	v.Set("response_type", "code")
	v.Set("client_id", client.ClientID)
	v.Set("redirect_uri", client.RedirectURL)
	v.Set("scope", "openid")
	v.Set("state", state)
	v.Set("code_challenge", "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM")
	v.Set("code_challenge_method", "S256")
	u.RawQuery = v.Encode()
	session.Values["state"] = state
	if err := session.Save(r, w); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	w.Header().Set("location", u.String())
	w.WriteHeader(http.StatusMovedPermanently)
}

func authzCodeCallbackHandler(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	code := params.Get("code")
	state := params.Get("state")
	session, err := store.Get(r, "session")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	if state != session.Values["state"] {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	v := url.Values{}
	v.Set("grant_type", "authorization_code")
	v.Set("client_id", client.ClientID)
	v.Set("client_secret", client.ClientSecret)
	v.Set("redirect_uri", client.RedirectURL)
	v.Set("code_verifier", "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk")
	v.Set("code", code)
	tokenRes, err := http.Post(client.TokenURL, "application/x-www-form-urlencoded", strings.NewReader((v.Encode())))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	body, err := ioutil.ReadAll(tokenRes.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	session.Values["tokenData"] = string(body)
	session.Values["status"] = true
	if err := session.Save(r, w); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	w.Header().Set("location", "")
	w.WriteHeader(http.StatusMovedPermanently)
}

func loadTemplate(name string) *template.Template {
	t, err := template.ParseFiles("public/views/" + name + ".html")
	if err != nil {
		log.Fatalf("template error: %v", err)
	}
	return t
}

func main() {
	templates["index"] = loadTemplate("index")
	r := chi.NewRouter()
	r.Get("/", authzCodeHandler)
	r.Get("/callback", authzCodeCallbackHandler)
	http.ListenAndServe(":3333", r)
}
