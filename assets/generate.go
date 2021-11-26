package main

import (
	"fmt"
	"io/ioutil"
	"os"
)

const header = `%s = []byte{%s}
`

func writeString(file *os.File, str string) {
	_, err := file.WriteString(str)
	if err != nil {
		panic(err)
	}
}

func copyToByteSlice(name string, inputPath string, outputPath string) {
	data, err := ioutil.ReadFile(inputPath)
	if err != nil {
		panic(err)
	}

	file, err := os.Create(outputPath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	writeString(file, `package main

// Automatically generated using assets/generate.go

var `)
	writeString(file, name)
	writeString(file, " = []byte{")

	for i, v := range data {
		if i > 0 {
			writeString(file, ", ")
		}

		writeString(file, fmt.Sprint(v))
	}

	writeString(file, `}
`)
}

func main() {
	copyToByteSlice("profileImage", "./assets/alertmanager-logo.png", "server/profile_image.go")
}
