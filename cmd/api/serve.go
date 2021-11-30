package main

import smr "github.com/satisfactorymodding/smr-api"

// @title Satisfactory Mod Repo API
// @version 1
// @description Satisfactory Mod Repo API documentation.

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host api.ficsit.app
// @BasePath /v1
func main() {
	smr.Serve()
}
