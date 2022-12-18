package mysql

import (
	"context"
	"github.com/995933447/easytask/internal/util/logger"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"sync"
)

type repoConnector struct {
	connDsn string
	conn *gorm.DB
	connMu sync.Mutex
}

func (c *repoConnector) mustGetConn(ctx context.Context) *gorm.DB {
	if conn, err := c.getConn(ctx); err != nil {
		panic(any(err))
	} else {
		return conn
	}
}

func (c *repoConnector) getConn(ctx context.Context) (*gorm.DB, error) {
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
		logger.MustGetRepoLogger().Error(ctx, err)
		return nil, err
	}
	return c.conn.WithContext(ctx), nil
}
