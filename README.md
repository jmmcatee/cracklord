# Cracklord #
Queue and resource system for cracking passwords

[![GoDoc](https://godoc.org/github.com/jmmcatee/cracklord?status.svg)](http://godoc.org/github.com/jmmcatee/cracklord)

### Server Setups ###
You are expected to have a working Go build environment with GOPATH setup

Get Cracklord

`go get github.com/jmmcatee/cracklord`

Build the server and resource server components

`go build github.com/jmmcatee/cracklord/src/server`

`go build github.com/jmmcatee/cracklord/common/resourceserver`

Navigate to the resource server directory and run it

`cd $GOPATH/src/github.com/jmmcatee/cracklord/common/resourceserver`

`./resourceserver.exe`

Now open another prompt and navigate to the Cracklord server and run it

`cd $GOPATH/src/github.com/jmmcatee/cracklord/src/server`

`./server.exe`
