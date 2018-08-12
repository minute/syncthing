package main

import "C"

//export RunSyncthing
func RunSyncthing() {
	syncthingMain(RuntimeOptions{})
}
