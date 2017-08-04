[![Build Status](https://travis-ci.org/Scalarm/scalarm_simulation_manager_go.svg?branch=master)](https://travis-ci.org/Scalarm/scalarm_simulation_manager_go)   [![Codacy Badge](https://api.codacy.com/project/badge/Grade/cf4f2afcbefe46ffaff607f5090c055e)](https://www.codacy.com/app/Dragner8/scalarm_simulation_manager_go?utm_source=github.com&amp;utm_medium=referral&amp;utm_content=Dragner8/scalarm_simulation_manager_go&amp;utm_campaign=Badge_Grade)

scalarm_simulation_manager_go
=============================

Scalarm Simulation Manager is a Scalarm Platform worker component that manages computations on various computational resources. This version is written in Go.


Installation guide:
----------------------
Go
--
To build and install Scalarm Workers Manager you need to install go programming language.
You can install it from official binary distribution:

https://golang.org/doc/install

or from source:

https://golang.org/doc/install/source

After that you have to specify your $GOPATH. Read more about it here:

https://golang.org/doc/code.html#GOPATH

Installation
--------------
You can download Scalarm Simulation Manager directly from GitHub. You have to download it into your $GOPATH/src folder
```
go get github.com/scalarm/scalarm_simulation_manager_go
```
Now you can install Scalarm Simulation Manager:
````
go install github.com/scalarm/scalarm_simulation_manager_go
````
This command will install Scalarm Simulation Manager in $GOPATH/bin. It's name will be scalarm_simulation_manager.

Config
--------
Configuration is read from config.json file that contains required informations for Scalarm Simulation Manager:

* experiment_id (string) - optional, if not specified, all user's experiment in random order will be computed
* information_service_url (string)
* experiment_manager_user (string)
* experiment_manager_pass (string)
* development (bool)
* start_at (string)
* timeout (int)
* scalarm_certificate_path (string)
* insecure_ssl (bool)
* simulations_limit (int) - optional, if specified, execute max. N simulations

Command line options
----------------------
* ``-simulations_limit <N>`` (int) - optional, if specified, execute max. N simulations.
  Note, that it overrides ``simulations_limit`` from ``config.json``.

Run
----
Before running program you have to copy contents of config folder to folder with executable file of Scalarm Simulation Manager. By default it will be $GOPATH/bin

Testing
-------
To run all test execute in the main directory
````
go test -v ./...
````
