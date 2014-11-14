package smtpserver

type Bit8mime struct {
	Extension
}

func (b *Bit8mime) Keyword() string {
	return "8BITMIME"
}

func (b *Bit8mime) Option() *SubOption {
	return &SubOption{"MAIL", "BODY", b.OptionMailBody}
}

func (b *Bit8mime) OptionMailBody(verb string, address string, key string, value string) {
	return
}