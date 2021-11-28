package main

import (
	"encoding/gob"
	"encoding/json"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
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

type Template struct {
	templates *template.Template
}

var client = &Client{
	AuthzURL:     "https://accounts.google.com/o/oauth2/v2/auth",
	TokenURL:     "https://oauth2.googleapis.com/token",
	ClientID:     "",
	ClientSecret: "",
	RedirectURL:  "http://localhost:8080/callback",
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func authzCodeHandler(c echo.Context) error {
	sess, _ := session.Get("session", c)
	if sess.Values["status"] == true {
		return c.Render(http.StatusOK, "result", sess.Values["tokenData"])
	}
	u, err := url.Parse(client.AuthzURL)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Error")
	}
	state := "abc"
	v := u.Query()
	v.Set("response_type", "code")
	v.Set("client_id", client.ClientID)
	v.Set("redirect_uri", client.RedirectURL)
	v.Set("scope", "openid")
	v.Set("state", state)
	u.RawQuery = v.Encode()
	sess.Values["state"] = state
	if err := sess.Save(c.Request(), c.Response()); err != nil {
		log.Println("err: ", err)
		return c.NoContent((http.StatusInternalServerError))
	}
	return c.Redirect(http.StatusMovedPermanently, u.String())
}

func authzCodeCallbackHandler(c echo.Context) error {
	code := c.QueryParam("code")
	state := c.QueryParam("state")
	sess, err := session.Get("session", c)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Error")
	}
	if state != sess.Values["state"] {
		return c.String(http.StatusBadRequest, "400")
	}
	v := url.Values{}
	v.Set("grant_type", "authorization_code")
	v.Set("client_id", client.ClientID)
	v.Set("client_secret", client.ClientSecret)
	v.Set("redirect_uri", client.RedirectURL)
	v.Set("code", code)
	tokenRes, err := http.Post(client.TokenURL, "application/x-www-form-urlencoded", strings.NewReader((v.Encode())))
	if err != nil {
		return c.String(http.StatusBadRequest, "400")
	}
	var tokenData TokenResponse
	json.NewDecoder(tokenRes.Body).Decode(&tokenData)
	out, err := json.Marshal(tokenData)
	if err != nil {
		return c.NoContent((http.StatusInternalServerError))
	}
	sess.Values["tokenData"] = string(out)
	sess.Values["status"] = true
	if err := sess.Save(c.Request(), c.Response()); err != nil {
		log.Println("err: ", err)
		return c.NoContent((http.StatusInternalServerError))
	}
	return c.Redirect(http.StatusMovedPermanently, "/")
}

func main() {
	gob.Register(&TokenResponse{})
	t := &Template{
		templates: template.Must(template.ParseGlob("public/views/*.html")),
	}
	e := echo.New()
	e.Use(session.Middleware(sessions.NewCookieStore([]byte("secret"))))
	e.Renderer = t
	e.GET("/", authzCodeHandler)
	e.GET("/callback", authzCodeCallbackHandler)
	if err := e.Start(":8080"); err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
