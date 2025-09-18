// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestConnectJDBC(t *testing.T) {
	for _, tt := range []struct {
		name     string
		expected string
		input    Connect
	}{
		{
			name: "zero", expected: "jdbc:postgresql:/",
		},

		// https://jdbc.postgresql.org/documentation/use/#connecting-to-the-database
		{
			name: "JDBC example: database", expected: "jdbc:postgresql:database",
			input: Connect{
				Database: "database",
			},
		}, {
			name: "JDBC example: host, database", expected: "jdbc:postgresql://host/database",
			input: Connect{
				HostPorts: []string{"host"},
				Database:  "database",
			},
		}, {
			name: "JDBC example: host", expected: "jdbc:postgresql://host/",
			input: Connect{
				HostPorts: []string{"host"},
			},
		}, {
			name: "JDBC example: host, port, database", expected: "jdbc:postgresql://host:port/database",
			input: Connect{
				HostPorts: []string{"host:port"},
				Database:  "database",
			},
		}, {
			name: "JDBC example: host, port", expected: "jdbc:postgresql://host:port/",
			input: Connect{
				HostPorts: []string{"host:port"},
			},
		}, {
			name: "JDBC example: parameters, but sorted", expected: "jdbc:postgresql://localhost/test?password=secret&ssl=true&user=fred",
			input: Connect{
				HostPorts: []string{"localhost"},
				Database:  "test",
				Parameters: map[string]string{
					"user":     "fred",
					"password": "secret",
					"ssl":      "true",
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.input.JDBC())
		})
	}
}

func TestConnectURI(t *testing.T) {
	for _, tt := range []struct {
		name     string
		expected string
		input    Connect
	}{
		{
			name: "zero", expected: "postgresql:",
		},

		// https://www.postgresql.org/docs/current/libpq-connect.html#LIBPQ-CONNSTRING-URIS
		{
			name: "libpq example: host", expected: "postgresql://localhost",
			input: Connect{
				HostPorts: []string{"localhost"},
			},
		}, {
			name: "libpq example: host, port", expected: "postgresql://localhost:5433",
			input: Connect{
				HostPorts: []string{"localhost:5433"},
			},
		}, {
			name: "libpq example: host, database", expected: "postgresql://localhost/mydb",
			input: Connect{
				HostPorts: []string{"localhost"},
				Database:  "mydb",
			},
		}, {
			name: "libpq example: two hosts, but sorted", expected: "postgresql://host1:123,host2:456/somedb?application_name=myapp&target_session_attrs=any",
			input: Connect{
				HostPorts: []string{"host1:123", "host2:456"},
				Database:  "somedb",
				Parameters: map[string]string{
					"target_session_attrs": "any",
					"application_name":     "myapp",
				},
			},
		}, {
			name: "libpq example: host parameters", expected: "postgresql:///mydb?host=localhost&port=5433",
			input: Connect{
				Database:   "mydb",
				Parameters: map[string]string{"host": "localhost", "port": "5433"},
			},
		}, {
			name: "libpq example: encoding", expected: "postgresql://localhost:5433/mydb?options=-c%20synchronous_commit%3Doff",
			input: Connect{
				HostPorts:  []string{"localhost:5433"},
				Database:   "mydb",
				Parameters: map[string]string{"options": "-c synchronous_commit=off"},
			},
		}, {
			name: "libpq example: IPv6", expected: "postgresql://[2001:db8::1234]/database",
			input: Connect{
				HostPorts: []string{"[2001:db8::1234]"},
				Database:  "database",
			},
		}, {
			name: "libpq example: Unix-domain socket parameter, but encoded", expected: "postgresql:///dbname?host=%2Fvar%2Flib%2Fpostgresql",
			input: Connect{
				Database:   "dbname",
				Parameters: map[string]string{"host": "/var/lib/postgresql"},
			},
		}, {
			name: "libpq example: Unix-domain socket host", expected: "postgresql://%2Fvar%2Flib%2Fpostgresql/dbname",
			input: Connect{
				HostPorts: []string{"/var/lib/postgresql"},
				Database:  "dbname",
			},
		},

		// Issue: CrunchyData/postgres-operator#4286
		{
			name: "password with special characters", expected: "postgresql://localhost?password=Lgrd%7DoKjF287_nUn6%2C%3Ds%3C%3AO%7C",
			input: Connect{
				HostPorts: []string{"localhost"},
				Password:  `Lgrd}oKjF287_nUn6,=s<:O|`,
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.input.URI())
		})
	}
}
