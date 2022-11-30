package database

import (
	"database/sql"
	"fmt"
)

func (c *Client) GetURL(id string) (string, error) {
	var url string
	row := c.db.QueryRow(`SELECT url FROM links WHERE short = ?;`, id)
	switch err := row.Scan(&url); err {
	case sql.ErrNoRows:
		return "", fmt.Errorf("no rows found")
	case nil:
		return url,nil
	default:
		return "", fmt.Errorf("unknown error")
	}
}
