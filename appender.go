// Copyright (C) 2017, ccpaging <ccpaging@gmail.com>.  All rights reserved.

package nxlog4go

import (
	"errors"
	"strings"
	"strconv"
)

// Various error codes.
var (
    ErrBadOption   = errors.New("invalid or unsupported option")
    ErrBadValue    = errors.New("invalid option value")
)

// Appender's properties
type AppenderProp struct {
	Name  string `xml:"name,attr"`
	Value string `xml:",chardata"`
}

/****** Appender ******/

// This is an interface for anything that should be able to write logs
type Appender interface {
	// Set option about the Layout. The options should be set as default.
	// Chainable.
	Set(name string, v interface{}) Appender

	// Set option about the Layout. The options should be set as default.
	// Checkable
	SetOption(name string, v interface{}) error

	// Init is called by the filters.Add() function to allow the appender
	// protocol to perform any initialization steps it needs. 
	// It should save the handle for future use, as well.
	Init()

	// This will be called to log a LogRecord message.
	Write(rec *LogRecord)

	// This should clean up anything lingering about the Appender, as it is called before
	// the Appender is removed.  Write should not be called after Close.
	Close()
}

// Configure appender by properties. checkable
func AppenderConfigure(app Appender, props []AppenderProp) bool {
	ok := true
	for _, prop := range props {
		err := app.SetOption(prop.Name, strings.Trim(prop.Value, " \r\n"))
		if err != nil {
			switch err {
			case ErrBadValue:
				LogLogError("appender", "Bad value of \"%s\"", prop.Name)
				ok = false
			case ErrBadOption:
				LogLogError("appender", "Unknown property \"%s\"", prop.Name)
			default:
			}
		}
	}
	return ok
}

// Parse a number with K/M/G suffixes based on thousands (1000) or 2^10 (1024)
func StrToNumSuffix(str string, mult int) int {
	num := 1
	if len(str) > 1 {
		switch str[len(str)-1] {
		case 'G', 'g':
			num *= mult
			fallthrough
		case 'M', 'm':
			num *= mult
			fallthrough
		case 'K', 'k':
			num *= mult
			str = str[0 : len(str)-1]
		}
	}
	parsed, _ := strconv.Atoi(str)
	return parsed * num
}
