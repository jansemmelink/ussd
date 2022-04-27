package pcm

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"bitbucket.org/vservices/utils/v4/errors"
	"bitbucket.org/vservices/utils/v4/logger"
	"github.com/gchaincl/sqlhooks"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

var (
	log = logger.NewLogger()
	db  *sqlx.DB
)

func init() {
	sql.Register("mysqlwithlog", sqlhooks.Wrap(&mysql.MySQLDriver{}, Hooks{}))
}

func Connect(c DatabaseConfig) error {
	if db != nil {
		return errors.Errorf("db already connected")
	}

	if err := c.Validate(); err != nil {
		return errors.Wrapf(err, "invalid database config")
	}

	//connect to the database to create the pool of connections
	connResultChan := make(chan connResult, 1)
	go func() {
		db, err := sqlx.Connect("mysqlwithlog", c.ConnectString())
		connResultChan <- connResult{
			db:  db,
			err: err,
		}
	}()

	//wait for connect result or timeout
	select {
	case connResult := <-connResultChan:
		if connResult.err != nil {
			return errors.Wrapf(connResult.err, "failed to connect to database %s on %s:%d", c.Database, c.Host, c.Port)
		}
		db = connResult.db
		db.SetMaxOpenConns(c.MaxConnOpen)
		db.SetMaxIdleConns(c.MaxConnIdle)
		return nil

	case <-time.After(time.Duration(c.MaxConnSeconds) * time.Second):
		return errors.Errorf("%d second timeout connecting to db %s on %s:%d", c.MaxConnSeconds, c.Database, c.Host, c.Port)
	} //select
}

func IntDefault(s string, def int) int {
	if i64, err := strconv.ParseInt(s, 10, 64); err != nil {
		return def
	} else {
		if int(i64) == 0 {
			return def
		}
		return int(i64)
	}
}

func StrDefault(s string, def string) string {
	if s != "" {
		return s
	}
	return def
}

type DatabaseConfig struct {
	Host           string `json:"host"`
	Port           int    `json:"port"`
	Username       string `json:"username"`
	Password       string `json:"password"`
	Database       string `json:"database"`
	MaxConnSeconds int    `json:"max_conn_seconds" doc:"Max nr of seconds to wait for db connection to be established"`
	MaxConnOpen    int    `json:"max_conn_open" doc:"Max nr of open connections in pool"`
	MaxConnIdle    int    `json:"max_conn_idle" doc:"Max nr of idle connections in pool"`
}

func (c *DatabaseConfig) Validate() error {
	if c.Host == "" {
		c.Host = "127.0.0.1"
	}
	if c.Port == 0 {
		c.Port = 3307
	}
	if c.Username == "" {
		return errors.Errorf("missing username")
	}
	if c.Password == "" {
		return errors.Errorf("missing password")
	}
	if c.Database == "" {
		return errors.Errorf("missing database name")
	}
	if c.MaxConnSeconds == 0 {
		c.MaxConnSeconds = 2
	}
	if c.MaxConnSeconds < 0 {
		return errors.Errorf("invalid max_conn_seconds:%d", c.MaxConnSeconds)
	}
	if c.MaxConnOpen == 0 {
		c.MaxConnOpen = 5
	}
	if c.MaxConnOpen < 0 {
		return errors.Errorf("invalid max_conn_open:%d", c.MaxConnOpen)
	}
	if c.MaxConnIdle == 0 {
		c.MaxConnIdle = 5
	}
	if c.MaxConnIdle < 0 {
		return errors.Errorf("invalid max_conn_idle:%d", c.MaxConnIdle)
	}
	return nil
} //DatabaseConfig.Validate()

func (c DatabaseConfig) ConnectString() string {
	return fmt.Sprintf("%s:%s@(%s:%d)/%s",
		c.Username,
		c.Password,
		c.Host,
		c.Port,
		c.Database)
}

type connResult struct {
	db  *sqlx.DB
	err error
}

var (
	compilesMutex      sync.Mutex
	compiledStatements = map[string]*sqlx.NamedStmt{}
)

//compile statements only once
func getCompiledStatement(query string) (*sqlx.NamedStmt, error) {
	compilesMutex.Lock()
	defer compilesMutex.Unlock()
	st, ok := compiledStatements[query]
	if ok {
		return st, nil //already compiled
	}
	st, err := db.PrepareNamed(query)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to prepare SQL statement")
	}
	compiledStatements[query] = st
	log.Infof("Compiled SQL (now %d): %s", len(compiledStatements), query)
	return st, nil
}

//return sql.ErrNoRows if not found
func NamedGet(rowPtr interface{}, query string, arg interface{}) (err error) {
	st, err := getCompiledStatement(query)
	if err != nil {
		return errors.Wrapf(err, "failed to prepare SQL statement")
	}
	err = st.Get(rowPtr, arg)
	if err != nil {
		return errors.Wrapf(err, "failed to get row")
	}
	return nil
}

//select a list of rows
func NamedSelect(list interface{}, query string, arg interface{}) (err error) {
	st, err := getCompiledStatement(query)
	if err != nil {
		return errors.Wrapf(err, "failed to prepare SQL statement")
	}
	log.Debugf("query: %s", query)
	log.Debugf("  arg: %v", arg)
	err = st.Select(list, arg)
	if err != nil {
		return errors.Wrapf(err, "failed to get list of rows")
	}
	return nil
}

// Hooks satisfies the sqlhook.Hooks interface
type Hooks struct{}

type HookBegin struct{}

// Before hook will print the query with it's args and return the context with the timestamp
func (h Hooks) Before(ctx context.Context, query string, args ...interface{}) (context.Context, error) {
	//log.Infof("SQL... %s (%d args=%+v)", query, len(args), args)
	return context.WithValue(ctx, HookBegin{}, time.Now()), nil
}

// After hook will get the timestamp registered on the Before hook and print the elapsed time
func (h Hooks) After(ctx context.Context, query string, args ...interface{}) (context.Context, error) {
	begin := ctx.Value(HookBegin{}).(time.Time)
	log.Infof("SQL (dur: %10.10s) %s (%d args=%+v)", time.Since(begin), query, len(args), args)
	return ctx, nil
}

func FilteredSelect(list interface{}, selectSQL string, filter map[string]interface{}, limit int) error {
	filterQuery := []string{}
	filterArgs := map[string]interface{}{}
	for n, v := range filter {
		log.Debugf("filter(%s)=\"%s\"", n, v)
		if s, ok := v.(string); ok && strings.HasPrefix(s, "*") {
			filterQuery = append(filterQuery, fmt.Sprintf("%s like %%:%s%%", n, n))
		} else {
			filterQuery = append(filterQuery, fmt.Sprintf("%s=:%s", n, n))
		}
		filterArgs[n] = v
	}

	query := selectSQL
	for i, f := range filterQuery {
		if i == 0 {
			query += " where " + f
		} else {
			query += " and " + f
		}
	}
	query += fmt.Sprintf(" limit %d", limit)
	return NamedSelect(list, query, filterArgs)
} //FilteredSelect()

func mapValues(z interface{}) map[string]interface{} {
	v := map[string]interface{}{}
	t := reflect.TypeOf(z)
	if t.Kind() != reflect.Struct {
		panic(errors.Errorf("not a struct (%T)", z))
	}
	for i := 0; i < t.NumField(); i++ {
		fv := reflect.ValueOf(z).Field(i)
		if fv.Kind() != reflect.Ptr || (fv.Kind() == reflect.Ptr && !fv.IsNil()) { //exclude nil values
			n := t.Field(i).Name
			if nn := t.Field(i).Tag.Get("json"); nn != "" {
				n = strings.SplitN(nn, ",", 2)[0]
			}
			if nn := t.Field(i).Tag.Get("db"); nn != "" {
				n = strings.SplitN(nn, ",", 2)[0]
			}
			if fv.Kind() != reflect.Ptr {
				v[n] = reflect.ValueOf(z).Field(i).Interface()
			} else {
				v[n] = reflect.ValueOf(z).Field(i).Elem().Interface()
			}
		}
	}
	log.Debugf("mapValues(%+v) -> (%+v)", z, v)
	return v
}
