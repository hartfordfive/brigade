# Brigade - Distributed Load Testing Tool

## Description

A ditributed load testing tool


## Dependencies

* gnatsd (NATS messaging system)
* Go 1.4.2+


## How to run


###If testing local:

1. Ensure your NATS server is running. You can run:
  * gnatsd

2. Create a local vhost on your machine that you will use in your directives file 


Build and run the brigade process on each node:
* `cd [PATH_OF_BRIGADE_PROJECT]/node/`
* `go build`
* `./brigade -c conf/brigade.ini`

Build the commander:
* `cd [PATH_OF_BRIGADE_PROJECT]/commander/`
* `go build`
* `./commander -c conf/commander.ini`


## ToDo / Future Improvments

This tool is a work in progress so many features might be missing as well as documentation.  Much of the code will also need to be cleaned up and refactored.  The UI is also a major work in progress.  Contributions are more then welcomed although a Contributor License Agreement should be setup shortly for contributors to sign.


Keep in mind that the following types of tests capabilities are planed:

* http : Fetches an HTTP address with the specified params
* http_browser : Runs HTTP test as if it were a browser (using webloop?)
* ssh : Initiates an SSH connection and execute a given instruction once connected
* exec : Runs a shell command 
* script: Runs a specified Lua script within the go Lua VM
