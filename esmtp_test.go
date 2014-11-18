package smtpserver

import (
	"fmt"
	"github.com/lestrrat/go-tcptest"
	"log"
	"net"
	"regexp"
	"strconv"
	"testing"
	"time"
)

type MyServer struct {
	Esmtp
	Queue []string
}

func (s *MyServer) ValidateRecipient(args ...string) *Reply {
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

func (s *MyServer) QueueMessage(args ...string) *Reply {
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

func (s *MyServer) AddQueue(sender string, recipients []string, data string) int {
	s.Queue = append(s.Queue, data)
	return 1
}

func TestEsmtpMain(t *testing.T) {
	esmtp, esmtpd := PrepareServer()

	server, err := tcptest.Start(esmtpd, 30*time.Second)
	if err != nil {
		t.Error("Failed to start smtpserver: %s", err)
	}

	conn, err := net.Dial("tcp", "localhost:"+strconv.Itoa(server.Port()))
	if err != nil {
		t.Error("Failed to connect to smtpserver")
	}

	if res := ReadIO(conn); MatchRegex("220.+\\(Go\\)Service ready", res) != true {
		t.Error("Wrong Connection Response: " + res)
	}

	fmt.Fprintf(conn, "EHLO localhost\r\n")
	if res := ReadIO(conn); MatchRegex(".+? Service ready", res) != true {
		t.Error("Wrong EHLO Response: " + res)
	}

	fmt.Fprintf(conn, "MAIL FROM: <from@example.net>\r\n")
	if res := ReadIO(conn); res != "250 sender from@example.net OK\r\n" {
		t.Error("Wrong MAIL FROM Response: " + res)
	}

	fmt.Fprintf(conn, "RCPT TO: <to@example.com>\r\n")
	if res := ReadIO(conn); res != "250 recipient to@example.com OK\r\n" {
		t.Error("Wrong RCPT TO Response: " + res)
	}

	fmt.Fprintf(conn, "DATA\r\n")
	if res := ReadIO(conn); res != "354 Start mail input; end with <CRLF>.<CRLF>\r\n" {
		t.Error("Wrong DATA Response: " + res)
	}

	fmt.Fprintf(conn, "From: from@example.net\r\nTo: to@example.com\r\nSubject: Test Mail\r\n\r\nThis is test mail.\r\n.\r\n")
	if res := ReadIO(conn); res != "250 message queued 1\r\n" {
		t.Error("Wrong Data Response: " + res)
	}

	fmt.Fprintf(conn, "QUIT\r\n")
	if res := ReadIO(conn); MatchRegex("221 .+ Service closing transmission channel", res) != true {
		t.Error("Wrong QUIT Response: " + res)
	}

	if esmtp.Queue[0] != "From: from@example.net\r\nTo: to@example.com\r\nSubject: Test Mail\r\n\r\nThis is test mail.\r\n" {
		t.Error("Wrong data in queue: " + esmtp.Queue[0])
	}
}

func Test8bitmimeInvalid(t *testing.T) {
	_, esmtpd := PrepareServer()

	server, err := tcptest.Start(esmtpd, 30*time.Second)
	if err != nil {
		t.Error("Failed to start smtpserver: %s", err)
	}

	conn, err := net.Dial("tcp", "localhost:"+strconv.Itoa(server.Port()))
	if err != nil {
		t.Error("Failed to connect to smtpserver")
	}

	if res := ReadIO(conn); MatchRegex("220.+\\(Go\\)Service ready", res) != true {
		t.Error("Wrong Connection Response: " + res)
	}

	fmt.Fprintf(conn, "HELO localhost\r\n")
	if res := ReadIO(conn); res != "250 Requested mail action okey, completed\r\n" {
		t.Error("Wrong HELO Response: " + res)
	}

	fmt.Fprintf(conn, "MAIL FROM: <from@example.net> BODY=8BITMIME\r\n")
	if res := ReadIO(conn); MatchRegex("555 Unsupported option: BODY=8BITMIME", res) != true {
		t.Error("Wrong MAIL FROM Response: " + res)
	}
}

func Test8bitmimeValid(t *testing.T) {
	esmtp, esmtpd := PrepareServer()

	server, err := tcptest.Start(esmtpd, 30*time.Second)
	if err != nil {
		t.Error("Failed to start smtpserver: ", err)
	}

	conn, err := net.Dial("tcp", "localhost:"+strconv.Itoa(server.Port()))
	if err != nil {
		t.Errorf("Failed to connect to smtpserver")
	}

	if res := ReadIO(conn); MatchRegex("220.+\\(Go\\)Service ready", res) != true {
		t.Error("Wrong Connection Response: " + res)
	}

	fmt.Fprintf(conn, "EHLO localhost\r\n")
	if res := ReadIO(conn); MatchRegex(".+? Service ready", res) != true {
		t.Error("Wrong EHLO Response: " + res)
	}

	fmt.Fprintf(conn, "MAIL FROM: <from@example.com> BODY=3BITMIME\r\n")
	if res := ReadIO(conn); MatchRegex("250 sender from@example.com OK\r\n", res) != true {
		t.Error("Wrong MAIL FROM Response: " + res)
	}

	fmt.Fprintf(conn, "RCPT TO: <to@example.com>\r\n")
	if res := ReadIO(conn); res != "250 recipient to@example.com OK\r\n" {
		t.Error("Wrong RCPT TO Response: " + res)
	}

	fmt.Fprintf(conn, "DATA\r\n")
	if res := ReadIO(conn); res != "354 Start mail input; end with <CRLF>.<CRLF>\r\n" {
		t.Error("Wrong DATA Response: " + res)
	}

	fmt.Fprintf(conn, "From: from@example.net\r\nTo: to@example.com\r\nSubject: Test Mail\r\n\r\nこれはテストメールです。\r\n.\r\n")
	if res := ReadIO(conn); res != "250 message queued 1\r\n" {
		t.Error("Wrong Data Response: " + res)
	}

	fmt.Fprintf(conn, "QUIT\r\n")
	if res := ReadIO(conn); MatchRegex("221 .+ Service closing transmission channel", res) != true {
		t.Error("Wrong QUIT Response: " + res)
	}

	if esmtp.Queue[0] != "From: from@example.net\r\nTo: to@example.com\r\nSubject: Test Mail\r\n\r\nこれはテストメールです。\r\n" {
		t.Error("Wrong data in queue: " + esmtp.Queue[0])
	}
}

func PrepareServer() (*MyServer, func(port int)) {
	esmtp := &MyServer{}
	esmtpd := func(port int) {
		addr, err := net.ResolveTCPAddr("tcp", "localhost:"+strconv.Itoa(port))
		if err != nil {
			panic(err)
		}
		listener, err := net.ListenTCP("tcp", addr)
		if err != nil {
			panic(err)
		}

		for {
			conn, err := listener.AcceptTCP()
			if err != nil {
				log.Printf("Accept Error: %v\n", err)
				continue
			}

			esmtp.Init(&Option{Socket: conn})
			esmtp.Register(&Pipelining{})
			esmtp.Register(&Bit8mime{})
			esmtp.SetCallback("RCPT", esmtp.ValidateRecipient)
			esmtp.SetCallback("DATA", esmtp.QueueMessage)
			esmtp.Process()
			conn.Close()
		}
	}

	return esmtp, esmtpd
}
