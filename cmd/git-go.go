package main

import (
	"bufio"
	"bytes"
	"cmp"
	"compress/zlib"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
	"log"
)

func main(){
	// First element of os.Args is the path to the program and the others are the input

	if len(os.Args) < 2{
		printUsageAndExit("")
	}

	switch os.Args[1]{
	case "init":
		gitInit()
	case "add":
		gitAdd()
	default:
		fmt.Println("Invalid Commands: %s, %s", os.Args[0], os.Args[1])
		os.Exit(1)
	}
}

// Git init basicly creates the .git folder and all the main folders and files there
func gitInit(){
	directories := []string{".git", ".git/objects", ".git/branches", ".git/refs", 
	".git/refs/tags", ".git/refs/heads", ".git/refs/remotes"}
	
	// Creating the main folders
	for _, directory := range directories {
		err := os.Mkdir(directory, 0755)
		
		// Checks if the error is not that the file already exists
		if err != nil && !os.IsExist(err){
			log.fatal("error creating directory %s : %s", directory, err)
		}

	}

	// Writing the .git/HEAD
	headPath := ".git/HEAD"
	headContent := "ref: refs/heads/main"
	err := os.WriteFile(headPath, []byte(headContent), 0644)
	if err != nil{
		log.fatal("Error while writing file %s : %s", headPath, err)
	}

	// Writing .git/description
	descriptionPath := ".git/description"
	descriptionContent := "Unnamed repository; edit this file 'description' to name the repository.\n"
	err := os.WriteFile(descriptionPath, []byte(descriptionContent), 0644)
	if err != nil{
		log.fatal("Error while writing file %s : %s", descriptionPath, err)
	}
	
	// Writing .git/config can be an upgrade

	fmt.Println("Empty repository initiated!")
}


