package main

import (
	"golang.org/x/crypto/ssh"
	//"net"
	"strconv"
)

func NewSshClient(host string, port int, user string, pass string) *ssh.Session {

	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(pass),
		},
	}

	client, err := ssh.Dial("tcp", host+":"+strconv.Itoa(port), config)

	// Do panic if the dial fails
	if err != nil {
		panic("Failed to dial: " + err.Error())
	}

	session, err := client.NewSession()
	if err != nil {
		panic("Failed to create session: " + err.Error())
	}
	defer session.Close()

	return session
}
