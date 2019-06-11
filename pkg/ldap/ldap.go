package ldap

import (
	"crypto/tls"
	"fmt"
	ldapBase "gopkg.in/ldap.v2"
	"mirrors_status/internal/config"
	"mirrors_status/internal/log"
	"time"
)

type Client struct {
	conn     *ldapBase.Conn
	dn       string
	password string
	search   string
}

func NewLdapClient() (
	client *Client, err error) {
	config := configs.NewServerConfig().Ldap
	ldapBase.DefaultTimeout = 20 * time.Second
	conn, err := ldapBase.DialTLS("tcp", config.Server+":"+fmt.Sprint(config.Port),
		&tls.Config{InsecureSkipVerify: true})
	if err != nil {
		return
	}
	client = &Client{
		conn:     conn,
		dn:       config.Dn,
		password: config.Passwd,
		search:   config.USearch,
	}
	return
}

func (c *Client) prepare() (err error) {
	err = c.conn.Bind(c.dn, c.password)
	if err != nil {
		err = fmt.Errorf("LDAP search is not set properly")
	}
	return
}

func (c *Client) CheckUserPassword(username, password string) (err error) {
	err = c.prepare()
	if err != nil {
		return
	}
	req := ldapBase.NewSearchRequest(
		c.search, ldapBase.ScopeWholeSubtree, ldapBase.NeverDerefAliases,
		0, 0, false,
		fmt.Sprintf("(uid=%s)", ldapBase.EscapeFilter(username)),
		[]string{"dn"}, nil)
	resp, err := c.conn.Search(req)
	if err != nil {
		log.Errorf("search for username:%s found error:[%v]", username, err)
		return
	}
	if len(resp.Entries) != 1 {
		err = fmt.Errorf("ldap failed to match")
		return
	}
	userDn := resp.Entries[0].DN
	err = c.conn.Bind(userDn, password)
	if err != nil {
		err = fmt.Errorf("invalid username or password")
		return
	}
	return
}
