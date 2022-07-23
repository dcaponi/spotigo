package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/http"
	"os"

	"github.com/dcaponi/spotigo/src/handlers"
	"github.com/dcaponi/spotigo/src/repositories"
	"github.com/dcaponi/spotigo/src/repositories/db_entities"
	"github.com/dcaponi/spotigo/src/services"
	newrelic "github.com/newrelic/go-agent"
	"github.com/robfig/cron/v3"
	"github.com/zmb3/spotify"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// Get environment variables
	newRelicAppName := os.Getenv("SPOTIFY_SLACK_APP_NEW_RELIC_APP_NAME")
	newRelicLicense := os.Getenv("NEW_RELIC_LICENSE_KEY")
	databaseURL := os.Getenv("SPOTIFY_SLACK_APP_DATABASE_URL")
	slackAuthURL := os.Getenv("SPOTIFY_SLACK_APP_SLACK_AUTH_URL")
	spotifyRedirectURL := os.Getenv("SPOTIFY_SLACK_APP_SPOTIFY_REDIRECT_URL")
	slackClientID := os.Getenv("SPOTIFY_SLACK_APP_SLACK_CLIENT_ID")
	slackClientSecret := os.Getenv("SPOTIFY_SLACK_APP_SLACK_CLIENT_SECRET")
	port := os.Getenv("PORT")

	newrelicConfig := newrelic.NewConfig(newRelicAppName, newRelicLicense)
	newrelicApp, err := newrelic.NewApplication(newrelicConfig)
	if err != nil {
		fmt.Errorf("Error: %s\n", err)
	}

	// Setup connection to the database
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	db.AutoMigrate(&db_entities.User{})

	spotifyAuthenticator := spotify.NewAuthenticator(spotifyRedirectURL, spotify.ScopeUserReadCurrentlyPlaying)

	// Creating app layers (repositories, services, handlers)
	repositories := repositories.NewRepository(db)
	services := services.NewServices(repositories, spotifyAuthenticator)
	handlers := handlers.NewHandlers(services, spotifyAuthenticator, stateGenerator(), slackClientID, slackClientSecret, slackAuthURL)

	// Setup cronjob for updating status
	c := cron.New(cron.WithSeconds())
	c.AddFunc("@every 10s", func() { services.ChangeUserStatus(context.Background()) })
	c.Start()

	// Add 3p callback handlers
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", handlers.SpotifyCallbackHandler)
	mux.HandleFunc("/slackAuth", handlers.SlackCallbackHandler)
	mux.HandleFunc(newrelic.WrapHandleFunc(newrelicApp, "/users", handlers.HealthHandler))
	fs := http.FileServer(http.Dir("./static"))
	mux.Handle("/", fs)

	http.ListenAndServe(":"+port, mux)
}

func stateGenerator() string {
	b := make([]byte, 4)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}
