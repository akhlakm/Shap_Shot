package argparser

import (
	"errors"
	"snap/internal/logger"
	"strconv"
	"strings"
)

type Parser struct {
	args []string
}

var initialized *Parser = nil

func Create(a []string) {
	initialized = &Parser{
		args: a,
	}
}

func GetParser() *Parser {
	return initialized
}

func (p *Parser) GetStr(position int) (string, error) {
	if position < len(p.args) {
		return p.args[position], nil
	}
	return "", errors.New("not enough arguments")
}

func (p *Parser) ReqStr(position int, errormsg string) string {
	if position < len(p.args) {
		return p.args[position]
	}
	logger.Error("required-arg", "not enough arguments", errormsg)
	return ""
}

func (p *Parser) GetInt(position int) (int, error) {
	if position < len(p.args) {
		return strconv.Atoi(p.args[position])
	}
	return -1, errors.New("not enough arguments")
}

func (p *Parser) HasFlag(flag string) bool {
	for _, v := range p.args {
		if strings.TrimSpace(v) == strings.TrimSpace(flag) {
			return true
		}
	}
	return false
}

// getInt(position, default=)
// needStr(position, errormsg)
// needInt(position, errormsg)
// hasFlag(flag)
// getKeyStr(flag, default=)
// getKeyInt(flag, default=)
// needKeyStr(flag, errormsg)
// needKeyInt(flag, errormsg)
