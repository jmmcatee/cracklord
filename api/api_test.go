package api

import (
	"bytes"
	"github.com/jmmcatee/cracklord/common"
	"io"
	"net/http"
	"testing"
	"time"
)

type emptyAuthenticator struct{}

func (t emptyAuthenticator) Login(user, pass string) (common.User, error) {
	u := common.User{}
	u.Username = "Tester"
	u.Groups = []string{"Standard"}
	u.LogOnTime = time.Now()

	return u, nil
}

func TestLogin(t *testing.T) {
	auth := emptyAuthenticator{}
	ServerAPI(auth)

	body := bytes.NewBuffer([]byte("{username:test,password:testing}"))
	resp, err := http.Post("http://localhost:3000/login", "application/json", body)
	if err != nil {
		t.Fatal(err.Error())
	}

	response := bytes.NewBuffer([]byte{})
	io.Copy(response, resp.Body)

	println("Output: " + response.String())
}
