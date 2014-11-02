package smtpserver

import (
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

type MailServer struct {
	In                  net.Conn
	Out                 net.Conn
	DoJob               bool
	Context             string
	CallbackMap         map[string]*Callback
	Verb                map[string]func(interface{}, ...string) (close bool)
	NextInput           func(string) bool
	Options             *Option
	BannerString        string
	CurProcessOperation func(string) bool
}

type Option struct {
	HandleIn       net.Conn
	HandleOut      net.Conn
	Socket         net.Conn
	ErrorSleepTime int
	IdleTimeout    int
}

type Reply struct {
	Success int
	Code    int
	Message string
}

type Event struct {
	Name         string
	Arguments    []string
	OnSuccess    func()
	OnFailure    func()
	DefaultReply *Reply
	SuccessReply *Reply
	FailureReply *Reply
}

type Callback struct {
	Code    func(...string) *Reply
	Context string
}

func (m *MailServer) Init(options *Option) *MailServer {
	m.Options = options

	m.In = options.Socket
	m.Out = options.Socket
	m.CallbackMap = make(map[string]*Callback)
	m.Verb = make(map[string]func(interface{}, ...string) (close bool))

	m.CurProcessOperation = m.ProcessOperation

	return m
}

func (m *MailServer) InitDojob() {
	m.DoJob = true
}

func (m *MailServer) MakeEvent(e *Event) int {
	name := e.Name
	args := e.Arguments

	m.InitDojob()
	reply := m.Callback(name, args...)

	// we have to take a proper decision if successness is undefined
	if reply.Success == -1 {
		if e.DefaultReply != nil {
			reply.Success = e.DefaultReply.Success
			reply.Code = e.DefaultReply.Code
			reply.Message = e.DefaultReply.Message
		} else {
			// default
			reply.Success = 1
		}
	}

	// command may have some job to do regarding to the result. handler
	// can avoid it by calling dojob() method with a false value.
	if m.DoJob {
		if reply.Success > 0 {
			if e.OnSuccess != nil {
				e.OnSuccess()
			}
		} else {
			if e.OnFailure != nil {
				e.OnFailure()
			}
		}
	}

	// ensure that a reply is sent, all SMTP command need at most 1 reply.
	if reply.Code == -1 {
		if reply.Success > 0 {
			reply.Code, reply.Message = m.GetDefaultReply(e.SuccessReply, 250)
		} else {
			reply.Code, reply.Message = m.GetDefaultReply(e.FailureReply, 550)
		}
	}

	// if defined code and length code
	if reply.Code > 0 {
		m.HandleReply(name, reply)
	}

	return reply.Success
}

func (m *MailServer) GetDefaultReply(config *Reply, default_code int) (code int, msg string) {
	if config != nil {
		code = config.Code
		msg = config.Message
	} else {
		code = default_code
		msg = ""
	}
	return
}

func (m *MailServer) HandleReply(verb string, reply *Reply) {
	// don't reply anything if code is empty
	if reply.Code != 0 {
		m.Reply(reply.Code, reply.Message)
	}
}

func (m *MailServer) Callback(name string, args ...string) *Reply {
	if cb, ok := m.CallbackMap[name]; ok == true {
		m.Context = cb.Context
		reply := cb.Code(args...)
		return reply
	}

	return &Reply{Success: 1, Code: -1}
}

func (m *MailServer) SetCallback(name string, code func(...string) *Reply, context ...string) {
	cb := &Callback{Code: code}
	if len(context) > 0 {
		cb.Context = context[0]
	}
	m.CallbackMap[name] = cb
}

func (m *MailServer) DefVerb(verb string, cb func(interface{}, ...string) bool) {
	m.Verb[strings.ToUpper(verb)] = cb
}

func (m *MailServer) UndefVerb(verb string) {
	if _, ok := m.Verb[verb]; ok == true {
		delete(m.Verb, verb)
	}
}

func (m *MailServer) ListVerb() []string {
	var keys []string
	for k := range m.Verb {
		keys = append(keys, k)
	}
	return keys
}

func (m *MailServer) NextInputTo(method_ref func(string) bool) func(string) bool {
	if method_ref != nil {
		m.NextInput = method_ref
	}
	return m.NextInput
}

func (m *MailServer) TellNextInputMethod(input string) bool {
	// calling the method and reinitialize. Note: we have to reinit
	// before calling the code, because code can resetup this variable.
	code := m.NextInput
	m.NextInput = nil
	rv := code(input)
	return rv
}

func (m *MailServer) Process() bool {
	in := m.In

	m.Banner()

	var buffer []byte
	ch := make(chan int)
	for {
		rv := 0

		buffer = make([]byte, 512*1024)
		read_size := 0

		go func() {
			var err error
			read_size, err = in.(*net.TCPConn).Read(buffer)
			if err != nil {
				ch <- 0
				return
			}
			ch <- 1
		}()
		if m.Options.IdleTimeout > 0 {
			go func() {
				time.Sleep(time.Second * time.Duration(m.Options.IdleTimeout))
				in.(*net.TCPConn).Close()
				ch <- 0
			}()
		}
		rv = <-ch

		if rv == 0 {
			// timeout, read error or connection closed
			break
		}

		// process all terminated lines
		// Note: Should accept only CRLF according to RFC. We accept
		// plain LFs anyway because its more liberal and works as well.
		newline_idx := strings.LastIndex(string(buffer), "\n")
		if newline_idx >= 0 {

			// one or more lines, terminated with \r?\n
			chunk := buffer[0 : newline_idx+1]

			// remaining buffer
			buffer = buffer[:newline_idx+1]

			rv := false
			if m.NextInput != nil {
				rv = m.TellNextInputMethod(string(chunk))
			} else {
				rv = m.CurProcessOperation(string(chunk))
			}

			// if rv is defined, we have to close the connection
			if rv == true {
				return rv
			}
		}

		// limit the size of lines to protect from excesssive memory consumption
		// (RFC specifies 1000 bytes including \r\n)
		if read_size > 1000 {
			m.MakeEvent(&Event{
				Name: "linetobig",
				SuccessReply: &Reply{
					Code:    552,
					Message: "line too long",
				},
			})
			return true
		}
	}

	return m.Timeout()
}

func (m *MailServer) ProcessOnce(operation string) bool {
	if m.NextInput != nil {
		return m.TellNextInputMethod(operation)
	} else {
		return m.CurProcessOperation(operation)
	}
}

func (m *MailServer) ProcessOperation(operation string) bool {
	verb, params := m.TokenizeCommand(operation)
	if params != "" && (strings.Contains(params, "\r") || strings.Contains(params, "\n")) {
		// doesn't support grouping of operations
		m.Reply(453, "Command received prior to completion of previous command sequence")
		return false
	}
	return m.ProcessCommand(verb, params)
}

func (m *MailServer) ProcessCommand(verb string, params string) bool {
	if action, ok := m.Verb[verb]; ok {
		return m.ExecAction(action, params)
	} else {
		m.Reply(500, "Syntax error: unrecognized command")
		return false
	}
}

func (m *MailServer) ExecAction(action func(interface{}, ...string) (close bool), params string) bool {
	return action(m, params)
}

func (m *MailServer) TokenizeCommand(line string) (verb string, params string) {
	line = strings.TrimRight(line, "\r\n")
	line = strings.TrimRight(line, "\n")
	line = strings.TrimSpace(line)
	t := strings.SplitN(line, " ", 2)
	if len(t) == 1 {
		return strings.ToUpper(t[0]), ""
	}
	return strings.ToUpper(t[0]), t[1]
}

func (m *MailServer) Reply(code int, msg string) {
	out := m.Out

	// tempo on error
	if code >= 400 && m.Options.ErrorSleepTime > 0 {
		time.Sleep(time.Duration(m.Options.ErrorSleepTime))
	}

	// default message
	if msg == "" {
		if code >= 400 {
			msg = "Failure"
		} else {
			msg = "Ok"
		}
	}

	// handle multiple lines
	lines := strings.Split(msg, "\n")

	for i := 0; i < len(lines); i++ {
		line := strings.TrimRight(lines[i], "\r")

		// RFC says that all lines but the last must
		// split the code and the message with a dash (-)
		var sep string
		if i == len(lines)-1 {
			sep = " "
		} else {
			sep = "-"
		}
		fmt.Fprintf(out, "%d%s%s\r\n", code, sep, line)
	}
}

func (m *MailServer) GetHostname() string {
	h, _ := os.Hostname()
	return h
}

func (m *MailServer) GetProtoname() string {
	return "NOPROTO"
}

func (m *MailServer) GetAppname() string {
	return "mailserver (Go)"
}

func (m *MailServer) Banner() {
	if m.BannerString == "" {
		hostname := m.GetHostname()
		protoname := m.GetProtoname()
		appname := m.GetAppname()

		str := ""
		if hostname != "" {
			str += hostname
		}
		if protoname != "" {
			str += protoname
		}
		if appname != "" {
			str += appname
		}
		str += "Service ready"
		m.BannerString = str
	}

	m.MakeEvent(&Event{
		Name: "banner",
		SuccessReply: &Reply{
			Code:    220,
			Message: m.BannerString,
		},
		FailureReply: &Reply{},
	})
}

func (m *MailServer) Timeout() bool {
	m.MakeEvent(&Event{
		Name: "timeout",
		SuccessReply: &Reply{
			Code:    421,
			Message: m.GetHostname() + " Timeout exceeded, closing transmission channel",
		},
	})

	return true
}
