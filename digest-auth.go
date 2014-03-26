package main

import (
	"bufio"
	"crypto/md5"
	"fmt"
	auth "github.com/abbot/go-http-auth"
	"log"
	"os"
	"strings"
)

func loadOrCreateDigestFile(name string) *os.File {
	if _, err := os.Stat(name); os.IsNotExist(err) {
		file, err := os.Create(name)
		if err != nil {
			log.Fatalf("Error with os.Create(): %v", err)
		}
		return file
	} else {
		file, err := os.Open(name)
		if err != nil {
			log.Fatalf("Error with os.Open(): %v", err)
		}
		return file
	}
}

func loadDigestAuth(realm string) auth.DigestAuth {
	file := loadOrCreateDigestFile(settings.DigestPath)
	info, _ := file.Stat()
	if info.Size() == 0 {
		if *debug {
			log.Println("Digest File is empty, requesting user and pass")
		}
		fmt.Println("You specified a digest file but it's empty, please choose a user & pass to login to your BTSyncInator site.")
		// Get Username
		fmt.Println("Username:")
		bufferedIO := bufio.NewReader(os.Stdin)
		username, err := bufferedIO.ReadString('\n')
		if err != nil {
			log.Fatalf("Error with bufferedIO.ReadString(\"\n\"): %v", err)
		}
		username = strings.Replace(username, "\n", "", -1)
		// Get Password
		fmt.Println("Password:")
		bufferedIO = bufio.NewReader(os.Stdin)
		password, err := bufferedIO.ReadString('\n')
		if err != nil {
			log.Fatalf("Error with bufferedIO.ReadString(\"\n\"): %v", err)
		}
		password = strings.Replace(password, "\n", "", -1)
		// Setup string to hash through md5
		byteContents := []byte(username)
		byteContents = append(byteContents, []byte(":")...)
		byteContents = append(byteContents, []byte(realm)...)
		byteContents = append(byteContents, []byte(":")...)
		byteContents = append(byteContents, []byte(password)...)
		encryptedBytePassArray := md5.Sum(byteContents)
		encryptedBytePass := encryptedBytePassArray[:]
		// Setup bytes to write to htdigest file
		byteContents = []byte(username)
		byteContents = append(byteContents, []byte(":")...)
		byteContents = append(byteContents, []byte(realm)...)
		byteContents = append(byteContents, []byte(":")...)
		byteContents = append(byteContents, []byte(fmt.Sprintf("%x", encryptedBytePass))...)
		byteContents = append(byteContents, []byte("\n")...)
		_, err = file.Write(byteContents)
		if err != nil {
			log.Fatalf("Error with file.Write(): %v", err)
		}
		file.Close()
	} else if *debug {
		log.Printf("Digest File %s loaded", file.Name())
	}
	secretProvider := auth.HtdigestFileProvider(file.Name())
	digestAuth := auth.NewDigestAuthenticator(realm, secretProvider)
	return *digestAuth
}
