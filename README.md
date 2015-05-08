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

* experiment_id (string)
* information_service_url (string)
* experiment_manager_user (string)
* experiment_manager_pass (string)
* development (bool)
* start_at (string)
* timeout (int)
* scalarm_certificate_path (string)
* insecure_ssl (bool)

Run 
---- 
Before running program you have to copy contents of config folder to folder with executable file of Scalarm Simulation Manager. By default it will be $GOPATH/bin 

