package smtpserver

import (
	"fmt"
	"regexp"
)

type Xforward struct {
	ExtensionBase
	XforwardValue map[string]string
}

func (x *Xforward) Init(parent *Esmtp) Extension {
	x.Parent = parent
	return x
}

func (x *Xforward) Verb() map[string]func(interface{}, ...string) (close bool) {
	m := make(map[string]func(interface{}, ...string) (close bool))
	m["XFORWARD"] = x.XforwardFunc
	return m
}

func (x *Xforward) Keyword() string {
	return "XFORWARD"
}

func (x *Xforward) Parameter() []string {
	return []string{"NAME ADDR PROTO HELO SOURCE"}
}

func (x *Xforward) XforwardFunc(obj interface{}, args ...string) (close bool) {
	switch esmtp := obj.(type) {
	case Esmtp:
		re, _ := regexp.Compile("(NAME|ADDR|PROTO|HELO|SOURCE)=([^\\s]+)\\s*")
		var h map[string]string
		args[0] = re.ReplaceAllStringFunc(args[0], func(s string) string {
			matches := re.FindStringSubmatch(s)
			h[matches[1]] = matches[2]
			return ""
		})
		if args[0] != "" {
			esmtp.Reply(501, fmt.Sprintf("5.5.4 Bad XFORWARD attribute name: %s", args[0]))
		} else {
			for k, v := range h {
				x.XforwardValue[k] = v
			}
			esmtp.MakeEvent(&Event{
				Name:      "XFORWARD",
				Arguments: []string{x.GetForwardedName(), x.GetForwardedAddress(), x.GetForwardedProto(), x.GetForwardedHelo(), x.GetForwardedSource()},
				OnSuccess: func() {
				},
				SuccessReply: &Reply{Code: 250, Message: "OK"},
				FailureReply: &Reply{Code: 550, Message: "Failure"},
			})
		}
	}
	return false
}

func (x *Xforward) GetForwardedValues() map[string]string {
	return x.XforwardValue
}

func (x *Xforward) GetForwardedName() string {
	return x.XforwardValue["name"]
}

func (x *Xforward) GetForwardedAddress() string {
	return x.XforwardValue["addr"]
}

func (x *Xforward) GetForwardedProto() string {
	return x.XforwardValue["proto"]
}

func (x *Xforward) GetForwardedHelo() string {
	return x.XforwardValue["helo"]
}

func (x *Xforward) GetForwardedSource() string {
	return x.XforwardValue["source"]
}
