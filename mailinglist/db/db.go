package db

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/mattn/go-sqlite3"
)

type EmailEntry struct {
	Id          int64
	Email       string
	ConfirmedAt *time.Time
	Optout      bool
}

func TryCreate(db *sql.DB) {
	_, err := db.Exec(`
		CREATE TABLE emails (
			id INTEGER PRIMARY KEY,
			email TEXT UNIQUE,
			confirmed_at INTEGER,
			opt_out INTEGER,
		);
	`)

	if err != nil {
		if sqlError, ok := err.(sqlite3.Error); ok {
			// code 1 == "table already exists"
			if sqlError.Code != 1 {
				log.Fatal(sqlError)
			}
		} else {
			log.Fatal(err)
		}
	}
}

func emailEntryFromRow(row *sql.Rows) (*EmailEntry, error) {
	vars := EmailEntry{}

	err := row.Scan(
		&vars.Id,
		&vars.Email,
		&vars.ConfirmedAt,
		&vars.Optout,
	)

	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	return &vars, nil

}

func CreateEmail(db *sql.DB, email string) error {
	_, err := db.Exec(`
		INSERT INTO
		emails(email, confirmed_at, opt_out)
		VALUES(?, 0, false)
	`, email,
	)

	if err != nil {
		log.Println(err)
		return err
	}

	return nil

}

func GetEmail(db *sql.DB, email string) (*EmailEntry, error) {
	rows, err := db.Query(`
		SELECT *
		FROM emails
		WHERE email = ?
	`, email,
	)

	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	defer db.Close()

	// iterate in each row
	for rows.Next() {
		emailEntryFromRow(rows)
	}

	return nil, nil
}

func EmailUpdate(db *sql.DB, email string) error {

	values, err := GetEmail(db, email)

	if err != nil {
		fmt.Println(err)
		return err
	}

	t := values.ConfirmedAt.Unix()

	_, err = db.Exec(`
		INSERT INTO 
		emails(email, confirmed_at, opt_out)
		VALUES(?, ?, ?)
		ON CONFLICT(email) DO UPDATE SET
			confirmed_at=?
			opt_out=?
	`, values.Email, t, values.Optout, t, values.Optout,
	)

	if err != nil {
		fmt.Println(err)
		return err
	}

	return nil
}

func DeleteEmail(db *sql.DB, email string) error {
	_, err := db.Exec(`
		UPDATE emails 
		SET opt_out=true 
		WHERE email=?
	`, email,
	)

	if err != nil {
		fmt.Println(err)
		return err
	}

	return nil

}

type GetEmailBatchQueryParams struct {
	Page  int
	Count int
}

func GetEmailBatch(db *sql.DB, params GetEmailBatchQueryParams) ([]EmailEntry, error) {
	var empty []EmailEntry

	rows, err := db.Query(`
		SELECT id, email, confirmed_at, opt_out
		FROM emails
		WHERE opt_out = false
		ORDER BY id ASC
		LIMIT ? OFFSET ?
	`, params.Count, (params.Page-1)*params.Count,
	)

	if err != nil {
		fmt.Println(err)
		return empty, err
	}

	defer rows.Close()

	emails := make([]EmailEntry, 0, params.Count)

	for rows.Next() {
		email, err := emailEntryFromRow(rows)
		if err != nil {
			return nil, err
		}
		emails = append(emails, *email)
	}

	return emails, nil

}
