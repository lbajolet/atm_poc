package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/lbajolet/atm_service/pkg/persistence"
	"github.com/rs/zerolog/log"
)

const SessionKeyCtx = "Context"

type Session struct {
	ID         uuid.UUID
	Account    persistence.Account
	Expiration time.Time
}

// IsValid checks that the session is still able to be used
func (s *Session) IsValid() bool {
	if !time.Now().Before(s.Expiration) {
		return false
	}

	// Auto-renew session if it expires in less than a minute
	if time.Now().Add(time.Minute).After(s.Expiration) {
		s.Renew()
	}
	return true
}

func (s *Session) Renew() {
	s.Expiration = time.Now().Add(10 * time.Minute)
}

// NewSession returns a new Session for the account
//
// Sessions are valid for 10 minutes after they're created
func NewSession(acc persistence.Account) *Session {
	session := &Session{
		ID:      uuid.New(),
		Account: acc,
	}
	session.Renew()
	return session
}

// AuthServer authenticates users that connect to routes that require authentication
type AuthServer struct {
	AuthMap *sync.Map
	Wrapped http.Handler
}

// NewAuthServer returns a new instance of AuthServer
func NewAuthServer(wrapped http.Handler) AuthServer {
	return AuthServer{
		AuthMap: &sync.Map{},
		Wrapped: wrapped,
	}
}

func (as AuthServer) NewSession(acc persistence.Account) (*Session, error) {
	sess := NewSession(acc)
	as.AuthMap.Store(sess.ID, sess)
	return sess, nil
}

// HandleAuthRequest checks that the authentication is valid before processing the request
func (as AuthServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		log.Error().Msg("missing auth header")
		w.WriteHeader(401)
		fmt.Fprint(w, "unauthorized")
		return
	}

	uuid, err := uuid.Parse(authHeader)
	if err != nil {
		log.Error().Str("Authorisation", authHeader).Msg("not a uuid")
		w.WriteHeader(400)
		fmt.Fprint(w, "invalid authorization")
		return
	}

	val, ok := as.AuthMap.Load(uuid)
	if !ok {
		log.Error().Str("Authorisation", authHeader).Msg("not in session cache")
		w.WriteHeader(401)
		fmt.Fprintf(w, "invalid authorization")
		return
	}

	sess := val.(*Session)
	if !sess.IsValid() {
		w.WriteHeader(401)
		fmt.Fprintf(w, "session expired")
		return
	}

	r = r.WithContext(context.WithValue(r.Context(), SessionKeyCtx, sess))

	as.Wrapped.ServeHTTP(w, r)
}

// Server serves the main routes for the public API
type Server struct {
	as  AuthServer
	db  *persistence.DB
	mux *http.ServeMux
}

func NewServer(db *persistence.DB) *Server {
	srv := &Server{
		db: db,
	}

	mux := &http.ServeMux{}
	mux.HandleFunc("/login", srv.login)

	authRoutesHandlers := &http.ServeMux{}
	authRoutesHandlers.HandleFunc("/balance", srv.getBalance)
	authRoutesHandlers.HandleFunc("/deposit", srv.doDeposit)
	authRoutesHandlers.HandleFunc("/withdraw", srv.doWithdrawal)

	srv.as = NewAuthServer(authRoutesHandlers)
	mux.Handle("/", srv.as)

	srv.mux = mux

	return srv
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	hdr := r.Header.Get("nip")
	if hdr == "" {
		w.WriteHeader(400)
		fmt.Fprint(w, "missing header: 'nip'")
		return
	}

	acc, err := s.db.Auth(hdr)
	if err != nil {
		w.WriteHeader(400)
		fmt.Fprint(w, "invalid nip")
		return
	}

	sess, err := s.as.NewSession(acc)
	w.Header().Add("SessionID", sess.ID.String())
	return
}

func (s *Server) getBalance(w http.ResponseWriter, r *http.Request) {
	sessItf := r.Context().Value(SessionKeyCtx)
	if sessItf == nil {
		panic("Session must not be nil if authenticated.")
	}

	sess := sessItf.(*Session)
	balance, err := s.db.Balance(sess.Account)
	if err != nil {
		log.Error().Err(err).Int("account_id", int(sess.Account)).Msg("failed to get balance")
	}

	fmt.Fprintf(w, "%d", balance)
}

func (s *Server) doDeposit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(405)
		fmt.Fprint(w, "not allowed")
		return
	}

	sessItf := r.Context().Value(SessionKeyCtx)
	if sessItf == nil {
		panic("Session must not be nil if authenticated.")
	}

	sess := sessItf.(*Session)

	depAmount := int64(-1)
	dec := json.NewDecoder(r.Body)
	err := dec.Decode(&depAmount)
	if err != nil {
		log.Error().Err(err).Msg("failed to decode deposit amount")
	}

	err = s.db.DoTransaction(sess.Account, persistence.Transaction{
		Type:   persistence.Deposit,
		Amount: depAmount,
	})
	if err != nil {
		log.Error().Err(err).Msg("transaction failed")
		w.WriteHeader(500)
		fmt.Fprint(w, "failed to perform deposit")
		return
	}

	fmt.Fprint(w, "ok")
}

func (s *Server) doWithdrawal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(405)
		fmt.Fprint(w, "not allowed")
		return
	}

	sessItf := r.Context().Value(SessionKeyCtx)
	if sessItf == nil {
		panic("Session must not be nil if authenticated.")
	}

	sess := sessItf.(*Session)

	depAmount := int64(-1)
	dec := json.NewDecoder(r.Body)
	err := dec.Decode(&depAmount)
	if err != nil {
		log.Error().Err(err).Msg("failed to decode withdrawn amount")
	}

	err = s.db.DoTransaction(sess.Account, persistence.Transaction{
		Type:   persistence.Withdrawal,
		Amount: depAmount,
	})
	if err != nil {
		log.Error().Err(err).Msg("transaction failed")
		w.WriteHeader(500)
		fmt.Fprint(w, "failed to perform deposit")
		return
	}

	fmt.Fprint(w, "ok")
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}
