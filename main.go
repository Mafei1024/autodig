package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/Mafei1024/autodig/dep"
)

var (
	scanDir    string
	outputFile string
)

func init() {
	dir, err := os.Getwd()
	if err != nil {
		fmt.Println(err)
		return
	}
	flag.StringVar(&scanDir, "scans", fmt.Sprintf("%s/app", dir), "source code scan dirs, split with ','")
	flag.StringVar(&outputFile, "output", fmt.Sprintf("%s/app/entrypoint/autodig.go", dir), "output file path")
}

func main() {
	fmt.Println("\033[34m=========autodig start==========\033[0m")
	flag.Parse()
	scanDirFlag := flag.Lookup("scan")
	outputFileFlag := flag.Lookup("output")
	if scanDirFlag != nil {
		scanDir = scanDirFlag.Value.String()
	}
	if outputFileFlag != nil {
		outputFile = outputFileFlag.Value.String()
	}
	fmt.Println("dir", os.Args[0])
	fmt.Println("scanDir", scanDir)
	fmt.Println("outputFile", outputFile)
	scanDirSplit := strings.Split(scanDir, ",")
	scanDirs := make([]string, 0, len(scanDirSplit))
	for _, s := range scanDirSplit {
		if s != "" {
			scanDirs = append(scanDirs, s)
		}
	}
	err := dep.NewAutodig(scanDirs, outputFile).GenDigFile()
	if err != nil {
		fmt.Println("\033[31m=========autodig failed!!!==========\033[0m")
		fmt.Println(err)
		fmt.Println("\033[31m^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^\033[0m")
		return
	}
	fmt.Println("\033[32m=========autodig success!!!==========\033[0m")
}
