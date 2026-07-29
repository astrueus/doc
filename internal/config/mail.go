package config

type SmtpConf struct {
	EnableMail   bool
	MailNumber   int
	SmtpUserName string
	SmtpHost     string
	SmtpPassword string
	SmtpPort     int
	FormUserName string
	MailExpired  int
	Secure       string
}

func GetMailConfig() *SmtpConf {
	g := MustGlobal()
	secure := g.Mail.Secure
	if secure != "NONE" && secure != "LOGIN" && secure != "SSL" {
		secure = "NONE"
	}
	return &SmtpConf{
		EnableMail:   g.Mail.Enable,
		MailNumber:   g.Mail.MailNumber,
		SmtpUserName: g.Mail.SMTPUserName,
		SmtpHost:     g.Mail.SMTPHost,
		SmtpPassword: g.Mail.SMTPPassword,
		FormUserName: g.Mail.FormUserName,
		SmtpPort:     g.Mail.SMTPPort,
		MailExpired:  g.Mail.MailExpired,
		Secure:       secure,
	}
}
