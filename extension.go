package smtpserver

type Extension interface {
	Init(*Esmtp) Extension
	Verb() map[string]func(interface{}, ...string) (close bool)
	Keyword() string
	Parameter() []string
	Option() []*SubOption
	Reply() map[string]func(string, *Reply) (int, string)
	SetExtendMode(bool)
}

type ExtensionBase struct {
	ExtendMode bool
	Parent     *Esmtp
}

func (e *ExtensionBase) Init(s *Esmtp) Extension {
	return e
}

func (e *ExtensionBase) Verb() map[string]func(interface{}, ...string) (close bool) {
	return nil
}

func (e *ExtensionBase) Keyword() string {
	return "XNOTOVERLOADED"
}

func (e *ExtensionBase) Parameter() []string {
	return nil
}

func (e *ExtensionBase) Option() []*SubOption {
	return nil
}

func (e *ExtensionBase) Reply() map[string]func(string, *Reply) (int, string) {
	return nil
}

func (e *ExtensionBase) SetExtendMode(mode bool) {
	e.ExtendMode = mode
}
