package server

import (
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/poisonbian/hersql/config"
	"github.com/poisonbian/hersql/ntunnel"
	"github.com/dolthub/vitess/go/mysql"
	"github.com/dolthub/vitess/go/sqltypes"
	"github.com/dolthub/vitess/go/vt/proto/query"
	"go.uber.org/zap"
)

var useRe = regexp.MustCompile("^USE `(.+)`$")
var dsnRe = regexp.MustCompile(`^<(.+)>$`)

type Handler struct {
	readTimeout time.Duration
	logger      *zap.SugaredLogger
	qer         *ntunnel.Querier
	sm          *SessionManager
	conf		*config.Conf
}

func NewHandler(readTimeout time.Duration, conf *config.Conf, logger *zap.SugaredLogger, sm *SessionManager) *Handler {
	qer := ntunnel.NewQuerier(conf.DBInfo.NTunnelUrl, logger)
	return &Handler{
		readTimeout: readTimeout,
		logger:      logger,
		qer:         qer,
		sm:          sm,
		conf:		 conf,
	}
}

func (h *Handler) NewConnection(c *mysql.Conn) {
	h.logger.Infof("NewConnection:[%s], id:[%d]", c.Conn.RemoteAddr().String(), c.ID)
	h.sm.NewSession(c)
}

func (h *Handler) ComInitDB(c *mysql.Conn, schemaName string) error {
	h.logger.Infof("ComInitDB:[%s], id:[%d] schemaName:[%s]", c.Conn.RemoteAddr().String(), c.ID, schemaName)
	return nil
}

func (h *Handler) ComPrepare(c *mysql.Conn, query string) ([]*query.Field, error) {
	h.logger.Infof("ComPrepare:[%s], id:[%d]", c.Conn.RemoteAddr().String(), c.ID)
	return nil, errors.New("transactions are not supported")
}

func (h *Handler) ComStmtExecute(c *mysql.Conn, prepare *mysql.PrepareData, callback func(*sqltypes.Result) error) error {
	h.logger.Infof("ComStmtExecute:[%s], id:[%d]", c.Conn.RemoteAddr().String(), c.ID)
	return errors.New("transactions are not supported")
}

func (h *Handler) ComResetConnection(c *mysql.Conn) {
	h.logger.Infof("ComResetConnection:[%s], id:[%d]", c.Conn.RemoteAddr().String(), c.ID)
}

func (h *Handler) ConnectionClosed(c *mysql.Conn) {
	h.logger.Infof("ConnectionClosed:[%s], id:[%d]", c.Conn.RemoteAddr().String(), c.ID)
	h.sm.DeleteSession(c)
}

func (h *Handler) ComQuery(
	c *mysql.Conn,
	query string,
	callback func(*sqltypes.Result) error,
) (err error) {
	defer func() {
		if err != nil {
			h.logger.Errorf("ComQuery:[%s], connection:[%s], id:[%d] err:[%s]", query, c.Conn.RemoteAddr().String(), c.ID, err.Error())
		} else {
			h.logger.Infof("ComQuery:[%s], connection:[%s], id:[%d] success", query, c.Conn.RemoteAddr().String(), c.ID)
		}
	}()

	dsn := h.sm.GetSession(c).GetDSN()
	if (dsn == nil) {
		str := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", 
			h.conf.DBInfo.User, h.conf.DBInfo.Password, 
			h.conf.DBInfo.Host, h.conf.DBInfo.Port,
			h.conf.DBInfo.DataBase)
		err = h.sm.GetSession(c).SetDSN(str)
		if (err != nil) {
			h.logger.Errorf("set dsn error: %s", err.Error())
		}
		h.logger.Infof("dsn is nil, init dsn: %s", str)
	}
	
	if useMatches := useRe.FindStringSubmatch(query); len(useMatches) == 2 {
		if dsnMatches := dsnRe.FindStringSubmatch(useMatches[1]); len(dsnMatches) == 2 {
			err = h.sm.GetSession(c).SetDSN(dsnMatches[1])
			if err != nil {
				return
			}
		} else {
			dsn := h.sm.GetSession(c).GetDSN()
			
			if dsn == nil {
				err = fmt.Errorf("no dsn specified before the query:[%s], you can try restarting the mysql client", query)
				return
			}

			dsn.SetDB(useMatches[1])
		}
		callback(new(sqltypes.Result))
	} else {
		dsn := h.sm.GetSession(c).GetDSN()
		if dsn == nil {
			err = fmt.Errorf("no dsn specified before the query:[%s], you can try restarting the mysql client", query)
			return
		}

		var result *sqltypes.Result
		result, err = h.qer.Query(query, h.sm.GetSession(c).GetDSN())
		if err != nil {
			return
		}

		err = callback(result)
	}

	return
}

func (h *Handler) WarningCount(c *mysql.Conn) uint16 {
	return 0
}
