package smtpserver

type Extension interface {
	Init()
	Verb()
	Reyword() string
	Parameter()
	Option() *SubOption
	Reply()
}

func (e *Extension) Init(parent interface{}) *Extension {
	return e
}

func (e *Extension) Verb() {
}

func (e *Extension) Keyword() string {
	return "XNOTOVERLOADED"
}

func (e *Extension) Parameter() {
}

func (e *Extension) Option() *SubOption {
	return nil
}

func (e *Extension) Reply() {
}
