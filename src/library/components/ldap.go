package components

import (
	"fmt"
	ldap "github.com/go-ldap/ldap/v3"
	"github.com/linclin/gopub/src/library/config"
	"github.com/linclin/gopub/src/library/logger"
)

type Ldap struct {
	link *ldap.Conn
}

func new_ldap() (l Ldap) {
	l.connect()
	return l
}

func (l *Ldap) connect() (err bool) {
	ldapHost := config.String("ldapHost")
	ldapPort, _ := config.Int("ldapPort")
	link, e := ldap.Dial("tcp", fmt.Sprintf("%s:%d", ldapHost, ldapPort))

	if e != nil {
		logger.Info("ldap connect error")
		return false
	}

	e = link.Bind(config.String("ldapManagerDn"), config.String("ldapManagerPassword"))
	if e != nil {
		logger.Info("ldap login error")
		return false
	}
	l.link = link
	return true
}
