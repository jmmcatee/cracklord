package api

import (
	"crypto/rand"
	"github.com/codegangsta/martini"
	"github.com/jmmcatee/cracklord/common"
	"github.com/martini-contrib/sessions"
)

func ServerAPI(auth common.Authenticator) {
	m := martini.Classic()

	// Get Token store
	ts := common.NewTokenStore()

	m.Map(ts)
	m.Map(auth)

	sessionAuth := make([]byte, 64)
	aeskey := make([]byte, 32)
	rand.Read(sessionAuth)
	rand.Read(aeskey)

	sessionStore := sessions.NewCookieStore(sessionAuth, aeskey)
	m.Use(sessions.Sessions("cracklord", sessionStore))

	m.Post("/login", APILogin)

	go m.Run()
}
