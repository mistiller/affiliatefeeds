package email

import (
	"fmt"
	"net/smtp"
	"strings"
)

type sender struct {
	adress string
	pw     string
	host   string
}

// Mail delivers an email from a sender to a recipient
type Mail struct {
	sender     sender
	recipients []string
	message    []byte
	server     string
}

// NewEmail creates a sender with adress, host, and password
func NewEmail(fromAdress, fromServer, fromPassword string) (m *Mail) {
	m = new(Mail)
	m.server = fromServer
	host := strings.Split(fromServer, ":")[0]
	m.sender = sender{
		adress: fromAdress,
		pw:     fromPassword,
		host:   host,
	}

	return m
}

// Send composes the actual message and delivers it via the stillgrove mail server
func (m *Mail) Send(subject, body string, to []string) error {
	auth := smtp.PlainAuth(
		"",
		m.sender.adress,
		m.sender.pw,
		m.sender.host,
	)

	msg := m.compose(subject, body)
	err := smtp.SendMail(m.server, auth, "no-reply@stillgrove.com", to, msg)

	return err
}

func (m *Mail) compose(subjectLine, body string) []byte {
	rec := "To: "
	for i := range m.recipients {
		rec += m.recipients[i]
		if i < len(m.recipients)-1 {
			rec += ","
		}
	}

	text := fmt.Sprintf("%s\r\nSubject: %s\r\n%s\r\n", rec, subjectLine, body)

	return []byte(text)
}
