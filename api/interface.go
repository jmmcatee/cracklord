package api

import (
	"time"
)

/*
 * Structure used to represent a user logged into the API.
 */
type User struct {
	Username  string
	Groups    []string
	Token     string
	LogOnTime time.Time
}

/*
 * This interface is used to allow multiple different types of authenticator
 * mechanisms to be used. Given a username and password it should return User
 * structure if the login was successful and an error if not. The Username,
 * Groups, Email, and LogOnTime should be populated by the Authenticator. Token will
 * be taken care of by the API package itself. It will overide any value
 * provided by default.
 */
type Authenticator interface {
	Login(user, pass string) (User, error)
}
