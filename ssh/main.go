package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// check for any error
func check(err error) {
	if err != nil {
		fmt.Printf("Error Happened %s \n", err)
		os.Exit(1)
	}
}

var (
	sshUserName        = "ssafarveisi"
	sshKeyPath         = "/home/ssafarveisi/.ssh/id_rsa"
	sshHostname        = "85.215.182.83:22"
	commandToExec      = "echo \"I am connected to $(hostname)\""
	fileToUpload       = "./upload.txt"
	fileUploadLocation = "/home/ssafarveisi/upload.txt"
	fileToDownload     = "/home/ssafarveisi/download.txt"
)

func main() {

	fmt.Println("....Golang SSH Demo......")

	conf := sshDemoWithPrivateKey() // username and private key authentication

	// open ssh connection
	sshClient, err := ssh.Dial("tcp", sshHostname, conf)
	check(err)
	session, err := sshClient.NewSession()
	check(err)
	defer session.Close()

	// execute command on remote server
	var b bytes.Buffer
	session.Stdout = &b
	err = session.Run(commandToExec)
	check(err)
	log.Printf("%s: %s", commandToExec, b.String())

	// open sftp connection
	sftpClient, err := sftp.NewClient(sshClient)
	check(err)
	defer sftpClient.Close()

	// Upload a file
	srcFile, err := os.Open(fileToUpload)
	check(err)
	defer srcFile.Close()

	dstFile, err := sftpClient.Create(fileUploadLocation)
	check(err)
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	check(err)
	fmt.Println("File uploaded successfully ", fileUploadLocation)

	// Download a file
	remoteFile, err := sftpClient.Open(fileToDownload)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to open remote file: %v\n", err)
		return
	}
	defer remoteFile.Close()

	localFile, err := os.Create("./download.txt")
	check(err)
	defer localFile.Close()

	_, err = io.Copy(localFile, remoteFile)
	check(err)
	fmt.Println("File downloaded successfully")

}

func sshDemoWithPrivateKey() *ssh.ClientConfig {
	keyByte, err := os.ReadFile(sshKeyPath)
	check(err)
	key, err := ssh.ParsePrivateKey(keyByte)
	check(err)

	// ssh config
	conf := &ssh.ClientConfig{
		User: sshUserName,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(key),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	return conf
}
