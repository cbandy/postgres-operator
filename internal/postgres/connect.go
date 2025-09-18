// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"net/url"
	"strings"
)

// Connect holds the components of a Postgres connection string.
//
// https://www.postgresql.org/docs/current/libpq-connect.html#LIBPQ-CONNSTRING
type Connect struct {
	User       string
	Password   string
	HostPorts  []string
	Database   string
	Parameters map[string]string
}

// JDBC returns c as a [JDBC] connection URI.
//
// [JDBC]: https://jdbc.postgresql.org/documentation/use#connecting-to-the-database
func (c *Connect) JDBC() string {
	database := max("/", c.Database)
	out := c.url()

	if len(out.Host) == 0 {
		out.Path = ""
		return "jdbc:" + out.String() + database
	}

	out.Path = database
	return "jdbc:" + out.String()
}

// URI returns c as a [libpq] connection URI, which slightly more constrained than [RFC 3986].
//
// [libpq]: https://www.postgresql.org/docs/current/libpq-connect.html#LIBPQ-CONNSTRING-URIS
// [RFC 3986]: https://datatracker.ietf.org/doc/rfc3986
func (c *Connect) URI() string {
	out := c.url()

	if len(out.Host) == 0 && len(c.Database) > 0 {
		out.Path = "/" + c.Database
	}

	return out.String()
}

// url returns a copy of c as a [url.URL].
func (c *Connect) url() url.URL {
	// Add user and password as query parameters so the URI authority is exclusively hostnames.
	// [RFC 3986] allows comma U+002C there which can confuse clients.
	//
	// JDBC: https://jdbc.postgresql.org/documentation/use#connection-parameters
	// libpq: https://github.com/postgres/postgres/blob/-/src/interfaces/libpq/fe-connect.c
	query := url.Values{}
	for k, v := range c.Parameters {
		query.Set(k, v)
	}
	if len(c.User) > 0 {
		query.Set("user", c.User)
	}
	if len(c.Password) > 0 {
		query.Set("password", c.Password)
	}

	return url.URL{
		Scheme: "postgresql",
		Path:   c.Database,

		// Hosts are left in the order they are defined.
		Host: strings.Join(c.HostPorts, ","),

		// The following produces a sorted string.
		RawQuery: strings.ReplaceAll(query.Encode(), "+", "%20"),
	}
}
