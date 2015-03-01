package main

import (
	"time"
)

/*
 * The available groups are as follows
 * - Read-Only: This group can view the current cracks and all outputs,
 *   but cannot create a job.
 * - Standard User: This group can create, view, and stop jobs, but has no
 *   access to add, remove, or pause resource or the queue itself.
 * - Administrator: These users can do any function provided by the API.
 */
const (
	ReadOnly      = "Read-Only"
	StandardUser  = "Standard User"
	Administrator = "Administrator"
)

// Value in minutes
var SessionExpiration = 30 * time.Minute

/*
 * Structure used to represent a user logged into the API.
 */
type User struct {
	Username  string
	Groups    []string
	LogOnTime time.Time
	Timeout   time.Time
}

/*
 * This interface is used to allow multiple different types of authenticator
 * mechanisms to be used. Given a username and password it should return User
 * structure if the login was successful and an error if not. The Username,
 * Groups, Email, and LogOnTime should be populated by the Authenticator. Token will
 * be taken care of by the API package itself. It will overide any value
 * provided by default. Authenticators must be thread safe.
 */
type Authenticator interface {
	Setup(mapping map[string]string)
	Login(user, pass string) (User, error)
}

/*
 * The token store saves the valid tokens and the time they expire. The 30
 * minute timer is renewed after every successful check.
 */
type TokenStore struct {
	store map[string]*User
	sync.Mutex
}

func NewTokenStore() TokenStore {
	return TokenStore{
		store: map[string]*User{},
	}
}

func (t *TokenStore) AddToken(token string, user User) {
	t.Lock()
	defer t.Unlock()

	t.store[token] = &user
	t.store[token].Timeout = time.Now().Add(30 * time.Minute)
}

func (t *TokenStore) RemoveToken(token string) {
	t.Lock()
	defer t.Unlock()

	delete(t.store, token)
}

func (t *TokenStore) CheckToken(token string) bool {
	t.Lock()
	defer t.Unlock()

	if user, ok := t.store[token]; ok {
		// Check that this ticket hasn't timed out
		if 0 > user.Timeout.Sub(time.Now()) {
			// Token has expired so we should return false and remove the token
			delete(t.store, token)
			return false
		}

		// Token exists and has not timed out so return true and reset time
		t.store[token].Timeout = time.Now().Add(30 * time.Minute)
		return true
	}

	// Token did not exist to return false
	return false
}

func (t *TokenStore) GetUser(token string) (User, error) {
	t.Lock()
	defer t.Unlock()

	// Check for valid token
	if user, ok := t.store[token]; ok {
		// return the user we just got
		return *user, nil
	}

	return User{}, errors.New("Invalid Token")
}
