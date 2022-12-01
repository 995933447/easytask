package mysql

import (
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"sync"
)

type repoConnector struct {
	connDsn string
	conn *gorm.DB
	connMu sync.Mutex
}

func (c *repoConnector) mustGetConn() *gorm.DB {
	if conn, err := c.getConn(); err != nil {
		panic(any(err))
	} else {
		return conn
	}
}

func (c *repoConnector) getConn() (*gorm.DB, error) {
	if c.conn != nil {
		return c.conn, nil
	}
	c.connMu.Lock()
	defer c.connMu.Unlock()
	if c.conn != nil {
		return c.conn, nil
	}
	var err error
	if c.conn, err = gorm.Open(mysql.Open(c.connDsn), &gorm.Config{}); err != nil {
		return nil, err
	}
	return c.conn, nil
}
