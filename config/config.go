package config

import (
	"errors"
	"io/ioutil"

	yaml "gopkg.in/yaml.v2"
)

type Conf struct {
	DBInfo     *DBInfo	`yaml:"db_info"`
	Server     *Server	`yaml:"server"`
	Log        *Log		`yaml:"log"`
}

type DBInfo struct {
	NTunnelUrl		string `yaml:"ntunnel_url"`
	Host 			string `yaml:"host"`
	Port			string `yaml:"port"`
	DataBase 		string `yaml:"database"`
	User			string `yaml:"user"`
	Password		string `yaml:"password"`
}

type Server struct {
	Protocol         string `yaml:"protocol"`
	Address          string `yaml:"address"`
	Version          string `yaml:"version"`
	ConnReadTimeout  uint64 `yaml:"conn_read_timeout"`
	ConnWriteTimeout uint64 `yaml:"conn_write_timeout"`
	MaxConnections   uint64 `yaml:"max_connections"`
	UserName         string `yaml:"user_name"`
	UserPassword     string `yaml:"user_password"`
}

type Log struct {
	InfoLogFilename  string `yaml:"info_log_filename"`
	ErrorLogFilename string `yaml:"error_log_filename"`
}

func Parse(filename string) (conf *Conf, err error) {
	fileData, err := ioutil.ReadFile(filename)
	if err != nil {
		return
	}

	conf = new(Conf)
	err = yaml.Unmarshal(fileData, conf)
	if err != nil {
		return
	}

	if conf.DBInfo.NTunnelUrl == "" {
		err = errors.New("please specify ntunnel_url in the configuration file")
		return
	}

	if conf.Log == nil {
		conf.Log = new(Log)
	}
	
	return
}
