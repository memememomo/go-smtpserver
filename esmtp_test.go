package smtpserver

import (
	. "./testutil"
	"fmt"
	"github.com/lestrrat/go-tcptest"
	"net"
	"strconv"
	"testing"
	"time"
)

func TestEsmtpMain(t *testing.T) {
	esmtp, esmtpd, fin := PrepareEsmtpServer()

	server, err := tcptest.Start(esmtpd, 30*time.Second)
	if err != nil {
		t.Error("Failed to start smtpserver: %s", err)
	}

	conn, err := net.Dial("tcp", "localhost:"+strconv.Itoa(server.Port()))
	if err != nil {
		t.Error("Failed to connect to smtpserver")
	}
	defer conn.Close()

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
	fin <- 1
	server.Wait()
}

func Test8bitmimeInvalid(t *testing.T) {
	_, esmtpd, fin := PrepareEsmtpServer()

	server, err := tcptest.Start(esmtpd, 30*time.Second)
	if err != nil {
		t.Error("Failed to start smtpserver: %s", err)
	}

	conn, err := net.Dial("tcp", "localhost:"+strconv.Itoa(server.Port()))
	if err != nil {
		t.Error("Failed to connect to smtpserver")
	}
	defer conn.Close()

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

	fin <- 1
	server.Wait()
}

func Test8bitmimeValid(t *testing.T) {
	esmtp, esmtpd, fin := PrepareEsmtpServer()

	server, err := tcptest.Start(esmtpd, 30*time.Second)
	if err != nil {
		t.Error("Failed to start smtpserver: ", err)
	}

	conn, err := net.Dial("tcp", "localhost:"+strconv.Itoa(server.Port()))
	if err != nil {
		t.Errorf("Failed to connect to smtpserver")
	}
	defer conn.Close()

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

	fin <- 1
	server.Wait()
}

func TestPipelining(t *testing.T) {
	esmtp, esmtpd, fin := PrepareEsmtpServer()

	server, err := tcptest.Start(esmtpd, 30*time.Second)
	if err != nil {
		t.Error("Failed to start smtpserver: %s", err)
	}

	conn, err := net.Dial("tcp", "localhost:"+strconv.Itoa(server.Port()))
	if err != nil {
		t.Error("Failed to connect to smtpserver")
	}
	defer conn.Close()

	if res := ReadIO(conn); MatchRegex("220.+\\(Go\\)Service ready", res) != true {
		t.Error("Wrong Connection Response: " + res)
	}

	fmt.Fprintf(conn, "EHLO localhost\r\n")
	if res := ReadIO(conn); MatchRegex(".+? Service ready", res) != true {
		t.Error("Wrong EHLO Response: " + res)
	}

	fmt.Fprintf(conn, "MAIL FROM: <from@example.com> BODY=8BITMIME\r\nRCPT TO: <to@example.com>\r\n")
	if res := ReadIO(conn); MatchRegex("250 sender from@example.com OK\r\n", res) != true {
		t.Error("Wrong MAIL FROM Response: " + res)
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

	fin <- 1
	server.Wait()
}
