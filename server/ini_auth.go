package main

import (
	"errors"
	"time"
)

// INI Auth structure for implementing the basic authenticator
type INIAuth struct {
	UserPass map[string]string
	UserMap  map[string]string
}

// The Setup function is used to provide usernames and passwords and a mapping
// of users to the three group types in Cracklord
func (a *INIAuth) Setup(userpass map[string]string, usermap map[string]string) {
	a.UserPass = userpass
	a.UserMap = usermap
}

func (a *INIAuth) Login(user, pass string) (User, error) {
	// Lookup the user
	p, ok := a.UserPass[user]
	if !ok {
		// No user found so return an error
		return User{}, errors.New("User not found.")
	}

	if p != pass {
		return User{}, errors.New("Bad password")
	}

	// Build the user object
	var u = User{}
	u.Username = user

	// Apply correct group
	group, ok := a.UserMap[user]
	if !ok {
		return User{}, errors.New("No group set")
	}

	u.Groups = append(u.Groups, group)
	u.LogOnTime = time.Now()

	return u, nil
}
