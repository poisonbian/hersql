package ntunnel

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"io"
	"io/ioutil"
	"bytes"
	
	"github.com/dolthub/vitess/go/sqltypes"
	"github.com/go-sql-driver/mysql"
	"go.uber.org/zap"
)

type DSN struct {
	Host     string
	Port     string
	Login    string
	Password string
	DB       string
}

func NewDSN(cfg *mysql.Config) (*DSN, error) {
	addrs := strings.Split(cfg.Addr, ":")
	if len(addrs) != 2 {
		return nil, errors.New("the addr must be in host:port format")
	}

	port, err := strconv.Atoi(addrs[1])
	if err != nil {
		return nil, fmt.Errorf("port:[%s] parse err:[%w]", addrs[1], err)
	}

	if port < 0 || port > math.MaxUint16 {
		return nil, fmt.Errorf("invalid port:[%d]", port)
	}

	return &DSN{
		Host:     addrs[0],
		Port:     addrs[1],
		Login:    cfg.User,
		Password: cfg.Passwd,
		DB:       cfg.DBName,
	}, nil
}

func (dsn *DSN) SetDB(db string) {
	dsn.DB = db
}

type Querier struct {
	ntunnelUrl string
	logger      *zap.SugaredLogger
}

func NewQuerier(ntunnelUrl string, logger *zap.SugaredLogger) *Querier {
	return &Querier{
		ntunnelUrl: ntunnelUrl,
		logger: logger,
	}
}

func (qer *Querier) Query(query string, dsn *DSN) (result *sqltypes.Result, err error) {
	params := url.Values{}
	params.Set("actn", "Q")
	params.Set("q[]", query)
	params.Set("host", dsn.Host)
	params.Set("port", dsn.Port)
	params.Set("login", dsn.Login)
	params.Set("password", dsn.Password)
	params.Set("db", dsn.DB)
	
	body := strings.NewReader(params.Encode())
	req, _ := http.NewRequest("POST", qer.ntunnelUrl, body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	httpClient := &http.Client{
		Timeout: 20 * time.Second,
	}
	
	var resp *http.Response
	var buf bytes.Buffer

	resp, _ = httpClient.Do(req)
	tee := io.TeeReader(resp.Body, &buf)
	
	body_str, _ := ioutil.ReadAll(tee)
//	qer.logger.Infof(string(body_str))
	qer.logger.Infof("length: %d", len(body_str))
	
	result, err = NewParser(&buf, qer.logger).Parse()

	if err != nil {
		qer.logger.Errorf("response parse error!")
		return
	}

	return
}
