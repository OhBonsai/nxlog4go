// Copyright (C) 2017, ccpaging <ccpaging@gmail.com>.  All rights reserved.

package nxlog4go

import (
	"strings"
)

type NameValue struct {
	Name  string `xml:"name,attr" json:"name"`
	Value string `xml:",chardata" json:"value"`
}

type FilterConfig struct {
	Enabled string `xml:"enabled,attr" json:"enabled"`
	Tag     string `xml:"tag" json:"tag"`
	Type    string `xml:"type" json:"type"`
	Level   string `xml:"level" json:"level"`
	Pattern string `xml:"format" json:"format"`
	Properties []NameValue `xml:"property" json:"properties"`
}

type LoggerConfig struct {
	Filters []FilterConfig `xml:"filter" json:"filters"`
}

func loadLogLog(level Level, pattern string) {
	if level >= SILENT {
		LogLogTrace("Disable loglog for level \"%d\"", level)
	} else {
		loglog := GetLogLog().Set("level", level)
		if pattern != "" {
			loglog.Set("pattern", pattern)
		}
	}
}

func loadStdout(log *Logger, level Level, pattern string) {
	if level >= SILENT {
		LogLogTrace("Disable stdout for level \"%d\"", level)
		log.SetOutput(nil)
	} else {
		log.Set("level", level)
		if pattern != "" {
			log.Set("pattern", pattern)
		}
	}
}

func loadAppender(level Level, typ string, props []NameValue) Appender {
	if level >= SILENT {
		LogLogTrace("Disable \"%s\" for level \"%d\"", typ, level)
		return nil
	}

	newFunc := GetAppenderNewFunc(typ)
	if newFunc == nil {
		LogLogWarn("Unknown appender type \"%s\"", typ)
		return nil
	}

	appender := newFunc()
	if appender == nil {
		return nil
	}

	for _, prop := range props {
		v := strings.Trim(prop.Value, " \r\n")
		if err := appender.SetOption(prop.Name, v); err != nil {
			LogLogWarn("%s. %s: %s", err.Error(), prop.Name, v)
		}
	}
	return appender
}

// Load configuration; see examples/example.xml for documentation
func LoadConfiguration(log *Logger, lc *LoggerConfig) {
	if lc == nil {
		LogLogWarn("Logger configuration is NIL")
		return
	}
	if len(lc.Filters) <= 0 {
		LogLogTrace("Filters configuration is NIL")
		return
	}

	filters := make(Filters)
	for _, fc := range lc.Filters {
		if fc.Tag == "" && fc.Type == "" {
			LogLogWarn("Missing tag and type")
			continue
		} else if fc.Tag == "" { 
			fc.Tag = fc.Type
		} else if fc.Type == "" {
			fc.Type = strings.ToLower(fc.Tag)
		}

		if enabled, err := ToBool(fc.Enabled); err != nil || !enabled{
			LogLogTrace("Disable \"%s\" for %s", fc.Tag, err)
			continue
		} 

		level := GetLevel(fc.Level)

		switch fc.Type {
		case "loglog":
			loadLogLog(level, fc.Pattern)
		case "stdout":
			loadStdout(log, level, fc.Pattern)
		default:
			appender := loadAppender(level, fc.Type, fc.Properties)
			if appender != nil {
				LogLogTrace("Succeeded loading appender \"%s\"", fc.Tag)
				filters.Add(fc.Tag, level, appender)
			}
		}
	}

	log.SetFilters(filters)
}