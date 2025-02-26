package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/dghubble/gologin/v2"
	"github.com/dghubble/gologin/v2/facebook"
	"github.com/dghubble/gologin/v2/google"
	"github.com/dghubble/sessions"
	"golang.org/x/oauth2"
	facebookOAuth2 "golang.org/x/oauth2/facebook"
	googleOAuth2 "golang.org/x/oauth2/google"
)

const (
	sessionName     = "example-google-app"
	sessionSecret   = "example cookie signing secret"
	sessionUserKey  = "googleID"
	sessionUsername = "googleName"
)

// sessionStore encodes and decodes session data stored in signed cookies
var sessionStore = sessions.NewCookieStore([]byte(sessionSecret), nil)

// Config configures the main ServeMux.
type Config struct {
	ClientID     string
	ClientSecret string
}

// New returns a new ServeMux with app routes.
func New(config *Config) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/", profileHandler)
	mux.HandleFunc("/logout", logoutHandler)
	// 1. Register Login and Callback handlers
	googleOauth2Config := &oauth2.Config{
		ClientID:     "990650209650-tidphg8cnge229cd3888st5jhkfk1g73.apps.googleusercontent.com",
		ClientSecret: "-sosFxmYeGyU-Pe6QutUvGlo",
		RedirectURL:  "http://localhost:8080/google/callback",
		Endpoint:     googleOAuth2.Endpoint,
		Scopes:       []string{"profile", "email"},
	}

	facebookOauth2Config := &oauth2.Config{
		ClientID:     "872931336611060",
		ClientSecret: "df28dc60eaa5e26d625f19806207322d",
		RedirectURL:  "http://localhost:8080/google/callback",
		Endpoint:     facebookOAuth2.Endpoint,
	}

	// state param cookies require HTTPS by default; disable for localhost development

	stateConfig := gologin.DebugOnlyCookieConfig

	mux.Handle("/google/login", google.StateHandler(stateConfig, google.LoginHandler(googleOauth2Config, nil)))
	mux.Handle("/google/callback", google.StateHandler(stateConfig, google.CallbackHandler(googleOauth2Config, issueSession(), nil)))
	mux.Handle("/facebook/login", facebook.StateHandler(stateConfig, facebook.LoginHandler(facebookOauth2Config, nil)))
	mux.Handle("/facebook/callback", facebook.StateHandler(stateConfig, facebook.CallbackHandler(facebookOauth2Config, issueSession(), nil)))
	return mux
}

// issueSession issues a cookie session after successful Google login
func issueSession() http.Handler {
	fn := func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		googleUser, err := google.UserFromContext(ctx)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// 2. Implement a success handler to issue some form of session
		session := sessionStore.New(sessionName)
		session.Values[sessionUserKey] = googleUser.Id
		session.Values[sessionUsername] = googleUser.Name
		session.Save(w)
		http.Redirect(w, req, "/profile", http.StatusFound)
	}
	return http.HandlerFunc(fn)
}

// profileHandler shows a personal profile or a login button (unauthenticated).
func profileHandler(w http.ResponseWriter, req *http.Request) {
	session, err := sessionStore.Get(req, sessionName)
	if err != nil {
		// welcome with login button
		page, _ := ioutil.ReadFile("home.html")
		fmt.Fprintf(w, string(page))
		return
	}
	// authenticated profile
	fmt.Fprintf(w, `<p>You are logged in %s!</p><form action="/logout" method="post"><input type="submit" value="Logout"></form>`, session.Values[sessionUsername])
}

// logoutHandler destroys the session on POSTs and redirects to home.
func logoutHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method == "POST" {
		sessionStore.Destroy(w, sessionName)
	}
	http.Redirect(w, req, "/", http.StatusFound)
}

// main creates and starts a Server listening.
func main() {
	const address = "localhost:8080"
	// read credentials from environment variables if available
	config := &Config{
		ClientID:     "990650209650-tidphg8cnge229cd3888st5jhkfk1g73.apps.googleusercontent.com",
		ClientSecret: "-sosFxmYeGyU-Pe6QutUvGlo",
	}
	// allow consumer credential flags to override config fields
	clientID := flag.String("client-id", "", "990650209650-tidphg8cnge229cd3888st5jhkfk1g73.apps.googleusercontent.com")
	clientSecret := flag.String("client-secret", "", "-sosFxmYeGyU-Pe6QutUvGlo")

	flag.Parse()
	if *clientID != "" {
		config.ClientID = *clientID
	}
	if *clientSecret != "" {
		config.ClientSecret = *clientSecret
	}
	if config.ClientID == "" {
		log.Fatal("Missing Google Client ID")
	}
	if config.ClientSecret == "" {
		log.Fatal("Missing Google Client Secret")
	}

	log.Printf("Starting Server listening on %s\n", address)

	err := http.ListenAndServe(address, New(config))
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
