// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.20.0

package db

import ()

type Config struct {
	ID               int32
	AppName          string
	ClientIdentifier string
	PlexServerUrl    string
	PlexServerToken  string
}