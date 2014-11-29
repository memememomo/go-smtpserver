package smtpserver

import (
	"fmt"
)

type Lmtp struct {
	Esmtp
}

func (l *Lmtp) Init(options *Option) *Lmtp {
	l.Esmtp.Init(options)
	l.UndefVerb("HELO")
	l.UndefVerb("EHLO")
	l.DefVerb("LHLO", l.Lhlo)

	// Required by RFC
	l.Register(&Pipelining{})

	return l
}

func (l *Lmtp) GetProtoname() string {
	return "LMTP"
}

func (l *Lmtp) Lhlo(obj interface{}, args ...string) (close bool) {
	if len(args) == 0 || args[0] != "" {
		l.Reply(501, "Syntax error in parameters or arguments")
		return
	}

	hostname := args[0]
	response := l.GetHostname() + " Service ready"

	l.ExtendMode = true

	l.MakeEvent(&Event{
		Name:      "LHLO",
		Arguments: []string{hostname},
		OnSuccess: func() {
			l.ExtendMode = true
			l.ReversePath = "1"
			l.ForwardPath = []string{}
			l.MaildataPath = false
		},
		SuccessReply: &Reply{Code: 250, Message: response},
	})

	return false
}

func (l *Lmtp) DataFinished(more_data string) bool {
	recipients := l.ForwardPath

	for _, forward_path := range recipients {
		l.MakeEvent(&Event{
			Name:         "DATA",
			Arguments:    []string{l.DataBuf, forward_path},
			SuccessReply: &Reply{Code: 250, Message: "Ok"},
			FailureReply: &Reply{Code: 550, Message: fmt.Sprintf("%s Failed", forward_path)},
		})
	}

	// reinitiate the connection
	l.ReversePath = "1"
	l.ForwardPath = []string{}
	l.StepMaildataPath(false)

	return false
}
