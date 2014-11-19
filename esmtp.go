package smtpserver

import (
	"crypto/tls"
	"fmt"
	"strings"
)

type Esmtp struct {
	Smtp
	ExtendMode bool
	Extensions []Extension
	Xoption    map[string]map[string]func(verb string, address string, key string, value string)
	Xreply     map[string][]func(string, *Reply) (int, string)
	Options    EsmtpOption
}

type SubOption struct {
	Verb      string
	OptionKey string
	Code      func(verb string, address string, key string, value string)
}

type EsmtpOption struct {
	Option
	Ssl *tls.Config
}

func (e *Esmtp) Init(options *Option) *Esmtp {
	e.Smtp.Init(options)
	e.DefVerb("EHLO", e.Ehlo)
	e.ExtendMode = false
	e.Xoption = make(map[string]map[string]func(verb string, address string, key string, value string))
	e.Xreply = make(map[string][]func(string, *Reply) (int, string))
	e.OptionHandler = e.HandleOptions
	return e
}

func (e *Esmtp) GetProtoname() string {
	return "ESMTP"
}

func (e *Esmtp) GetExtensions() []Extension {
	return e.Extensions
}

func (e *Esmtp) Register(extend Extension) bool {
	extend.Init(e)

	for verb, code := range extend.Verb() {
		e.DefVerb(verb, code)
	}

	for _, option_def := range extend.Option() {
		e.SubOption(option_def)
	}

	for verb, code := range extend.Reply() {
		e.SubReply(verb, code)
	}

	e.Extensions = append(e.Extensions, extend)

	return true
}

func (e *Esmtp) SubOption(opt *SubOption) error {
	if opt.Verb != "MAIL" && opt.Verb != "RCPT" {
		return fmt.Errorf("can't subscribe to option for verb '%s'", opt.Verb)
	}
	if _, exists := e.Xoption[opt.Verb][opt.OptionKey]; exists == true {
		return fmt.Errorf("already subscribed '%s'", opt.OptionKey)
	}
	if e.Xoption[opt.Verb] == nil {
		e.Xoption[opt.Verb] = make(map[string]func(verb string, address string, key string, value string))
	}
	e.Xoption[opt.Verb][opt.OptionKey] = opt.Code
	return nil
}

func (e *Esmtp) SubReply(verb string, code func(string, *Reply) (int, string)) error {
	exists := false
	for _, l := range e.ListVerb() {
		if l == verb {
			exists = true
			break
		}
	}
	if exists != false {
		return fmt.Errorf("trying to subscribe to an unsupported verb '%s'", verb)
	}
	e.Xreply[verb] = append(e.Xreply[verb], code)
	return nil
}

func (e *Esmtp) SetExtendMode(mode bool) {
	e.ExtendMode = mode
	for _, extend := range e.Extensions {
		extend.SetExtendMode(mode)
	}
}

func (e *Esmtp) Ehlo(obj interface{}, args ...string) (close bool) {
	hostname := args[0]

	if hostname == "" {
		e.Reply(501, "Syntax error in parameters or arguments")
		return false
	}

	response := e.GetHostname() + " Service ready"

	e.SetExtendMode(true)
	e.MakeEvent(&Event{
		Name:      "EHLO",
		Arguments: []string{hostname},
		OnSuccess: func() {
			// according to the RFC, EHLO ensures "that both the SMTP client
			// and the SMTP server are in the initial state"
			e.ReversePath = "1"
			e.ForwardPath = []string{}
			e.StepMaildataPath(false)
		},
		SuccessReply: &Reply{Code: 250, Message: response},
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

	if len(options) == 0 {
		return true
	}

	for i := len(options) - 1; i >= 0; i-- {
		opts := strings.SplitN(options[i], "=", 2)
		var key, value string
		key = opts[0]
		if len(opts) == 2 {
			value = opts[1]
		}
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
		for _, handler := range e.Xreply[verb] {
			reply.Code, reply.Message = handler(verb, reply)
		}
	}
	e.Reply(reply.Code, reply.Message)
}
