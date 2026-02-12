package main

import (
	"fmt"
	"os"

	"ariga.io/atlas-provider-gorm/gormschema"
)

func main() {
	stmts, err := gormschema.New("mysql").Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if _, err := os.Stdout.WriteString(stmts); err != nil {
		fmt.Fprintf(os.Stderr, "error writing output: %v\n", err)
		os.Exit(1)
	}
}
