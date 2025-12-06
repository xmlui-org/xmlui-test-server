package localsvr

import (
	"errors"
	"fmt"
	"strings"
)

type ErrorStyle string

func (es ErrorStyle) ErrorMessage(msg, devInfo string, err error) string {
	switch es {
	case DevelopmentStyle:
		msg = fmt.Sprintf("%s [%s]; %s", msg, devInfo, err.Error())
	case PresentationStyle:
		msg = fmt.Sprintf("%s; check logs if you have server access.", msg)
	}
	return msg
}

const DefaultErrorStyle = DevelopmentStyle

const (
	InvalidErrorStyle ErrorStyle = ""
	DevelopmentStyle  ErrorStyle = "dev"
	PresentationStyle ErrorStyle = "pres"
)

var (
	ErrInvalidErrorStyle = errors.New("invalid error style")
)

func ParseErrorStyle(style string) (es ErrorStyle, err error) {
	es = ErrorStyle(strings.ToLower(style))
	switch es {
	case DevelopmentStyle, PresentationStyle:
		// S'all good, man
	default:
		es = InvalidErrorStyle
		err = ErrInvalidErrorStyle
	}
	return es, err
}
