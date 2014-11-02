go-mailserver
===============

```go
func main() {
    port := 8888

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

func (s *Smtp) ValidateRecipient(args ...string) (int, int, string) {
	local_domains := []string{"example.com", "example.org"}
	recipient := args[0]

	re, _ := regexp.Compile("@(.*)\\s*$")
	var domain string

	if re.MatchString(recipient) {
		rets := re.FindStringSubmatch(recipient)
		domain = rets[1]
	}

	if domain == "" {
		return 0, 513, "Syntax error."
	}

	var valid = false
	for i := 0; i < len(local_domains); i++ {
		if domain == local_domains[i] {
			valid = true
			break
		}
	}

	if valid == false {
		return 0, 554, fmt.Sprintf("%s: Recipient address rejected: Relay access denied", recipient)
	}

	return 1, -1, ""
}

func (s *Smtp) QueueMessage(args ...string) (int, int, string) {
	data := args[0]
	sender := s.GetSender()
	recipients := s.GetRecipients()

	if len(recipients) == 0 {
		return 0, 554, "Error: no valid recipients"
	}

	msgid := s.AddQueue(sender, recipients, data)
	if msgid == 0 {
		return 0, -1, ""
	}

	return 1, 250, fmt.Sprintf("message queued %d", msgid)
}

func (s *Smtp) AddQueue(sender string, recipients []string, data string) int {
	queue = append(queue, data)
	return 1
}
```
