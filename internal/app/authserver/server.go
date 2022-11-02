package authserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/SButnyakov/music-backend-authorization/internal/app/model"
	"github.com/SButnyakov/music-backend-authorization/internal/app/store"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/sirupsen/logrus"
)

const (
	sessionName        = "music_auth_cookie"
	ctxKeyUser  ctxKey = iota
	ctxKeyRequestId
)

var (
	errIncorrectEmailOrPassword = errors.New("incorrect login or password")
	errNotAuthenticated         = errors.New("not authenticated")
)

type ctxKey int8

type server struct {
	router       *mux.Router
	logger       *logrus.Logger
	store        store.Store
	sessionStore sessions.Store
}

func newServer(store store.Store, sessionsStore sessions.Store) *server {
	s := &server{
		router:       mux.NewRouter(),
		logger:       logrus.New(),
		store:        store,
		sessionStore: sessionsStore,
	}

	s.configureRouter()

	return s
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func (s *server) configureRouter() {
	s.router.Use(s.setRequestID)
	s.router.Use(s.logRequest)
	s.router.HandleFunc("/users", s.handleUsersCreate()).Methods(http.MethodPost, http.MethodOptions)
	s.router.HandleFunc("/sessions", s.handleSessionsCreate()).Methods(http.MethodPost, http.MethodOptions)
	s.router.HandleFunc("/updateCookie", s.handleCookieUpdate()).Methods(http.MethodPost, http.MethodOptions)
	s.router.HandleFunc("/checkCookie", s.handleCookieCheck()).Methods(http.MethodPost, http.MethodOptions)
	s.router.HandleFunc("/users/{id}", s.handleUserById()).Methods(http.MethodGet, http.MethodOptions)
	s.router.Use(mux.CORSMethodMiddleware(s.router))

	private := s.router.PathPrefix("/private").Subrouter()
	private.Use(s.authenticateUser)
	private.HandleFunc("/whoami", s.handleWhoami()).Methods("GET")
}

func (s *server) setRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := uuid.New().String()
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKeyRequestId, id)))
	})
}

func (s *server) logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := s.logger.WithFields(logrus.Fields{
			"remote_addr": r.RemoteAddr,
			"request_id":  r.Context().Value(ctxKeyRequestId),
		})
		// started [METHOD] /[ENDPOINT]
		logger.Infof("started %s %s", r.Method, r.RequestURI)

		start := time.Now()
		rw := &responseWriter{w, http.StatusOK}
		next.ServeHTTP(rw, r)

		logger.Infof(
			"completed with %d %s in %v",
			rw.code,
			http.StatusText(rw.code),
			time.Now().Sub(start))
	})
}

func (s *server) prepareHeaders(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "http://localhost")
	(*w).Header().Set("Access-Control-Allow-Credentials", "true")
	(*w).Header().Set("Access-Control-Allow-Headers", "access-control-allow-credentials,access-control-allow-origin,content-type")
}

func (s *server) authenticateUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := s.sessionStore.Get(r, sessionName)
		if err != nil {
			s.error(w, r, http.StatusInternalServerError, err)
			return
		}

		id, ok := session.Values["user_id"]
		if !ok {
			s.error(w, r, http.StatusUnauthorized, errNotAuthenticated)
			return
		}

		u, err := s.store.User().Find(id.(int))
		if err != nil {
			s.error(w, r, http.StatusUnauthorized, errNotAuthenticated)
			return
		}

		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKeyUser, u)))
	})
}

func (s *server) handleWhoami() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.respond(w, r, http.StatusOK, r.Context().Value(ctxKeyUser).(*model.User))
	}
}

func (s *server) handleUserById() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.prepareHeaders(&w)

		if r.Method == http.MethodOptions {
			s.respond(w, r, http.StatusOK, nil)
			return
		}

		paramId := mux.Vars(r)["id"]

		id, err := strconv.Atoi(paramId)
		if err != nil {
			s.error(w, r, http.StatusBadRequest, err)
			return
		}
		
		u, err := s.store.User().Find(id)
		if err != nil {
			s.error(w, r, http.StatusNoContent, err)
			return
		}

		fmt.Println("OK")

		s.respond(w, r, http.StatusOK, u)
	}
}

func (s *server) handleUsersCreate() http.HandlerFunc {
	type request struct {
		Login    string `json:"login"`
		Password string `json:"password"`
		StageName string `json:"stage_name"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		s.prepareHeaders(&w)

		if r.Method == http.MethodOptions {
			s.respond(w, r, http.StatusOK, nil)
			return
		}

		req := &request{}
		if err := json.NewDecoder(r.Body).Decode(req); err != nil {
			s.error(w, r, http.StatusBadRequest, err)
			return
		}

		u := &model.User{
			Login:    req.Login,
			Password: req.Password,
			StageName: req.StageName,
			AuthCookie: " ",
		}

		if err := s.store.User().Create(u); err != nil {
			s.error(w, r, http.StatusUnprocessableEntity, err)
			return
		}

		u.Sanitize()
		s.respond(w, r, http.StatusCreated, u)
	}
}

func (s *server) handleSessionsCreate() http.HandlerFunc {
	type request struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		s.prepareHeaders(&w)

		if r.Method == http.MethodOptions {
			s.respond(w, r, http.StatusOK, nil)
			return
		}

		req := &request{}

		if err := json.NewDecoder(r.Body).Decode(req); err != nil {
			s.error(w, r, http.StatusBadRequest, err)
			return
		}

		u, err := s.store.User().FindByLogin(req.Login)
		if err != nil || !u.ComparePassword(req.Password) {
			s.error(w, r, http.StatusUnauthorized, errIncorrectEmailOrPassword)
			return
		}

		session, err := s.sessionStore.Get(r, sessionName)
		if err != nil {
			s.error(w, r, http.StatusInternalServerError, err)
			return
		}

		session.Values["user_id"] = u.ID
		if s.sessionStore.Save(r, w, session); err != nil {
			s.error(w, r, http.StatusInternalServerError, err)
			return
		}

		s.respond(w, r, http.StatusOK, u.ID)
	}
}

func (s *server) handleCookieUpdate() http.HandlerFunc {
	type request struct {
		Login    string `json:"login"`
		Cookie   string `json:"auth_cookie"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		s.prepareHeaders(&w)

		if r.Method == http.MethodOptions {
			s.respond(w, r, http.StatusOK, nil)
			return
		}

		req := &request{}

		if err := json.NewDecoder(r.Body).Decode(req); err != nil {
			s.error(w, r, http.StatusBadRequest, err)
			return
		}

		if err := s.store.User().UpdateCookie(req.Login, req.Cookie); err != nil {
			s.error(w, r, http.StatusInternalServerError, err)
			return
		}

		s.respond(w, r, http.StatusOK, nil)
	}
}

func (s *server) handleCookieCheck() http.HandlerFunc {
	type request struct {
		Cookie   string `json:"auth_cookie"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		s.prepareHeaders(&w)

		if r.Method == http.MethodOptions {
			s.respond(w, r, http.StatusOK, nil)
			return
		}

		req := &request{}

		if err := json.NewDecoder(r.Body).Decode(req); err != nil {
			s.error(w, r, http.StatusBadRequest, err)
			return
		}

		u, err := s.store.User().FindByCookie(req.Cookie)
		if err != nil {
			s.error(w, r, http.StatusUnauthorized, err)
			return
		}

		s.respond(w, r, http.StatusOK, u)
	}
}

func (s *server) error(w http.ResponseWriter, r *http.Request, code int, err error) {
	s.respond(w, r, code, map[string]string{"error": err.Error()})
}

func (s *server) respond(w http.ResponseWriter, r *http.Request, code int, data interface{}) {
	w.WriteHeader(code)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}
