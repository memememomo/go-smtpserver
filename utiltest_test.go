package smtpserver

import (
	"fmt"
	"log"
	"net"
	"regexp"
)

type MySmtpServer struct {
	Smtp
	Queue []string
}

func (s *MySmtpServer) ValidateRecipient(args ...string) *Reply {
	local_domains := []string{"example.com", "example.org"}
	recipient := args[0]

	re, _ := regexp.Compile("@(.*)\\s*$")
	var domain string

	if re.MatchString(recipient) {
		rets := re.FindStringSubmatch(recipient)
		domain = rets[1]
	}

	if domain == "" {
		return &Reply{0, 513, "Syntax error."}
	}

	var valid = false
	for i := 0; i < len(local_domains); i++ {
		if domain == local_domains[i] {
			valid = true
			break
		}
	}

	if valid == false {
		return &Reply{0, 554, fmt.Sprintf("%s: Recipient address rejected: Relay access denied", recipient)}
	}

	return &Reply{1, -1, ""}
}

func (s *MySmtpServer) QueueMessage(args ...string) *Reply {
	data := args[0]
	sender := s.GetSender()
	recipients := s.GetRecipients()

	if len(recipients) == 0 {
		return &Reply{0, 554, "Error: no valid recipients"}
	}

	msgid := s.AddQueue(sender, recipients, data)
	if msgid == 0 {
		return &Reply{0, -1, ""}
	}

	return &Reply{1, 250, fmt.Sprintf("message queued %d", msgid)}
}

func (s *MySmtpServer) AddQueue(sender string, recipients []string, data string) int {
	s.Queue = append(s.Queue, data)
	return 1
}

type MyEsmtpServer struct {
	Esmtp
	Queue []string
}

func (s *MyEsmtpServer) ValidateRecipient(args ...string) *Reply {
	local_domains := []string{"example.com", "example.org"}
	recipient := args[0]

	re, _ := regexp.Compile("@(.*)\\s*$")
	var domain string

	if re.MatchString(recipient) {
		rets := re.FindStringSubmatch(recipient)
		domain = rets[1]
	}

	if domain == "" {
		return &Reply{0, 513, "Syntax error."}
	}

	var valid = false
	for i := 0; i < len(local_domains); i++ {
		if domain == local_domains[i] {
			valid = true
			break
		}
	}

	if valid == false {
		return &Reply{0, 554, fmt.Sprintf("%s: Recipient address rejected: Relay access denied", recipient)}
	}

	return &Reply{1, -1, ""}
}

func (s *MyEsmtpServer) QueueMessage(args ...string) *Reply {
	data := args[0]
	sender := s.GetSender()
	recipients := s.GetRecipients()

	if len(recipients) == 0 {
		return &Reply{0, 554, "Error: no valid recipients"}
	}

	msgid := s.AddQueue(sender, recipients, data)
	if msgid == 0 {
		return &Reply{0, -1, ""}
	}

	return &Reply{1, 250, fmt.Sprintf("message queued %d", msgid)}
}

func (s *MyEsmtpServer) AddQueue(sender string, recipients []string, data string) int {
	s.Queue = append(s.Queue, data)
	return 1
}

func PrepareEsmtpServer() (*MyEsmtpServer, func(port int), chan int) {
	fin := make(chan int)
	esmtp := &MyEsmtpServer{}
	esmtpd := func(port int) {
		go func() {
			listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
			if err != nil {
				return
			}

			for {
				conn, err := listener.Accept()
				if err != nil {
					log.Printf("Accept Error: %v\n", err)
					return
				}

				esmtp.Init(&Option{Socket: conn})
				esmtp.Register(&Pipelining{})
				esmtp.Register(&Bit8mime{})
				esmtp.SetCallback("RCPT", esmtp.ValidateRecipient)
				esmtp.SetCallback("DATA", esmtp.QueueMessage)
				esmtp.Process()
				conn.Close()
			}
		}()
		<-fin
	}

	return esmtp, esmtpd, fin
}
