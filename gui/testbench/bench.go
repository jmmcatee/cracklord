package testbench

import (
    "crypto/rand"
    "github.com/codegangsta/martini"
    "github.com/jmmcatee/cracklord/common"
    "github.com/jmmcatee/cracklord/api"
    "github.com/martini-contrib/sessions"
)

func main() {
    m := martini.Classic()

    sessionAuth := make([]byte, 64)
    aeskey := make([]byte, 32)
    rand.Read(sessionAuth)
    rand.Read(aeskey)

    sessionStore := sessions.NewCookieStore(sessionAuth, aeskey)
    m.Use(sessions.Sessions("cracklord", sessionStore))

    m.Post("/login", DoLogin)
    m.Post("/logout", DoLogout)
    m.Post("/system/cracktypes", DoSystemCrackTypes)
    m.Post("/system/tools", DoSystemTools)  
    m.Post("/system/shutdown", DoSystemShutdown)
    m.Post("/job", DoQueueList)
    m.Post("/job/order", DoQueueReorder)
    m.Post("/job/create", DoCrackForm)
    m.Post("/job/create/submit", DoCrackSubmit)
    m.Post("/job/read", DoJobRead)
    m.Post("/job/pause", DoJobPause)
    m.Post("/job/delete", DoJobDelete)
    m.Post("/resource", DoResourceList)
    m.Post("/resource/create", DoResourceNew)
    m.Post("/resource/pause", DoResourcePause)
    m.Post("/resource/delete", DoResourceDelete)

    go m.Run()
}

func DoLogin(req *http.Request) (int, string) {
    login := api.APILoginReq{}
    err := json.NewDecoder(req.Body).Decode(login)
    if err != nil {
        return 500, err
    }

    seed := make([]byte, 256)
    token := sha256.New()
    rand.Read(seed)

    apitoken := api.APILoginResp{}
    apitoken.Token = base64.StdEncoding.EncodeToString(token.Sum(seed))
    resp, err := json.Marshal(apitoken)
    if err != nil {
        return 500, err
    }

    return 200, string(resp)   
}

func DoLogout(req *http.Request) (int, string) {
    logout := api.APILogoutReq{}
    err := json.NewDecoder(req.Body).Decode(login)
    if err!= nil {
        return 500, err
    }

    return 200, "{error: 0}"    
}
