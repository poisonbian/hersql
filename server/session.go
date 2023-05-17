package server

import (
	"fmt"
	"sync"

	"hersql/ntunnel"
	"github.com/dolthub/vitess/go/mysql"
	gomysql "github.com/go-sql-driver/mysql"
)

type Session struct {
	mu   sync.RWMutex
	dsn  *ntunnel.DSN
	conn *mysql.Conn
}

func (s *Session) SetDSN(dsn string) (err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cfg, err := gomysql.ParseDSN(dsn)
	if err != nil {
		err = fmt.Errorf("SetDsn ParseDSN err:[%w]", err)
		return
	}

	s.dsn, err = ntunnel.NewDSN(cfg)
	if err != nil {
		return
	}

	return
}

func (s *Session) GetDSN() *ntunnel.DSN {
	return s.dsn
}

type SessionManager struct {
	mu       sync.Mutex
	sessions map[uint32]*Session
}

func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[uint32]*Session),
	}
}

func (sm *SessionManager) NewSession(conn *mysql.Conn) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.sessions[conn.ConnectionID] = &Session{
		conn: conn,
	}
	return
}

func (sm *SessionManager) GetSession(conn *mysql.Conn) *Session {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return sm.sessions[conn.ConnectionID]
}

func (sm *SessionManager) DeleteSession(conn *mysql.Conn) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.sessions, conn.ConnectionID)
	return
}
