// Command extract exports native presentation data from an Atari executable.
package main

import (
	"flag"
	"github.com/olivierh59500/go-lockbuster2/internal/source"
	"log"
	"os"
)

func main() {
	input := flag.String("input", "", "native executable")
	directory := flag.String("assets", "", "asset output directory")
	flag.Parse()
	if *input == "" || *directory == "" {
		log.Fatal("-input and -assets required")
	}
	b, e := os.ReadFile(*input)
	if e != nil {
		log.Fatal(e)
	}
	if e = source.ExportAssets(b, *directory); e != nil {
		log.Fatal(e)
	}
}
