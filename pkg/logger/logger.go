// Copyright (C) 2019 SAP SE or an SAP affiliate company. All rights reserved.
// This file is licensed under the Apache Software License, v. 2 except as
// noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package logger

import (
	"github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type Logger struct {
	baseLogger *logrus.Logger
	component  string
	entry      entry
}
type entry struct {
	*logrus.Entry
}

func GetCallersPackagename() string {
	return filepath.Base(filepath.Dir(getCallersFile()))
}
func GetCallersFilename() string {
	var file = getCallersFile()
	return strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
}
func getCallersFile() string {
	var _, file, _, ok = runtime.Caller(2)
	if !ok {
		file = "---UNKNOWN---"
	}
	return file
}

func NewLogger() *Logger {
	var baseLogger = logrus.New()
	var logger = &Logger{baseLogger: baseLogger}
	logger.baseLogger.Formatter = &logrus.JSONFormatter{}
	logger.baseLogger.Out = os.Stdout
	logger.baseLogger.Level = logrus.TraceLevel
	logger.entry = *logger.log()
	return logger
}
func NewComponentLogger(componentName string) *Logger {
	var logger = NewLogger()
	logger.SetComponent(componentName)
	return logger
}

func (l *Logger) SetComponent(name string) {
	l.component = name
	l.entry = *l.log()
}
func (l *Logger) GetComponent() string {
	return l.component
}

func (l *Logger) log() *entry {
	var fields = logrus.Fields{}
	if len(l.component) > 0 {
		fields["component"] = l.component
	}
	return &entry{l.baseLogger.WithFields(fields)}
}
func (l *Logger) logWithAdditionalFields(additionalFields map[string]interface{}) *entry {
	return &entry{l.log().WithFields(additionalFields)}
}

func (e *entry) prependStrToFormat(prefix, main string) string {
	var format = main
	if len(prefix) > 0 {
		format = prefix + " " + main
	}
	return format
}
func (e *entry) prependStrToArgs(prefix string, main ...interface{}) []interface{} {
	var args = main
	if len(prefix) > 0 {
		args = append([]interface{}{prefix}, main...)
	}
	return args
}

type Level int

const (
	debugLevel Level = 2 + iota
	infoLevel
	warnLevel
	errorLevel
	fatalLevel
	panicLevel
)

var levelFormats = map[Level]string{
	debugLevel: "[DEBUG_INFO]",
	infoLevel:  "[INFO]",
	warnLevel:  "[WARN]",
	errorLevel: "[ERROR]",
	fatalLevel: "[FATAL]",
	panicLevel: "[PANIC]",
}

func (e *entry) f(level Level, format string, args ...interface{}) {
	var functions = map[Level]func(string, ...interface{}){
		debugLevel: e.Debugf,
		infoLevel:  e.Infof,
		warnLevel:  e.Warnf,
		errorLevel: e.Errorf,
		fatalLevel: e.Fatalf,
		panicLevel: e.Panicf,
	}
	functions[level](e.prependStrToFormat(levelFormats[level], format), args...)
}
func (e *entry) ln(level Level, args ...interface{}) {
	var functions = map[Level]func(...interface{}){
		debugLevel: e.Debugln,
		infoLevel:  e.Infoln,
		warnLevel:  e.Warnln,
		errorLevel: e.Errorln,
		fatalLevel: e.Fatalln,
		panicLevel: e.Panicln,
	}
	functions[level](e.prependStrToArgs(levelFormats[level], args...)...)
}

func (l *Logger) HandleCrash() {
	if r := recover(); r != nil {
		var entry = l.logWithAdditionalFields(map[string]interface{}{
			"handleCrash": "autoLog",
		})
		// guard against too large logs
		const size = 64 << 10
		stacktrace := make([]byte, size)
		stacktrace = stacktrace[:runtime.Stack(stacktrace, false)]
		if _, ok := r.(string); ok {
			entry.f(panicLevel, "%s\n%s", r, stacktrace)
		} else {
			entry.f(panicLevel, "%#v (%v)\n%s", r, r, stacktrace)
		}
		panic(r)
	}
}

func (l *Logger) Debugf(format string, args ...interface{}) {
	l.entry.f(debugLevel, format, args...)
}
func (l *Logger) Debugln(args ...interface{}) {
	l.entry.ln(debugLevel, args...)
}

func (l *Logger) Infof(format string, args ...interface{}) {
	l.entry.f(infoLevel, format, args...)
}
func (l *Logger) Infoln(args ...interface{}) {
	l.entry.ln(infoLevel, args...)
}

func (l *Logger) Warnf(format string, args ...interface{}) {
	l.entry.f(warnLevel, format, args...)
}
func (l *Logger) Warnln(args ...interface{}) {
	l.entry.ln(warnLevel, args...)
}

func (l *Logger) Errorf(format string, args ...interface{}) {
	l.entry.f(errorLevel, format, args...)
}
func (l *Logger) Errorln(args ...interface{}) {
	l.entry.ln(errorLevel, args...)
}

func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.entry.f(fatalLevel, format, args...)
}
func (l *Logger) Fatalln(args ...interface{}) {
	l.entry.ln(fatalLevel, args...)
}
