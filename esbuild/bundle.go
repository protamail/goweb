package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/evanw/esbuild/pkg/api"
)

func main() {
	var minify bool
	var inFile, outFile string
	flag.StringVar(&inFile, "in", "", "Input file to bundle and optionally minify")
	flag.StringVar(&outFile, "out", "", "Bundled and optionally minified output goes into this file")
	flag.BoolVar(&minify, "minify", false, "Minify output (default: false)")
	flag.Parse()

	if len(inFile) == 0 || len(outFile) == 0 {
		fmt.Fprint(os.Stderr, "Error: must specify both -in and -out options\n")
		fmt.Fprintf(os.Stderr, "Usage: %s options\nAvailable options are:\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}
	result := api.Build(api.BuildOptions{
		EntryPoints:       []string{inFile},
		Outfile:           outFile,
		Bundle:            true,
		MinifyWhitespace:  minify,
		MinifyIdentifiers: minify,
		MinifySyntax:      minify,
		Write:             true,
		TreeShaking:       api.TreeShakingTrue,
		LogLevel:          api.LogLevelWarning,
	})

	if len(result.Errors) > 0 {
		os.Exit(1)
	}
}
