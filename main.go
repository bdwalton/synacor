package main

import (
	"encoding/binary"
	"flag"
	"github.com/bdwalton/synacor/synacor"
	"io/ioutil"
	"log"
)

var binaryFile = flag.String("binary_file", "", "The binary program file.")

func main() {
	flag.Parse()

	bin, err := ioutil.ReadFile(*binaryFile)
	if err != nil {
		log.Fatalf("Couldn't open %q: %v", *binaryFile, err)
	}

	prog := make([]uint16, 0)

	for i := 0; i <= len(bin); i += 2 {
		val := binary.LittleEndian.Uint16(bin[i : i+2])
		prog = append(prog, val)
	}

	m := synacor.NewMachine(prog)

	m.Run()
}
