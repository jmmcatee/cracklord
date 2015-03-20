# Overview

The following documentation exists to provide documentation for the RESTful API used to communicate with the queue.  The API is divided into the following key areas: 

* Users / Credentials
  * [Login](#user-login)
  * [Logout](#user-logout)
* Tools
  * [List](#tool-list)
  * [Read](#tool-read)
* Jobs
  * [List](#job-list)
  * [Create](#job-create)
  * [Read](#job-read)
  * [Update](#job-update)
  * [Delete](#job-delete)
* Resources
  * [List](#resource-list)
  * [Create](#resource-create)
  * [Read](#resource-read)
  * [Update](#resource-update)
  * [Delete](#resource-delete)
* Queue
  * [Update](#queue-update)

## Terminology
- Resource – Individual servers / systems running that control tools and resources for the Queue Manager. 
- Tool – An external command or internal process that uses a specific type of resource, takes in a group of arguments and hashes, and attempts to crack these hashes. 
- Queue – The master queue that all jobs: running, paused, and stopped are kept in.  This is the master controller for the entire envrionment.
- Job – An individual job within the queue that is passed to individual Resource Clients 

## Notes
The API is hosted on the Queue at https://<HOST>/api and must be accessed over HTTPS.  As noted below, all requests must be authenticated utilizing the token provided through the login resource as described below.  In the case of GET requests, parameters will be expected within the query string.  In the case of POST requests, properties should be submitted as a JSON hash (don't forget to set Content-Type: application/json)! 

For all requests, a response code and message is provided whether it is successful or failed.  For additional information, see [[API-Status-Codes]] on the specific codes returned.  

## Object Status
The various types of objects (jobs, queues, and resources) have a set of states that are used to track their status.  The following lists provide a reference for the options and what they mean.

### Queue
+ Empty - The queue is running, but there is nothing in it currently.
+ Running - The queue is running and working on jobs.
+ Paused - The entire queue has been paused.
+ Exhausted - The queue finished all of the jobs in the stack.

### Job
+ Created - The job is now within the queue and will be processed as soon as possible.
+ Running - The job has been assigned to a resource and is being processed right now.
+ Paused - The job was paused, however the relevant state has been saved in memory on the resource.
+ Done - The job was successfully completed.
+ Failed - There was an error and the job could not be completed.
+ Quit - The job was stopped prematurely by a user and cannot be restarted.

### Resource
+ Running - The resource is accepting jobs and processing them.
+ Paused - The resource is not accepting jobs.

# API Documentation
## Users / Credentials
### Login <a name="user-login"></a>

__Resource Name:__ /login  

Allows the user to send a username and password and provides them with an authentication token to utilize for all further communications with the server.  

__Arguments:__  
+ username: [string]  - A username configured in the system
+ password: [string]  - The user's password  

__Return Value__  
+ status: [int] - The return code for our function, see [[API-Status-Codes]].
+ message: [string] - A message based on the return code.
+ token: [string]  - Provided by server to authenticate all future system activities.   
+ role: [string] - The role the user is operating as.  Will be one of "read-only", "standard user", or "administrator"

__Example Request__

```javascript
POST /api/login

{
  "username": "jdoe",
  "password": "P@ssw0rdExample",
}
```

__Example Return__  

```javascript
{  
  "status": 200,
  "message": "Login successful",
  "token": "fa2074ff31c20348bd6da41cd75fe3f4cd120fff6362386fb5b5dd367e08ca2f",
  "role": "Administrator"
}  
```

### Logout <a name="user-logout"></a>
__Resource Name:__ /logout  

This function allows a user to deactivate their token and log out from the system.  

__Arguments:__  

+ token: [string] -  User authentication token. 

__Return Value__  

+ status: [int] - The return code for our function, see [[API-Status-Codes]].
+ message: [string] - A message based on the return code.

__Example Request__

```javascript
GET /api/logout?token=fa2074ff31c20348bd6da41cd75fe3f4cd120fff6362386fb5b5dd367e08ca2f
```

__Example Return__  

```javascript
{  
  "status": 200,
  "message": "OK",
}  
```

## Tools 
### List <a name="tool-list"></a>
__Resource Name:__ GET /tools/

This will return a list of all tools configured within the system that could be used.

__Arguments:__  

+ token: [string] -  User authentication token. 

__Return Value__  

+ status: [int] - The return code for our function, see [[API-Status-Codes]].
+ message: [string] - A message based on the return code.
+ tools: [array] - The returned value is a JSON array of the names of all available tools.

__Example Request__

```javascript
GET /api/tools?token=fa2074ff31c20348bd6da41cd75fe3f4cd120fff6362386fb5b5dd367e08ca2f
```

__Example Return__  

```javascript
{
  "status": 200,
  "message": "OK",
  "tools": [
    {
      "toolid":"63ee8045-966f-449e-9839-58e7e0586f3c",
      "name":"Hashcat",
      "version":"1.3.3"
    },
    {
      "toolid":"8d660ce9-f15d-40a3-a997-a4e8867cb802",
      "name":"John the Ripper",
      "version":"1.7.9"
    },
    {
      "toolid":"1cee8439-7f22-457c-84b8-5a8b04414090",
      "name":"John the Ripper",
      "version":"1.8.0"
    }
  ]
}
```

### Read <a name="tool-list"></a>
__Resource Name:__ GET /tools/:id

This will read the information about an individual tool, primarily the form that will be used to create a job using the tool.

__Arguments:__  

+ token: [string] - User authentication token. 
+ id: [string] - UUID of the tool

__Return Value__  

+ status: [int] - The return code for our function, see [[API-Status-Codes]].
+ message: [string] - A message based on the return code.
+ resources: [array] - Array of strings containing the IDs of resources where this tool is configured.
+ form [array] - An associative array of data used to create the form for the tool.  See http://textalk.github.io/angular-schema-form/ for additional information on schema.
+ schema [array] - An associative array of the schema information that will be used to validate the form.  Validation also occurs on the server side, but should be included on the user side.  See http://textalk.github.io/angular-schema-form/ for additional information.

__Example Request__

```javascript
GET /api/tools/63ee8045-966f-449e-9839-58e7e0586f3c?token=fa2074ff31c20348bd6da41cd75fe3f4cd120fff6362386fb5b5dd367e08ca2f
```

__Example Return__  

```javascript
{
  "tool": {
    "id":"8d660ce9-f15d-40a3-a997-a4e8867cb802",
    "name":"John the Ripper",
    "version":"1.7.9",
    "form": [
      "algorithm",
      "dictionary",
      "rules",
      {
        "key":"hashes",
        "type":"textarea",
        "placeholder":"Hashes go here!"
      }
    ],
    "schema": {
      "type":"object",
      "properties":{
        "algorithm":{
          "title":"Algorithm",
          "type":"string",
          "enum":[
            "NTLM",
            "NTLMv2",
            "ms-cache",
            "ms-cache v2",
            "SQL 2005",
            "SQL 2008",
            "MD5"
          ]
        },
        "dictionary":{
          "title":"Dictionary",
          "type":"string",
          "enum":[
            "crackstation",
            "crackstation-human-only",
            "m3g9tr0n",
            "words-english"
          ]
        },
        "rules":{
          "title":"Rule File",
          "type":"string",
          "enum":[
            "d3ad0ne",
            "t0xic"
          ]
        },
        "hashes":{
          "title":"Hashes",
          "type":"string",
          "description":"Note: Use the file format as required by hashcat"
        }
      }
    },
    "status":200,
    "message":"OK"
  }
}
```

## Job

### List <a name="job-list"></a>
__Resource Name:__  GET /jobs

This will return a list of all jobs in the queue with some basic statistics about each job.

__Arguments:__  

+ token: [string] -  User authentication token. 

__Return Value__  

+ status: [int] - The return code for our function, see [[API-Status-Codes]].
+ message: [string] - A message based on the return code.
+ jobs: [array] - The returned value is a JSON array for all of the jobs in the queue with each item containing the following:
  + jobid: [string] – ID of the Job
  + name: [string] – Name of the job
  + status: [string] - The status of the job (running, paused, stopped, none)
  + resourceid: [string]  –  Resource the job is running on
  + owner: [string] - The username of the creator of the job
  + starttime: [timestamp] – UNIX Timestamp for the start time of the Job
  + cracked: [int] – Number of hashes that have been cracked. 
  + total: [int] - Number of hashes that were submitted
  + progress: [int] – Percentage of job completion

__Example Request__

```javascript
GET /api/jobs?token=2lkj1325098ek12lg98231
```

__Example Return__  

```javascript
{
  "status": 200,
  "message": "OK",
  "jobs": [ 
    {
      "id":"72fd24ca-e529-4b38-b70d-2ad566de7e49",
      "name":"The Cheerful Shark Logistics Company",
      "status":"running",
      "resourceid":"1116814b-7c59-4b5d-87b6-fabaa5f594d1",
      "owner":"emperorcow",
      "starttime":1426621220,
      "crackedhashes":5,
      "totalhashes":800,
      "progress":0.68,
      "toolid":"1116814b-7c59-4b5d-87b6-fabaa5f594d1"
    },
    {
      "id":"786c4f68-1b7f-46e0-b5bd-75090d78b25c",
      "name":"The Deep Lime Builders Company",
      "status":"paused",
      "resourceid":"1116814b-7c59-4b5d-87b6-fabaa5f594d1",
      "owner":"emperorcow",
      "starttime":1423739455,
      "crackedhashes":102,
      "totalhashes":539,
      "progress":0.17,
      "toolid":"1116814b-7c59-4b5d-87b6-fabaa5f594d1"
    }
  ]
}
```

### Create <a name="job-create"></a>
__Resource Name:__  POST /jobs/

Create a Job to be added to the Queue.  Takes three static pieces of information, the user token, tool, and name.  The remaining item is a list of form information produced from the job/create/form function.

__Arguments:__  
+ token: [string] -  User authentication token. 
+ toolid: [string] - ID of the tool to be used
+ name: [string] - Name used to track the job
+ params: [string] - JSON of parameters from new job form

__Return Value__  

+ status: [int] - The return code for our function, see [[API-Status-Codes]].
+ message: [string] - A message based on the return code.
+ jobid: [string] - The UUID of the job that was created.

__Example Request__

```javascript
POST /api/jobs

{
  "token": "2ldljk120o89fgh31wlk12",
  "toolid":"8d660ce9-f15d-40a3-a997-a4e8867cb802",
  "name":"Crack for ABC",
  "params":{
    "algorithm":"ms-cache",
    "dictionary":"m3g9tr0n",
    "rules":"t0xic",
    "hashes":"Administrator:500:6A98EB0FB88A449CBE6FABFD825BCA61:D144986C6122B1B1654BA39932465528:::\nGuest:501:A0E150C75A17008EAAD3B435B51404EE:3D2B4DFAC512B7EF6188248B8E113CB9:::\nfakeuser:1000:24500AFA4E78B1C1944E2DF489A880E4:F47E4045A58ECEBD1F544168E050B1A9:::"
  }
}
```

__Example Return__  

```javascript
{  
  "status": 200,
  "message": "OK",
  "jobid": "628c1286-05b5-4643-b9d2-07d53fc0f36e"
}  
```

### Read <a name="job-read"></a>
__Resource Name:__  GET /jobs/:id

Get a detailed status on a specific job

__Arguments:__  
+ token: [string] -  User authentication token. 
+ jobid: [string] – ID of the Job

__Return Value__  

+ status: [int] - The return code for our function, see [[API-Status-Codes]].
+ message: [string] - A message based on the return code.
+ jobid: [string] - The UUID of the job.
+ name: [string] - Name of the selected job.
+ status: [string] - Status of the job.  Will be one of completed, running, paused, stopped, or queued
+ resourceid: [string] - The UUID of the resource the job is running on.
+ owner: [string] - Username of the person who started this job.
+ starttime: [timestamp] - Timestamp of when this job was startedj
+ cracked: [int] - How many tasks or hashes have we completed/cracked
+ total: [int] - Total number of tasks or hashes that were originally submitted.
+ progress: [int] - A number from 0 to 1 representing how far the job has completed.
+ performance: [array] - An array of key value pairs, with the key as the timestamp of the data and the value being a piece of data showing over time performance of the tool
+ performancetitle: [string] - The title to show above the graph of performance.
+ output: [array] - An array of key/value pairs for output from the tool.  Will be different for each tool.

__Example Request__

```javascript
GET /api/jobs/eeeb309a-00db-40a2-966e-504c39f853eb?token=2lkj1325098ek12lg98231
```

__Example Return__  

```javascript
{
  "status":200,
  "message":"OK",
  "job":{
    "id":"786c4f68-1b7f-46e0-b5bd-75090d78b25c",
    "name":"The Deep Lime Builders Company",
    "status":"paused",
    "resourceid":"1116814b-7c59-4b5d-87b6-fabaa5f594d1",
    "owner":"emperorcow",
    "starttime":1423739455,
    "crackedhashes":102,
    "totalhashes":539,
    "progress":0.17,
    "params":{
      "rules":"d3ad0ne",
      "dictionary":"m3g9tr0n",
      "algorithm":"NTLMv2"
    },
    "toolid":"1116814b-7c59-4b5d-87b6-fabaa5f594d1",
    "performancetitle":"Hashes Per Second",
    "performancedata":{
      "1426540600":817520286429,
      "1426840600":943571424640,
      "1427140600":170408695042,
      "1427440600":887381030944,
      "1427740600":281011148854,
      "1428040600":829705171956,
      "1428340600":387598519958,
      "1428640600":875715683354,
      "1428940600":504016472857,
      "1429240600":191412988977,
      "1429540600":910057657506,
      "1429840600":292800919333,
      "1430140600":608991357943,
      "1430440600":133411525899,
      "1430740600":690580034950,
      "1431040600":547304740030,
      "1431340600":742755575255,
      "1431640600":364057020833,
      "1431940600":675835662873,
      "1432240600":70984913208,
      "1432540600":692616662126,
      "1432840600":359597199587,
      "1433140600":556924477062,
      "1433440600":434569672536,
      "1433740600":764635553536,
      "1434040600":183403765827,
      "1434340600":781993316002,
      "1434640600":229866146520,
      "1434940600":399912130723,
      "1435240600":413416842757,
      "1435540600":400435206762,
      "1435840600":766845239145,
      "1436140600":243617788193,
      "1436440600":597963029446
    },
    "outputtitles": [
      "Hash",
      "Plaintext"
    ],
    "outputdata": [
      ["hash0","password0"],
      ["hash1","password1"],
      ["hash2","password2"],
      ["hash3","password3"],
      ["hash4","password4"],
      ["hash5","password5"],
      ["hash6","password6"]
    ]
  }
}
```

### Update <a name="job-update"></a>
__Resource Name:__  PUT /jobs/:id 

Update the status of a job, pausing, resuming, or shutting it down.

__Arguments:__  
+ token: [string] -  User authentication token. 
+ jobid: [string] – ID of the Job
+ action: [string] - What should we change this job to?  Should be pause, stop, or resume

__Return Value__  

+ status: [int] - The return code for our function, see [[API-Status-Codes]].
+ message: [string] - A message based on the return code.

__Example Request__

```javascript
PUT /api/jobs/72fd24ca-e529-4b38-b70d-2ad566de7e49

{
  "token": "2ldljk120o89fgh31wlk12",
  "id":"b762b17a-c324-4385-8629-a829e1bc4395",
  "name":"Beta Gecko Films",
  "status":"paused",
  "resourceid":"1116814b-7c59-4b5d-87b6-fabaa5f594d1",
  "owner":"emperorcow",
  "starttime":1423494683,
  "crackedhashes":0,
  "totalhashes":352,
  "progress":0,
  "toolid":"1116814b-7c59-4b5d-87b6-fabaa5f594d1",
  "expanded":false
}
```

__Example Return__  

```javascript
{  
  "status": 200,
  "message": "OK",
}  
```

### Delete<a name="job-delete"></a>
__Resource Name:__  DELETE /jobs/:id 

Delete a job from the queue.

__Arguments:__  

+ token: [string] -  User authentication token. 
+ jobid: [string] – ID of the Job
 
__Return Value__  

+ status: [int] - The return code for our function, see [[API-Status-Codes]].
+ message: [string] - A message based on the return code.

__Example Request__

```javascript
DELETE /api/jobs/72fd24ca-e529-4b38-b70d-2ad566de7e49

{
  "token": "2ldljk120o89fgh31wlk12",
}
```

__Example Return__  

```javascript
{  
  "status": 200,
  "message": "OK",
}  
```

## Resources

### List <a name="resource-list"></a>

__Resource Name:__  GET /resources

List all resources currently configured within the Queue

__Arguments:__  

+ token: [string] -  User authentication token. 
 
__Return Value__  

+ status: [int] - The return code for our function, see [[API-Status-Codes]].
+ message: [string] - A message based on the return code.
+ resources: [array] - An array of resources that includes the following information:
  + id: [string] - String ID of the resource
  + name: [string] - Friendly name of the resource
  + address: [string] - IP address or hostname of the resource
  + status: [string] - Current status of the resource ("running" or "paused")
  + tools: [array] - An array of strings with the name of each tool on this resource

__Example Request__

```javascript
GET /api/resources?token=ld91209ugfelk212lkj2
```

__Example Return__  

```javascript
{  
  "status": 200,
  "message": "OK",
  "resources": [
    {
      "resourceid": "2390309g1kdlk12109ge1209u13",
      "status": "running",
      "name": "McAtee's Massive Magical Manipulator",
      "address": "192.168.1.1",
      "tools": {
        "63ee8045-966f-449e-9839-58e7e0586f3c": {
          "name": "Hashcat",
          "version": "1.3.3", 
        },
        "8d660ce9-f15d-40a3-a997-a4e8867cb802": {
          "name": "John the Ripper",
          "version": "1.7.9", 
        },
        "8d660ce9-f15d-328b-a997-39dl10d012ld": {
          "name": "John the Ripper",
          "version": "1.8.0", 
        }
      }
    },
    {
      "resourceid": "2390309g1kdlk12109ge1209u13",
      "status": "paused",
      "name": "Lucas' Lovely Logistical Loader",
      "address": "10.0.0.1",
      "tools": {
        "63ee8045-966f-449e-9839-58e7e0586f3c": {
            "name": "Hashcat",
            "version": "1.3.3", 
        },
        "8d660ce9-f15d-40a3-a997-a4e8867cb802": {
          "name": "John the Ripper",
          "version": "1.7.9", 
        },
        "8d660ce9-f15d-328b-a997-39dl10d012ld": {
          "name": "John the Ripper",
          "version": "1.8.0", 
        }
      }
    }
  ]
}  
```

### Create <a name="resource-create"></a>
__Resource Name:__  POST /resources

Connect a resource to the queue for use.  This works by providing the IP address of the resource that we should connect to, at which point the queue will then connect to the resource and add it to the queue. 

__Arguments:__  

+ token: [string] -  User authentication token. 
+ key: [string] - Connection key configured on the resource.  Note: This is only used during initial connection, not to secure the ongoing connection.
+ name: [string] - A friendly name for the resource.
+ address: [string] - The IP address or hostname to connect to.

__Return Value__  

+ status: [int] - The return code for our function, see [[API-Status-Codes]].
+ message: [string] - A message based on the return code.

__Example Request__

```javascript
POST /api/resources

{
  "token":"dk239e09dk12lkjfge",
  "key":"supers3cretk3y",
  "address": "192.168.1.2",
  "name": "GPU Cracker 1",
}
```

__Example Return__  

```javascript
{  
  "status": 200,
  "message": "OK",
}  
```

### Read <a name="resource-read"></a>
__Resource Name:__  GET /resources/:id

Get all information about a resource.

__Arguments:__  

+ token: [string] -  User authentication token. 
+ id: [string] – ID of the resource.
+ name: [string] - Friendly name of the resource.
+ address: [string] - IP address or hostname of the resource.
 
__Return Value__  

+ status: [int] - The return code for our function, see [[API-Status-Codes]].
+ message: [string] - A message based on the return code.
+ hardware: [array] - This is an array of the hardware values. These are arbitrary, but should be somewhat descriptive in their name.
+ tools [array] - This is a map of the tools available with the tool UUID being the key.

__Example Request__

```javascript
GET /api/resources/1116814b-7c59-4b5d-87b6-fabaa5f594d1

{
  "token":"dk239e09dk12lkjfge",
}
```

__Example Return__  

```javascript
{  
  "status": 200,
  "message": "OK",
  "resource": {
    "name": "GPU Cracker 1",
    "address": "10.0.0.1",
    "hardware": ["gpu","cpu"],
    "tools": [
      "63ee8045-966f-449e-9839-58e7e0586f3c": {
        "name": "Hashcat",
        "version": "1.3.3", 
      }
      "8d660ce9-f15d-40a3-a997-a4e8867cb802": {
        "name": "John the Ripper",
        "version": "1.7.9", 
      }
      "8d660ce9-f15d-328b-a997-39dl10d012ld": {
        "name": "John the Ripper",
        "version": "1.8.0", 
      }
    ]
    "status": "running",
  }
}  
```

### Update <a name="resource-update"></a>
__Resource Name:__  PUT /resources/:id

Update the status of the resource to either shut it down or pause all jobs on it.

__Arguments:__  

+ token: [string] -  User authentication token. 
+ id: [string] – ID of the resource
+ status [string] - A string representing the status that is either resume, pause or shutdown.
 
__Return Value__  

+ status: [int] - The return code for our function, see [[API-Status-Codes]].
+ message: [string] - A message based on the return code.

__Example Request__

```javascript
PUT /api/resources/1116814b-7c59-4b5d-87b6-fabaa5f594d1

{
  "token":"dk239e09dk12lkjfge",
  "status": "paused"
}
```


__Example Return__  

```javascript
{  
  "status": 200,
  "message": "OK",
}  
```

### Delete <a name="resource-delete"></a>
__Resource Name:__  DELETE /resources/:id

Completely delete a resource from our system, stopping all jobs, deleting all data, and removing everything associated with it.

__Arguments:__  

+ token: [string] -  User authentication token. 
+ resourceid: [string] – ID of the Job
 
__Return Value__  

+ status: [int] - The return code for our function, see [[API-Status-Codes]].
+ message: [string] - A message based on the return code.

__Example Request__

```javascript
DELETE /api/resources/1116814b-7c59-4b5d-87b6-fabaa5f594d1

{
  "token":"dk239e09dk12lkjfge",
}

```

__Example Return__  

```javascript
{  
  "status": 200,
  "message": "OK",
}  
```

## Queue
### Update <a name="queue-update"></a>
__Resource Name:__  PUT /queue/ 

Take the listing of jobs within the queue and reorder the stack or pause an individual job.

__Arguments:__  

+ token: [string] -  User authentication token. 
+ joborder: [array] - An array, in order, of job IDs based on their priority in the queue.

__Return Value__  

+ status: [int] - The return code for our function, see [[API-Status-Codes]].
+ message: [string] - A message based on the return code.

__Example Request__

```javascript
PUT /api/queue/

{
  "token": "2ldljk120o89fgh31wlk12",
  "joborder": [
    "72fd24ca-e529-4b38-b70d-2ad566de7e49",
    "786c4f68-1b7f-46e0-b5bd-75090d78b25c",
    "bedf31d2-25d6-4023-a2b1-9400926c6c92",
    "baba3905-a53b-4b37-bde1-027f1fa89766",
  ]
}
```

__Example Return__  

```javascript
{  
  "status": 200,
  "message": "OK",
}  
```