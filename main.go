package main

import "backend-go/internal/app"

var (
	Version = "dev"
	Commit  = "none"
	Build   = "local"
)

func main() {
	app.Run(Version, Commit, Build)
}
