package main

import (
	"fmt"
	"os"
)

func main() {
	if err := bootstrap().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "aifactory: %v\n", err)
		os.Exit(1)
	}
}
