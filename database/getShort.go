package database

import (
	"database/sql"
	"fmt"
)

func (c *Client) GetShort(domain string) (string, error) {
	var short string
	row := c.db.QueryRow(`SELECT short FROM links WHERE url = ?;`, domain)
	switch err := row.Scan(&short); err {
	case sql.ErrNoRows:
		return "", fmt.Errorf("no rows found")
	case nil:
		return short, nil
	default:
		return "", fmt.Errorf("unknown error")
	}
}
