package config

import (
	"errors"
	"fmt"
)

var (
	ErrConfigNotFound   = errors.New("config file not found")
	ErrConfigSyntax     = errors.New("config syntax error")
	ErrConfigValidation = errors.New("config validation error")
)

type ConfigSyntaxError struct {
	Line    int
	Column  int
	Message string
}

func (e *ConfigSyntaxError) Error() string {
	return fmt.Sprintf("syntax error at line %d, column %d: %s", e.Line, e.Column, e.Message)
}

func (e *ConfigSyntaxError) Unwrap() error {
	return ErrConfigSyntax
}
