package main

import (
	"flag"
	"os"

	"github.com/jguillaumes/xmit_reader/internal/xmitfile"
)

func main() {
	// Command line arguments:
	// --input <input_file>: The input XMIT file to process.
	// --target <target_directory>: The directory where the output files will be saved.

	inputFile := flag.String("input", "", "L'arxiu XMIT d'entrada a processar")
	targetDir := flag.String("target", "", "Directori on es guardaran els arxius de sortida")

	flag.Parse()

	// Check if input file and target directory are provided
	if *inputFile == "" || *targetDir == "" {
		flag.Usage()
		os.Exit(1)
	}

	// Check if the targert directory exists
	if _, err := os.Stat(*targetDir); os.IsNotExist(err) {
		println("Target directory does not exist:", *targetDir)
		os.Exit(1)
	}

	// Open the input file
	inFile, err := os.Open(*inputFile)
	if err != nil {
		println("Error opening input file:", err.Error())
		os.Exit(1)
	}
	defer inFile.Close()

	// Process the input file and generate output files
	var count int
	count, err = xmitfile.ProcessXMITFile(inFile, *targetDir)
	if err != nil {
		println("Error processing input file:", err.Error())
		os.Exit(1)
	}

	if count == 0 {
		println("No members extracted from the input file.")
	} else {
		println("Successfully processed", count, "members from the XMIT input file.")
	}
	os.Exit(0)
}
