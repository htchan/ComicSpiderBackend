package main

//	@title			Web History Backend
//	@version		1.0
//	@description	This is a service checking last update of specific websites

//go:generate swag fmt -d .,../../internal
//go:generate swag init -d ../../ -g cmd/api/swagger.go -o ../../docs -ot go,json
