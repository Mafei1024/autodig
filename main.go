package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/Mafei1024/autodig/dep"
)

func main() {
	flag.Var(&dep.Help.Value, dep.Help.Key, "")
	flag.Var(&dep.H.Value, dep.H.Key, "")
	flag.Var(&dep.SameName.Value, dep.SameName.Key, "")
	flag.StringVar(&dep.OutputFile.Value, dep.OutputFile.Key, "", "")
	flag.StringVar(&dep.ScanDir.Value, dep.ScanDir.Key, "", "")
	flag.Parse()
	if dep.H.Value || dep.Help.Value {
		dep.Reminder()
		os.Exit(0)
	}
	f := func(s string, err error) {
		fmt.Println("\033[31m=========autodig failed!!!==========\033[0m")
		fmt.Println(s, ":", err)
		fmt.Println("\033[31m^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^\033[0m")
	}
	defer func() {
		if err := recover(); err != nil {
			err2, y := err.(error)
			if y && err2 != nil {
				f("panic", err2)
			}
		}
	}()
	fmt.Println("\033[34m=========autodig start==========\033[0m")
	scanDirSplit := strings.Split(dep.ScanDir.Value, ",")
	scanDirs := make([]string, 0, len(scanDirSplit))
	for _, s := range scanDirSplit {
		if s != "" {
			scanDirs = append(scanDirs, s)
		}
	}
	if len(scanDirs) == 0 || dep.OutputFile.Value == "" {
		return
	}
	fmt.Println("dir", os.Args[0])
	fmt.Println(dep.SameName.Key, dep.SameName.Value)
	fmt.Println(dep.ScanDir.Key, scanDirs)
	fmt.Println(dep.OutputFile.Key, dep.OutputFile.Value)
	err := dep.NewAutodig(scanDirs, dep.OutputFile.Value).GenDigFile()
	if err != nil {
		f("error", err)
		return
	}
	fmt.Println("\033[32m=========autodig success!!!==========\033[0m")
}
