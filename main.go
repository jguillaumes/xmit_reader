package main

import (
	"flag"
	"io"
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/jguillaumes/xmit_reader/internal/unloadfile"
	"github.com/jguillaumes/xmit_reader/internal/xmitfile"
)

func main() {
	rc := 0

	logf := log.TextFormatter{
		PadLevelText:           true,
		DisableLevelTruncation: true,
		FullTimestamp:          true,
		TimestampFormat:        "02-Jan-2006 03:04:05.000",
	}

	log.SetFormatter(&logf)

	inputFile := flag.String("input", "", "Input XMIT file to be processed")
	targetDir := flag.String("target", "", "Path to the output directory")
	typeExt := flag.String("type", "", "File type (to be used as extension)")
	unloadFile := flag.String("unload", "", "Name of the IEBCOPY unload file. If not specified it will be not kept and a temporary file will be used")
	debugFlag := flag.Bool("debug", false, "Output debug information (maybe quite verbose)")
	encoding := flag.String("encoding", "IBM-1047", "EBCDIC encoding used in the original files. The default is IBM-1047")
	traceFlag := flag.Bool("trace", false, "Maximum debug output. VERY verbose")

	flag.Parse()

	if *debugFlag {
		log.SetLevel(log.DebugLevel)
	}

	if *traceFlag {
		log.SetLevel(log.TraceLevel)
		log.SetReportCaller(true)
	}

	var deleteUnloadFile bool = true

	// Check if input file, target directory and file type are provided
	if *inputFile == "" || *targetDir == "" || *typeExt == "" {
		flag.Usage()
		os.Exit(16)
	}

	// Check if the targert directory exists
	if _, err := os.Stat(*targetDir); os.IsNotExist(err) {
		log.Error("Target directory does not exist: ", *targetDir)
		os.Exit(4)
	}

	// Check if an unload file is specified. Id so, open it for write
	// otherwise, create a temporary file
	if *unloadFile == "" {
		tempFile, err := os.CreateTemp("", "xmit_unload_*.unload")
		if err != nil {
			log.Error("Error creating temporary unload file:", err.Error())
			os.Exit(8)
		}
		// defer tempFile.Close()
		*unloadFile = tempFile.Name()
		deleteUnloadFile = true
	} else {
		deleteUnloadFile = false
	}

	// Unconditionally open the unload file for writing
	unloadFileHandle, err := os.OpenFile(*unloadFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		log.Error("Error opening unload file:", err.Error())
		os.Exit(8)
	}

	// Open the input file
	inFile, err := os.Open(*inputFile)
	if err != nil {
		log.Error("Error opening input file:", err.Error())
		os.Exit(8)
	}
	defer inFile.Close()

	// Process the input file and generate output files
	xmitParms, err := xmitfile.ProcessXMITFile(inFile, *targetDir, unloadFileHandle, *encoding)
	if err != nil {
		log.Error("Error processing input file:", err.Error())
		os.Exit(8)
	}

	xmf := xmitParms.XmitFiles[0]
	log.Infof("Original dataset: %s\n", xmf.SourceDSName)
	log.Infof("Dataset attributes: DSORG=%s, DSTYPE=%s, RECFM=%s, LRECL=%d, BLKSIZE=%d\n",
		xmf.SourceDsorg, xmf.SourceDstype, xmf.SourceRecfm, xmf.SourceLrecl, xmf.SourceBlksize)
	log.Infof("Using codepage %s for conversion\n", *encoding)

	// Close the unload file handle
	if err := unloadFileHandle.Close(); err != nil {
		log.Error("Error closing unload file:", err.Error())
		os.Exit(8)
	}
	// Reopen the unload file to read its contents
	unloadFileHandle, err = os.Open(*unloadFile)
	if err != nil {
		log.Error("Error reopening unload file for reading:", err.Error())
		os.Exit(8)
	}
	defer unloadFileHandle.Close()

	nfiles, err := unloadfile.ProcessUnloadFile(*unloadFileHandle, *targetDir, *typeExt, xmf, *encoding)
	if err != nil && err != io.EOF {
		log.Errorln(err)
		rc = 8
	}

	err = unloadFileHandle.Close()
	if (err != nil) {
		log.Errorln(err)
	}
 
	if deleteUnloadFile {
		// Delete the unload file if it was created as a temporary file
		if err := os.Remove(*unloadFile); err != nil {
			log.Warn("Error deleting unload file:", err.Error())
			rc = 2
		} else {
			log.Debugln("Temporary unload file deleted:", *unloadFile)
		}
	}
	log.Infof("%d members expanded from XMIT file %s\n", nfiles, *inputFile)
	os.Exit(rc)
}
