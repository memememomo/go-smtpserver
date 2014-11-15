package smtpserver

import (
//	"crypto/tls"
)

type StartTls struct {
	ExtensionBase
}

const (
	REPLY_READY_TO_START = 220
	REPLY_SYNTAX_ERROR   = 502
	REPLY_NOT_AVAILABLE  = 454
)

// https://tools.ietf.org/html/rfc2487

func (s *StartTls) Verb() map[string]func(interface{}, ...string) (close bool) {
	m := make(map[string]func(interface{}, ...string) (close bool))
	m["STARTTLS"] = s.Starttls
	return m
}

func (s *StartTls) Keyword() string {
	return "STARTTLS"
}

// Return a non undef to signal the server to close the socket.
func (s *StartTls) Starttls(obj interface{}, args ...string) (close bool) {
	switch esmtp := obj.(type) {
	case Esmtp:
		if len(args) > 0 {
			// No parameter verb
			esmtp.Reply(REPLY_SYNTAX_ERROR, "Syntax error (no parameters allowed)")
			return false
		}

		ssl_config := esmtp.Options.Ssl
		if ssl_config == nil {
			esmtp.Reply(REPLY_NOT_AVAILABLE, "TLS not available due to temporary reason")
			return false
		}

		esmtp.Reply(REPLY_READY_TO_START, "Ready to start TLS")

		/*
			ssl_socket := tls.Server(esmtp.Options.Socket, ssl_config)
			if err != nil {
				esmtp.Reply(REPLY_NOT_AVAILABLE, "TLS not available due to temporary reason ["+err+"]")
				return true // to single the server to close the socket
			}
		*/

		if cb, ok := esmtp.CallbackMap["STARTTLS"]; ok {
			cb.Code()
		}
	}
	return false
}
