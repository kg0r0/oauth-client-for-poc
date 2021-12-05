package main

import (
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/labstack/echo/v4"
)

type Client struct {
	ClientID     string
	ClientSecret string
	Scope        []string
	TokenURL     string
	GrantType    string
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

type Template struct {
	templates *template.Template
}

var client = &Client{
	TokenURL:     "https://demo.identityserver.io/connect/token",
	ClientID:     "m2m",
	ClientSecret: "secret",
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func ClientCredentialsHandler(c echo.Context) error {
	v := url.Values{}
	v.Set("grant_type", "client_credentials")
	v.Set("scope", "api")
	cli := &http.Client{}
	req, _ := http.NewRequest("POST", client.TokenURL, strings.NewReader((v.Encode())))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(client.ClientID, client.ClientSecret)
	rsp, err := cli.Do(req)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Error")
	}
	defer rsp.Body.Close()
	body, _ := ioutil.ReadAll(rsp.Body)
	return c.Render(http.StatusOK, "result", string(body))
}

func main() {
	t := &Template{
		templates: template.Must(template.ParseGlob("public/views/*.html")),
	}
	e := echo.New()
	e.Renderer = t
	e.GET("/", ClientCredentialsHandler)
	if err := e.Start(":8080"); err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
