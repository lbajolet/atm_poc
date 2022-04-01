package persistence

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog/log"
)

type DB struct {
	connection *sql.DB
}

// Account is the ID of the account
type Account int

// NewDB returns the instance of the database
func NewDB() (*DB, error) {
	db, err := sql.Open("sqlite3", "db")
	if err != nil {
		return nil, err
	}

	return &DB{
		db,
	}, nil
}

const auth_sql = "SELECT id FROM users WHERE pin = ?"

// Auth authenticates to the database and returns the Account linked to `pin'
func (d DB) Auth(pin string) (Account, error) {
	stmt, err := d.connection.Prepare(auth_sql)
	if err != nil {
		panic(fmt.Sprintf(
			"failed to build prepared statement, SQL error: %s",
			err,
		))
	}

	acc := Account(-1)

	res, err := stmt.Query(pin)
	if err != nil {
		log.Error().Err(err).Msg("query failed")
		return acc, err
	}

	if !res.Next() {
		return acc, fmt.Errorf("no such account")
	}

	err = res.Scan(&acc)
	if err != nil {
		log.Error().Err(err).Msg("scan failed")
		return acc, err
	}

	return acc, nil
}
