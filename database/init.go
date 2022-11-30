package database

import (
	"database/sql"
	"log"
	"sync"

	_ "modernc.org/sqlite"
)

type Client struct {
	db   *sql.DB
	lock sync.Mutex
}

func New() *Client {
	var (
		err error
		c   *Client = &Client{}
	)
	c.lock.Lock()
	defer c.lock.Unlock()

	c.db, err = sql.Open("sqlite", "database.db")
	if err != nil {
		log.Fatal(err)
	}

	_, err = c.db.Exec(`PRAGMA journal_mode=WAL;`)
	if err != nil {
		log.Fatal(err)
	}

	_, err = c.db.Exec(`CREATE TABLE IF NOT EXISTS "links" (
		"id"	INTEGER NOT NULL,
		"short"	TEXT DEFAULT (hex(randomblob(6))) UNIQUE,
		"url"	TEXT UNIQUE,
		"ip"	TEXT,
		"timestamp"	INTEGER DEFAULT (CAST(strftime('%s', 'now') AS INTEGER)),
		PRIMARY KEY("id" AUTOINCREMENT)
	);`)
	if err != nil {
		log.Fatal(err)
	}

	return c
}
