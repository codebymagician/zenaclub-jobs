package utils

import (
	"log"
)

// Ported verbatim from ringly-jobs/crm-jobs/utils/logger.go -- same shape,
// so a future job copied over from crm-jobs needs no adjustment here.
type Logger struct {
	level string
}

func NewLogger(level string) *Logger {
	return &Logger{level: level}
}

func (l *Logger) Info(msg string) {
	if l.level == "INFO" || l.level == "DEBUG" {
		log.Println("[INFO]", msg)
	}
}

func (l *Logger) Debug(msg string) {
	if l.level == "DEBUG" {
		log.Println("[DEBUG]", msg)
	}
}

func (l *Logger) Error(msg string) {
	log.Println("[ERROR]", msg)
}
