package mail

import (
	"bytes"
	"fmt"
	"net/smtp"
	"strconv"
	"text/template"
)

type EmailUser struct {
	Username    string
	Password    string
	EmailServer string
	Port        int
}

type SmtpTemplateData struct {
	From    string
	To      string
	Subject string
	Body    string
}

const emailTemplate = `From: {{.From}}
To: {{.To}}
Subject: {{.Subject}}

{{.Body}}

Sincerely,

{{.From}}`

func Mail(emailUser EmailUser, smtpData SmtpTemplateData) {

	auth := smtp.PlainAuth("", emailUser.Username, emailUser.Password, emailUser.EmailServer)

	var err error
	var doc bytes.Buffer

	t := template.New("emailTemplate")
	t, err = t.Parse(emailTemplate)
	if err != nil {
		fmt.Println("error trying to parse mail template")
	}
	err = t.Execute(&doc, smtpData)
	if err != nil {
		fmt.Println("error trying to execute mail template")
	}

	err = smtp.SendMail(emailUser.EmailServer+":"+strconv.Itoa(emailUser.Port),
		auth,
		emailUser.Username,
		[]string{smtpData.To},
		doc.Bytes())
	if err != nil {
		fmt.Println("ERROR: attempting to send a mail ", err)
	}
}
