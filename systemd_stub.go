//go:build !linux

package main

import "github.com/rs/zerolog"

func CheckSystemInit(logger zerolog.Logger) {
	return
}
