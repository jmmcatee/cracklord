# Cracklord #
[![GoDoc](https://godoc.org/github.com/jmmcatee/cracklord?status.svg)](http://godoc.org/github.com/jmmcatee/cracklord)
[![Build Status](https://drone.io/github.com/jmmcatee/cracklord/status.png)](https://drone.io/github.com/jmmcatee/cracklord/latest)

### http://jmmcatee.github.io/cracklord/ ###

## What Is It? ##

CrackLord is a system designed to provide a scalable, pluggable, and distributed system for both password cracking as well as any other jobs needing lots of computing resources. Better said, CrackLord is a way to load balance the resources, such as CPU, GPU, Network, etc. from multiple hardware systems into a single queueing service across two primary services: the Resource and Queue.

## System Components ##
<img src="http://jmmcatee.github.io/cracklord/img/about.png" width=300/>

There are three primary components to CrackLord as shown in the above image: 
* **Queue** - The Queue is a service that runs on a single system, providing an interface for users to submit, pause, resume, and delete jobs. These jobs are then processed and sent to available Resources to perform the actual work and handle the results.
* **Resource / Resource Managers** - Resources are the individual servers that are connected into the queue.  They are managed by a resource manager plugins.  These are code that allow various types of resources to be connected.  Managers can directly connect to physical resources you own, or use cloud services to spawn resources as necessary. 
* **Tools** - Tools are a set of plugins, configured on resources, that perform the underlying tasks such as running oclHashcat to crack passwords. Tools are written in the Go programming language and have a standard interface to make them easy to write or enhance.  They are wrappers of the various tools used that require great deals of resources, such as John, HashCat, etc. 

## Server Installation ##

We have a set of packages built for every release we make, if you'd like to just use that you can do it by simply following the instructions [here](http://jmmcatee.github.io/cracklord/#install).

If you'd like to get things build from source, it will first require you to have a working Go build environment with the GOPATH setup.  Additionally, you'll probably want Git and Mercurial setup to gather the various libraries and plugins that we've used in the code.  

1. First, you'll need to get cracklord itself.    
  `go get github.com/jmmcatee/cracklord`   

2. Next we need to get all of the dependencies downloaded for both the resource daemon and queue daemon.    
  `go get github.com/jmmcatee/cracklord/cmd/queued`   
  `go get github.com/jmmcatee/cracklord/cmd/resourced`   

3. Now we can actually build the queue daemon and resource daemon   
  `go build github.com/jmmcatee/cracklord/cmd/queued`   
  `go build github.com/jmmcatee/cracklord/cmd/resourced`   

4. Finally, we can run both the resource and queue daemons, which will both be in the cmd/queued and cmd/resourced directories.  You will also need to setup the various configuration files, information for those can be found in [our wiki](https://github.com/jmmcatee/cracklord/wiki). 

## Contributing ##
### Addons ###
Probably the easiest way to get involved is to write a new tool plugin.  If you have tools that you use as part of testing, research, or work and would like to get them integrated, you can very easily write a new tool and send us a pull request.  We'll make sure to get it integrated in as soon as possible.  In the plugins directory we have created an empty tool to provide some guidance and help. If you also have a neat way to interact with resources, you would also write a resource manager plugin, maybe for a cloud service that we don't support yet or some new way to do the work.  

Because of the way the Go language works, we have to compile all of the tools in, so if you do something you'd like to share please send us a pull request and we'll test it and get it out for everyone to use. 

### Scripts / GUI ###
We have a standard [API](https://github.com/jmmcatee/cracklord/wiki/API) that the queue daemon publishes out for access.  We went ahead and wrote a standard web GUI which also uses the same API.  That doesn't mean you couldn't make a better one!  We're also looking at writing a few scripts to automate common jobs in our workflow, if you end up making them send us links or a pull request and we'll make sure to find a home / give you a shout out!

### Documentation ###
We're working hard to try and keep the documentation up to date with everything we're doing, but there's always room for a how-to, tutorial, or example and we'd love any help you can provide on those.  Head on over to our [wiki](https://github.com/jmmcatee/cracklord/wiki) and see what needs fixing or adding!

### Bugs / Issues ###
Of course, there's nothing saying you can't work on the CrackLord queue and resource daemons themselves.  We have our [issues](https://github.com/jmmcatee/cracklord/issues) list and any help getting those fixed would be greatly appreciated. 
