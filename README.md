# Cracklord #
Queue and resource system for cracking passwords

### Server Setups ###
You are expected to have a working Go build environment with GOPATH setup

`go get github.com/jmmcatee/cracklord`

`go install github.com/jmmcatee/cracklord/server`

Copy the `public` folder to `$GOPATH/bin`.

Create a INI file in the `$GOPATH/bin` that looks like the follow:

```
[Authentication]
type=INI
adminuser=admin
adminpass=password
standarduser=standard
standardpass=password
readonlyuser=readonly
readonlypass=readonly

#type=ActiveDirectory
#realm=example.lcl
#ReadOnlyGroup="Domain Users"
#StandardGroup="Cracking Users"
#AdminGroup="Domain Admins"
```
