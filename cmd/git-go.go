package main

import (
	"bufio"
	// "bytes"
	// "cmp"
	"compress/zlib"
	// "crypto/sha1"
	// "encoding/hex"
	"fmt"
	// "io"
	// "net/http"
	"os"
	"path/filepath"
	// "slices"
	"strconv"
	// "strings"
	// "time"
	"log"
)

func main(){
	// First element of os.Args is the path to the program and the others are the input

	fmt.Println(os.Args)

	if len(os.Args) < 2{
		fmt.Println("Invalid number of commands")
		os.Exit(1)
	}

	switch os.Args[1]{
	case "init":
		gitInit()
	case "cat-file":
		gitCatFile()
	case "hash-object":
		gitHashObject()
	default:
		fmt.Println("Invalid Commands: %s, %s", os.Args[0], os.Args[1])
		os.Exit(1)
	}
}

func gitInit(){
	directories := []string{".git", ".git/objects", ".git/branches", ".git/refs", 
	".git/refs/tags", ".git/refs/heads", ".git/refs/remotes"}
	
	// Creating the main folders
	for _, directory := range directories {
		err := os.Mkdir(directory, 0755)
		
		// Checks if the error is not that the file already exists
		if err != nil && !os.IsExist(err){
			log.Fatal("error creating directory %s : %s", directory, err)
		}

	}

	// Writing the .git/HEAD
	headPath := ".git/HEAD"
	headContent := "ref: refs/heads/main"
	err := os.WriteFile(headPath, []byte(headContent), 0644)
	if err != nil{
		log.Fatal("Error while writing file %s : %s", headPath, err)
	}

	// Writing .git/description
	descriptionPath := ".git/description"
	descriptionContent := "Unnamed repository; edit this file 'description' to name the repository.\n"
	err = os.WriteFile(descriptionPath, []byte(descriptionContent), 0644)
	if err != nil{
		log.Fatal("Error while writing file %s : %s", descriptionPath, err)
	}
	
	// Writing .git/config can be an upgrade

	fmt.Println("Empty repository initiated!")
}


func readObject(hash []byte) (objType string, objSize uint64, content []byte) {

	// Open the objects folder
	objPath := filepath.Join(".git", "objects", fmt.Sprintf("%x", hash[:1]), fmt.Sprintf("%x", hash[1:]))
	file, err := os.Open(objPath)
	if err != nil {
		fatal(err.Error())
	}
	defer file.Close()

	// Reading and decompressing the zip file
	zipReader, err := zlib.NewReader(file)
	if err != nil {
		fatal(err.Error())
	}

	// Read string reads until the first occurrence of the delimiter
	reader := bufio.NewReader(zipReader)
	objType, _ = reader.ReadString(' ')
	objType = objType[:len(objType)-1]

	lengthStr, err := reader.ReadString(0)
	lengthStr = lengthStr[:len(lengthStr)-1]
	if err != nil {
		fatal(err.Error())
	}
	objSize, _ = strconv.ParseUint(lengthStr, 10, 64)

	// Reading the content
	content = make([]byte, objSize)
	_, err = reader.Read(content)
	if err != nil && err != io.EOF {
		fatal(err.Error())
	}
	return
}


func gitCatFile() {
	if len(os.Args) < 4 || !(os.Args[2] == "-p" || os.Args[2] == "-t" || os.Args[2] == "-s" || os.Args[2] == "-e") {
		printUsageAndExit("cat-file (-p | -t | -s | -e) <object>")
	}

	objName := os.Args[3]
	if len(objName) != 40 {
		fatal("fatal: Not a valid object name %s\n", objName)
	}

	objDir := filepath.Join(".git", "objects", objName[:2])
	info, err := os.Stat(objDir)
	if err != nil {
		fatal(err.Error())
	}
	if !info.IsDir() {
		fatal("fatal: not a directory %s\n", objDir)
	}

	objPath := filepath.Join(objDir, objName[2:])
	file, err := os.Open(objPath)
	if err != nil {
		fatal(err.Error())
	}
	defer file.Close()

	if os.Args[2] == "-e" { // only check if object exists
		os.Exit(0)
	}

	zipReader, err := zlib.NewReader(file)
	if err != nil {
		fatal(err.Error())
	}

	reader := bufio.NewReader(zipReader)
	objType, _ := reader.ReadString(' ')
	objType = objType[:len(objType)-1]

	if os.Args[2] == "-t" {
		fmt.Println(objType)
		return
	}

	lengthStr, err := reader.ReadString(0)
	lengthStr = lengthStr[:len(lengthStr)-1]
	if err != nil {
		fatal(err.Error())
	}
	objSize, _ := strconv.ParseInt(lengthStr, 10, 64)

	if os.Args[2] == "-s" {
		fmt.Println(objSize)
		return
	}

	if objSize == 0 {
		fatal("error: object file %s is empty", objPath)
	}

	// default action "-p" (pretty-print)

	io.Copy(os.Stdout, reader)
}

func gitHashObject() {
	if len(os.Args) < 3 || (os.Args[2] == "-w" && len(os.Args) < 4) {
		printUsageAndExit("hash-object [-w] <object>")
	}

	var writeObject bool
	var filename string
	if os.Args[2] == "-w" {
		filename = os.Args[3]
		writeObject = true
	} else {
		filename = os.Args[2]
	}

	fmt.Printf("%x\n", hashFile(writeObject, filename))
}

func hashFile(writeObject bool, filename string) []byte {
	file, err := os.Open(filename)
	if err != nil {
		fatal(err.Error())
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		fatal(err.Error())
	}
	if info.IsDir() {
		fatal("'%s' is a directory", info.Name())
	}
	fileSize := info.Size()

	content := make([]byte, fileSize)
	_, err = file.Read(content)
	if err != nil {
		fatal(err.Error())
	}

	return hashObject(writeObject, "blob", fileSize, content)
}

func hashObject(writeObject bool, contentType string, contentSize int64, content []byte) []byte {
	payload := []byte(fmt.Sprintf("%s %d\000", contentType, contentSize))

	s := sha1.New()
	s.Write(payload)
	s.Write(content)

	hash := s.Sum(nil)
	objName := fmt.Sprintf("%x", hash)

	if !writeObject {
		return hash
	}

	objDir := filepath.Join(".git", "objects", objName[:2])
	objPath := filepath.Join(objDir, objName[2:])

	// no need to rewrite if contents match (same hash)
	if fileExists(objPath) {
		return hash
	}

	err := os.MkdirAll(objDir, 0755)
	if err != nil {
		fatal(err.Error())
	}

	objFile, err := os.OpenFile(objPath, os.O_CREATE, 0644)
	if err != nil {
		fatal(err.Error())
	}
	defer objFile.Close()
	writer := zlib.NewWriter(objFile)
	writer.Write(payload)
	writer.Write(content)
	err = writer.Close()
	if err != nil {
		fatal(err.Error())
	}

	return hash
}

func fileExists(path string) bool {
	// Stat returns file info
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
		fatal(err.Error())
	}
	return true
}





