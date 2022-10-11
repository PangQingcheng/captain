package main

import (
	"fmt"
	"runtime/debug"
)

func main() {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		fmt.Println("not ok")
		return
	}
	fmt.Printf("Version: %s\n", bi.Main.Version)
}
