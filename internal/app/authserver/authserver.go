package authserver

import (
	"database/sql"
	"net/http"

	"github.com/SButnyakov/music-backend-authorization/internal/app/store/sqlstore"
	"github.com/gorilla/sessions"
)

func Start(config *Config) error {
	db, err := newDB(config.DatabaseURL)
	if err != nil {
		return err
	}

	defer db.Close()
	store := sqlstore.New(db)
	sessionStore := sessions.NewCookieStore([]byte(config.SessionKey))
	sessionStore.Options = &sessions.Options{
		MaxAge: 86400,
		SameSite: 4, // None
		Secure: true,
	}
	s := newServer(store, sessionStore)

	return http.ListenAndServe(config.BindAddr, s)
}

func newDB(databaseURL string) (*sql.DB, error) {
	db, err := sql.Open("sqlserver", databaseURL)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}
