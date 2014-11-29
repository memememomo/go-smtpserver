package smtpserver

import (
	"fmt"
	"regexp"
	"strings"
)

type Smtp struct {
	MailServer
	ReversePath        string
	ForwardPath        []string
	MaildataPath       bool
	DataBuf            string
	DataHandleMoreData bool
	LastChunk          string
	OptionHandler      func(string, string, []string) bool
}

func (s *Smtp) Init(options *Option) *Smtp {
	s.MailServer.Init(options)

	s.DefVerb("HELO", s.Helo)
	s.DefVerb("VRFY", s.Vrfy)
	s.DefVerb("EXPN", s.Expn)
	s.DefVerb("TURN", s.Turn)
	s.DefVerb("HELP", s.Help)
	s.DefVerb("NOOP", s.Noop)
	s.DefVerb("MAIL", s.Mail)
	s.DefVerb("RCPT", s.Rcpt)
	s.DefVerb("SEND", s.Send)
	s.DefVerb("SOML", s.Soml)
	s.DefVerb("SAML", s.Saml)
	s.DefVerb("DATA", s.Data)
	s.DefVerb("RSET", s.Rset)
	s.DefVerb("QUIT", s.Quit)

	// go to the initial step
	s.ReversePath = "0"
	s.ForwardPath = []string{}
	s.StepMaildataPath(false)

	// handle data after the end of data indicator (.)
	s.DataHandleMoreData = false

	s.OptionHandler = s.HandleOptions

	return s
}

func (s *Smtp) StepMaildataPath(b bool) bool {
	s.MaildataPath = b
	if b == false {
		s.DataBuf = ""
	}
	return s.MaildataPath
}

func (s *Smtp) GetProtoname() string {
	return "SMTP"
}

func (s *Smtp) GetSender() string {
	return s.ReversePath
}

func (s *Smtp) GetRecipients() []string {
	return s.ForwardPath
}

func (s *Smtp) Helo(obj interface{}, args ...string) (close bool) {
	if len(args) == 0 || args[0] == "" {
		s.Reply(501, "Syntax error in parameters or arguments")
		return false
	}

	hostname := args[0]

	s.MakeEvent(&Event{
		Name:      "HELO",
		Arguments: []string{hostname},
		OnSuccess: func() {
			// according to the RFC, HELO ensures "that both the SMTP client
			// and the SMTP server are in the initial state"
			s.ReversePath = "1"
			s.ForwardPath = []string{}
			s.MaildataPath = false
		},
		SuccessReply: &Reply{
			Code:    250,
			Message: "Requested mail action okey, completed",
		},
	})

	return false
}

func (s *Smtp) Noop(obj interface{}, args ...string) (close bool) {
	s.MakeEvent(&Event{
		Name: "NOOP",
	})

	return false
}

func (s *Smtp) Expn(obj interface{}, args ...string) (close bool) {
	s.MakeEvent(&Event{
		Name:         "EXPN",
		Arguments:    args,
		DefaultReply: &Reply{Code: 502, Message: "Command not implemented"},
	})

	return false
}

func (s *Smtp) Turn(obj interface{}, args ...string) (close bool) {
	// deprecated in RFC 2821
	s.Reply(502, "Command not implemented")
	s.MakeEvent(&Event{
		Name:         "TURN",
		DefaultReply: &Reply{Code: 502, Message: "Command not implemented"},
	})

	return false
}

func (s *Smtp) Vrfy(obj interface{}, address ...string) (close bool) {
	s.MakeEvent(&Event{
		Name:         "VRFY",
		Arguments:    address,
		DefaultReply: &Reply{Code: 502, Message: "Command not implemented"},
	})

	return false
}

func (s *Smtp) Help(obj interface{}, args ...string) (close bool) {
	s.MakeEvent(&Event{
		Name:         "HELP",
		Arguments:    args,
		DefaultReply: &Reply{Code: 502, Message: "Command not implemented"},
	})

	return false
}

func (s *Smtp) Mail(obj interface{}, args ...string) (close bool) {
	if s.ReversePath == "0" {
		s.Reply(503, "Bad sequence of commands")
		return false
	}

	re, _ := regexp.Compile("^from:\\s*")
	if re.MatchString(strings.ToLower(args[0])) == false {
		s.Reply(501, "Syntax error in parameters or arguments")
		return false
	}
	index := re.FindStringIndex(strings.ToLower(args[0]))
	args[0] = args[0][index[1]:]

	if len(s.ForwardPath) > 0 {
		s.Reply(503, "Bad sequence of commands")
		return false
	}

	re, _ = regexp.Compile("^<(.*?)>(?: (\\S.*))?$")
	if re.MatchString(args[0]) == false {
		s.Reply(501, "Syntax error in parameters or arguments")
		return false
	}
	rets := re.FindAllStringSubmatch(args[0], -1)
	address := rets[0][1]

	var options []string
	if len(rets[0]) > 1 && rets[0][2] != "" {
		options = strings.Split(rets[0][2], " ")
	}

	if s.OptionHandler("MAIL", address, options) == false {
		return false
	}

	s.MakeEvent(&Event{
		Name:      "MAIL",
		Arguments: []string{address},
		OnSuccess: func() {
			s.ReversePath = address
			s.ForwardPath = []string{"1"}
		},
		SuccessReply: &Reply{Code: 250, Message: fmt.Sprintf("sender %s OK", address)},
		FailureReply: &Reply{Code: 550, Message: "Failure"},
	})

	return false
}

func (s *Smtp) Rcpt(obj interface{}, args ...string) (close bool) {
	if len(s.ForwardPath) <= 0 {
		s.Reply(503, "Bad sequence of commands")
		return false
	}

	re, _ := regexp.Compile("^to:\\s*")
	if re.MatchString(strings.ToLower(args[0])) == false {
		s.Reply(501, "Syntax error in parameters or arguments")
		return false
	}
	args[0] = re.ReplaceAllString(strings.ToLower(args[0]), "")

	re, _ = regexp.Compile("^<(.*?)>(?: (\\S.*))?$")
	if re.MatchString(args[0]) == false {
		s.Reply(501, "Syntax error int parameters or arguments")
		return false
	}
	rets := re.FindAllStringSubmatch(args[0], -1)
	address := rets[0][1]

	var options []string
	if len(rets) > 1 {
		options = strings.Split(rets[1][1], " ")
	}

	if s.OptionHandler("RCPT", address, options) == false {
		return false
	}

	s.MakeEvent(&Event{
		Name:      "RCPT",
		Arguments: []string{address},
		OnSuccess: func() {
			buffer := s.ForwardPath
			if len(buffer) == 1 && buffer[0] == "1" {
				buffer = []string{}
			}
			buffer = append(buffer, address)
			s.ForwardPath = buffer
			s.StepMaildataPath(true)
		},
		SuccessReply: &Reply{Code: 250, Message: fmt.Sprintf("recipient %s OK", address)},
		FailureReply: &Reply{Code: 550, Message: "Failure"},
	})

	return false
}

func (s *Smtp) Send(obj interface{}, args ...string) (close bool) {
	s.MakeEvent(&Event{
		Name:         "SEND",
		DefaultReply: &Reply{Code: 502, Message: "Command not implemented"},
	})
	return false
}

func (s *Smtp) Soml(obj interface{}, args ...string) (close bool) {
	s.MakeEvent(&Event{
		Name:         "SOML",
		DefaultReply: &Reply{Code: 502, Message: "Command not implemented"},
	})
	return false
}

func (s *Smtp) Saml(obj interface{}, args ...string) (close bool) {
	s.MakeEvent(&Event{
		Name:         "SAML",
		DefaultReply: &Reply{Code: 502, Message: "Command not implemented"},
	})
	return false
}

func (s *Smtp) Data(obj interface{}, args ...string) (close bool) {
	if s.MaildataPath == false {
		s.Reply(503, "Bad sequence of commands")
		return false
	}

	if args[0] != "" {
		s.Reply(501, "Syntax error in parameters or arguments")
		return false
	}

	s.LastChunk = ""
	s.MakeEvent(&Event{
		Name:         "DATA-INIT",
		OnSuccess:    func() { s.NextInputTo(s.DataPart) },
		SuccessReply: &Reply{Code: 354, Message: "Start mail input; end with <CRLF>.<CRLF>"},
	})

	return false
}

// Because data is cutted into pieces (4096 bytes), we have to search
// "\r\n.\r\n" sequence in 2 consecutive pieces. s.LastChunk
// contains the last 5 bytes.
func (s *Smtp) DataPart(data string) bool {

	// search for end of data indicator
	re, _ := regexp.Compile("\r?\n\\.\r?\n(.*)")
	if re.MatchString(s.LastChunk+data) == true {
		var more_data string
		ret := re.FindAllStringSubmatch(s.LastChunk+data, -1)
		if len(ret) > 0 {
			more_data = ret[0][1]
		}
		if len(more_data) > 0 {
			// Client sent a command after the end of data indicator ".".
			if s.DataHandleMoreData == false {
				s.Reply(453, "Command received prior to completion of previous command sequence")
				return false
			}
		}

		// RFC 821 compliance.
		data = s.LastChunk + data
		re, _ = regexp.Compile("(\r?\n)\\.\r?\n(QUIT\r?\n)?$")
		cb := func(s string) string {
			match := re.FindStringSubmatch(s)
			return match[1]
		}
		data = re.ReplaceAllStringFunc(data, cb)

		s.DataBuf += data

		// RFC 2821 by the letter
		re, _ = regexp.Compile("^\\.(.+\015\012)(.)?")
		cb = func(s string) string {
			match := re.FindStringSubmatch(s)
			if match[2] == "" || match[2] != "\n" {
				return match[1]
			} else {
				return s
			}
		}
		s.DataBuf = re.ReplaceAllStringFunc(s.DataBuf, cb)

		return s.DataFinished(more_data)
	}

	n := len(data)
	tmp := s.LastChunk
	if n >= 5 {
		s.LastChunk = data[n-5 : n]
		data = tmp + data[0:n-5]
	} else {
		s.LastChunk = data
		data = tmp + data
	}
	s.MakeEvent(&Event{
		Name:      "DATA-PART",
		Arguments: []string{data},
		OnSuccess: func() {
			s.DataBuf += data

			// please, recall me soon !
			s.NextInputTo(s.DataPart)
		},
		SuccessReply: &Reply{Code: 0}, // don't send any reply!
	})

	return false
}

func (s *Smtp) DataFinished(more_data string) bool {
	s.MakeEvent(&Event{
		Name:         "DATA",
		Arguments:    []string{s.DataBuf},
		SuccessReply: &Reply{Code: 250, Message: "message sent"},
	})

	// reinitiate the connection
	s.ReversePath = "1"
	s.ForwardPath = []string{}
	s.StepMaildataPath(false)

	// if more data, handle it
	if len(more_data) > 0 {
		return s.CurProcessOperation(more_data)
	} else {
		return false
	}
}

func (s *Smtp) Rset(obj interface{}, args ...string) bool {
	s.MakeEvent(&Event{
		Name: "RSET",
		OnSuccess: func() {
			if s.ReversePath != "0" {
				s.ReversePath = "1"
			}
			s.ForwardPath = []string{}
			s.StepMaildataPath(false)
		},
		SuccessReply: &Reply{Code: 250, Message: "Requested mail action okay, completed"},
	})
	return false
}

func (s *Smtp) Quit(obj interface{}, args ...string) bool {
	s.MakeEvent(&Event{
		Name:         "QUIT",
		SuccessReply: &Reply{Code: 221, Message: fmt.Sprintf("%s Service closing transmission channel", s.GetHostname())},
	})
	return true // close cnx
}

func (s *Smtp) HandleOptions(verb string, address string, options []string) bool {
	if len(options) > 0 {
		s.Reply(555, fmt.Sprintf("Unsupported option: %s", options[0]))
		return false
	}
	return true
}
