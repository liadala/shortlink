package database

func (c *Client) WriteShortURL(url string, ip string) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	
	stmt, err := c.db.Prepare(`INSERT OR IGNORE INTO links (url, ip) VALUES (?, ?);`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(url, ip)
	if err != nil {
		return err
	}
	return nil
}
