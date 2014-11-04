package smtpserver

import (
	"bufio"
	"fmt"
	"github.com/lestrrat/go-tcptest"
	"log"
	"net"
	"regexp"
	"strconv"
	"testing"
	"time"
)

var queue []string

func TestMain(t *testing.T) {

	smtpd := func(port int) {
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

			smtp := &Smtp{
				&MailServer{},
				"",
				[]string{},
				false,
				"",
				false,
				"",
			}
			smtp = smtp.Init(&Option{Socket: conn})
			smtp.SetCallback("RCPT", smtp.ValidateRecipient)
			smtp.SetCallback("DATA", smtp.QueueMessage)
			smtp.Process()
			conn.Close()
		}
	}

	server, err := tcptest.Start(smtpd, 30*time.Second)
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

	if queue[0] != "From: from@example.net\r\nTo: to@example.com\r\nSubject: Test Mail\r\n\r\nThis is test mail.\r\n" {
		t.Error("Wrong data in queue: " + queue[0])
	}
}

func (s *Smtp) ValidateRecipient(args ...string) *Reply {
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

func (s *Smtp) QueueMessage(args ...string) *Reply {
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

func (s *Smtp) AddQueue(sender string, recipients []string, data string) int {
	queue = append(queue, data)
	return 1
}

func ReadIO(conn net.Conn) string {
	res, _ := bufio.NewReader(conn).ReadString('\n')
	return res
}

func MatchRegex(regex string, target string) bool {
	re, err := regexp.Compile(regex)
	if err != nil {
		panic(err)
	}
	return re.MatchString(target)
}
