package storage

import (
	"fmt"
	"testing"
)

func TestBuiildConnectionString(t *testing.T) {
	testCases := []struct {
		username string
		password string
		host     string
		port     string
		dbName   string
		expected string
	}{
		{"testuser1", "testpass1", "localhost", "5432", "plexAnilistSync", "postgres://testuser1:testpass1@localhost:5432/plexAnilistSync?sslmode=disable"},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(fmt.Sprintf("BuildConnectionString(%s, %s, %s, %s, %s)", tc.username, tc.password, tc.host, tc.port, tc.dbName), func(t *testing.T) {
			t.Parallel()

			result := buildConnectionString(tc.username, tc.password, tc.host, tc.port, tc.dbName)

			if result != tc.expected {
				t.Errorf("BuildConnectionString(%s, %s, %s, %s, %s) = %s; want %s", tc.username, tc.password, tc.host, tc.port, tc.dbName, result, tc.expected)
				fmt.Printf("result  : %v\n", result)
				fmt.Printf("expected: %v\n", tc.expected)
			}
		})
	}
}
