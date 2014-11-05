package smtpserver

type Extension interface {
	Init()
	Verb() map[string]func(...string) (close bool)
	Reyword() string
	Parameter()
	Option() []*SubOption
	Reply() map[string]func(string, *Reply) (int, string)
	ExtendMode(bool)
}
