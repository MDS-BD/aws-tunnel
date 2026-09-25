package main

import "github.com/MDS-BD/aws-tunnel/internal/app"

// version is set at release time by goreleaser.
var version = "dev"

func main() {
	app.Main(version)
}
