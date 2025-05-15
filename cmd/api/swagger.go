package main

//	@title			Web History Backend
//	@version		1.0
//	@description	This is a service checking last update of specific websites

//go:generate go tool swag fmt -d .,../../internal
//go:generate go tool swag init -d ../../ -g cmd/api/swagger.go -o ../../docs -ot go,json
