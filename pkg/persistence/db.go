package persistence

import (
	"context"
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

	defer stmt.Close()

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

const balanceQuery = "SELECT balance FROM users WHERE id = ?"

// Balance gets the current balance for the account
func (d DB) Balance(acc Account) (int64, error) {
	stmt, err := d.connection.Prepare(balanceQuery)
	if err != nil {
		panic(fmt.Sprintf(
			"failed to build prepared statement, SQL error: %s",
			err,
		))
	}

	defer stmt.Close()

	res, err := stmt.Query(acc)
	if err != nil {
		log.Error().Err(err).Msg("query failed")
		return -1, err
	}

	if !res.Next() {
		log.Error().Msg("empty rowset")
		return -1, fmt.Errorf("no balance available for account")
	}

	balance := int64(-1)
	err = res.Scan(&balance)
	if err != nil {
		log.Error().Err(err).Msg("scan failed")
		return -1, err
	}

	return balance, nil
}

// TransactionType determines how the funds of an account will change
type TransactionType int

const (
	// Error is the default value of the Transaction type
	//
	// It is only defined so the default value for a Transaction will not
	// provoke misbehaviours
	Error TransactionType = iota
	// Deposit increases the amount of cash of an account
	Deposit
	// Withdrawal decreases the amount of cash on an account
	Withdrawal
)

// Transaction is a financial movement for an account
type Transaction struct {
	Type   TransactionType
	Amount int64
}

func (tx Transaction) getAmount() int64 {
	switch tx.Type {
	case Deposit:
		return tx.Amount
	case Withdrawal:
		return -tx.Amount
	}

	panic("invalid transaction type")
}

const balanceUpdateQuery = "UPDATE users SET balance = (SELECT balance FROM users WHERE id = ?) + ? WHERE id = ?"

const transactionInsertQuery = "INSERT INTO transactions(amount, user) VALUES(?, ?)"

func (d DB) DoTransaction(acc Account, tx Transaction) error {
	dbTx, err := d.connection.BeginTx(context.Background(), &sql.TxOptions{})
	if err != nil {
		log.Error().Err(err).Msg("failed to build DB transaction")
		return err
	}

	bup, err := dbTx.Prepare(balanceUpdateQuery)
	if err != nil {
		panic(fmt.Sprintf(
			"failed to build prepared statement, SQL error: %s",
			err,
		))
	}
	log.Info().Msg("Done preparing update")

	_, err = bup.Exec(acc, tx.getAmount(), acc)
	if err != nil {
		log.Error().Err(err).Int("account_id", int(acc)).Msg("failed to update balance")
		return dbTx.Rollback()
	}

	bup.Close()

	log.Info().Msg("Done update")

	txIns, err := dbTx.Prepare(transactionInsertQuery)
	if err != nil {
		panic(fmt.Sprintf(
			"failed to build prepared statement, SQL error: %s",
			err,
		))
	}
	log.Info().Msg("Done preparing insert query")

	_, err = txIns.Exec(acc, tx.getAmount())
	if err != nil {
		log.Error().Err(err).Int("account_id", int(acc)).Msg("failed to insert transaction")
		return dbTx.Rollback()
	}

	log.Info().Msg("Done insert query")

	txIns.Close()

	return dbTx.Commit()
}
