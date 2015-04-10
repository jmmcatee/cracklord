package main

import (
	"errors"
	log "github.com/Sirupsen/logrus"
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

	log.Debug("INI authentication setup")
}

func (a *INIAuth) Login(user, pass string) (User, error) {
	// Lookup the user
	p, ok := a.UserPass[user]
	if !ok {
		// No user found so return an error
		log.WithField("user", user).Error("User not found.")
		return User{}, errors.New("User not found.")
	}

	if p != pass {
		log.WithField("user", user).Error("Bad password.")
		return User{}, errors.New("Bad password")
	}

	// Build the user object
	var u = User{}
	u.Username = user

	// Apply correct group
	group, ok := a.UserMap[user]
	if !ok {
		log.WithField("user", user).Error("No user group set.")
		return User{}, errors.New("No group set")
	}

	u.Groups = append(u.Groups, group)
	u.LogOnTime = time.Now()

	log.WithFields(log.Fields{
		"user": u.Username, 
		"groups": u.Groups,
		"logontime": u.LogOnTime,
	}).Info("User successfully logged in.")

	return u, nil
}
