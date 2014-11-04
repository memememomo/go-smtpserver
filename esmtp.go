package smtpserver

import (
	"fmt"
)

type Esmtp struct {
	*Smtp
	ExtendMode bool
}

type SubOption struct {
	Verb      string
	OptionKey string
	Code      interface{}
}

func (e *Esmtp) Init(options *Option) *Esmtp {
	e.Smtp.Init(options)
	e.DefVerb("EHLO", e.Ehlo)
	e.ExtendMode = false
	return e
}

func (e *Esmtp) GetProtoname() string {
	return "ESMTP"
}

func (e *Esmtp) GetExtensions() {
	return e.Extensions
}

func (e *Esmtp) Register(extend *Extension) bool {
	for _, verb_def := range extend.Verb() {
		e.DefVerb(verb_def)
	}

	for _, option_def := range extend.Option {
		e.SubOption(option_def)
	}

	for _, reply_def := range extend.Reply {
		e.SubReply(reply_def)
	}

	e.Extensions = append(e.Extensions, extend)

	return true
}

func (e *Esmtp) SubOption(opt *SubOption) error {
	if opt.Verb != "MAIL" && opt.Verb != "RCPT" {
		return fmt.Sprintf("can't subscribe to option for verb '%s'", verb)
	}
	if _, exists := e.Xoption[opt.Verb][opt.OptionKey]; exists == true {
		return fmt.Sprintf("already subscribed '%s'", opt.OptionKey)
	}
	e.Xoption[opt.Verb][opt.OptionKey] = opt.Code
}

func (e *Esmtp) SubReply(verb string, code string) error {
	exists := false
	for _, l := range e.ListVerb() {
		if l == verb {
			exists = true
			break
		}
	}
	if exists != false {
		return fmt.Sprintf("trying to subscribe to an unsupported verb '%s'", verb)
	}

	e.Xreply[verb] = append(e.Xreply[verb], code)
	return nil
}

func (e *Esmtp) SetExtendMode(mode bool) {
	e.ExtendMode = mode
	for _, extend := range e.Extensions {
		extend.ExtendMode = mode
	}
}

func (e *Esmtp) Ehlo(hostname string) (close bool) {
	if len(hostname) > 0 {
		e.Reply(501, "Syntax error in parameters or arguments")
		return false
	}

	response := e.GetHostname() + " Service ready"

	var extends
	for _, extend := range e.GetExtensions() {
		extends = append(extends, extend)
	}

	e.SetExtendMode(true)
	e.MakeEvent(&Event{
		Name:      "EHLO",
		Arguments: []string{hostname, extends},
		OnSuccess: func() {
			// according to the RFC, EHLO ensures "that both the SMTP client
			// and the SMTP server are in the initial state"
			e.ReversePath = true
			e.ForwardPath = []string{}
			e.StepMaildataPath(0)
		},
		SuccessReply: &Reply{Code: 250}, // [$response, @extends]
	})

	return false
}

func (e *Esmtp) Helo(hostname string) (close bool) {
	e.ExtendMode = false
	return e.Smtp.Helo(hostname)
}

func (e *Esmtp) HandleOptions(verb string, address string, options []string) bool {
	if len(options) > 0 && e.ExtendMode == false {
		e.Reply(555, fmt.Sprintf("Unsupported option: %s", options[0]))
		return false
	}

	for i := len(options); i >= 0; i-- {
		key, value := strings.SplitN(options[i], "=", 2)
		handler, ok := e.Xoption[verb][key]
		if ok {
			handler(verb, address, key, value)
		} else {
			e.Reply(555, fmt.Sprintf("Unsupported option: %s", key))
			return false
		}
	}

	return true
}

func (e *Esmtp) HandleReply(verb string, reply *Reply) {
	if _, ok := e.Xreply[verb]; e.ExtendMode && ok {
		for handler := range e.Xreply[verb] {
			reply.Code, reply.Message = handler(verb, reply)
		}
	}
	e.Reply(reply.Code, reply.Message)
}
