package smtpserver

import (
	. "./testutil"
	"fmt"
	"github.com/lestrrat/go-tcptest"
	"log"
	"net"
	"strconv"
	"testing"
	"time"
)

func TestSmtpMain(t *testing.T) {
	smtp := &MySmtpServer{}

	smtpd := func(port int) {
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			panic(err)
		}

		for {
			conn, err := listener.Accept()

			if err != nil {
				log.Printf("Accept Error: %v\n", err)
				continue
			}

			smtp.Init(&Option{Socket: conn})
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

	if smtp.Queue[0] != "From: from@example.net\r\nTo: to@example.com\r\nSubject: Test Mail\r\n\r\nThis is test mail.\r\n" {
		t.Error("Wrong data in queue: " + smtp.Queue[0])
	}
}
