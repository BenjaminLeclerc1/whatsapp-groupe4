package logger

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"
)

var logFile *os.File

func Init(serviceName string) {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "dev"
	}

	var filename string
	if env == "prod" {
		date := time.Now().Format("2006-01-02")
		filename = fmt.Sprintf("logs/%s-%s-prod.log", serviceName, date)
	} else {
		filename = "logs/dev.log"
	}

	if err := os.MkdirAll("logs", 0755); err != nil {
		log.Fatalf("Impossible de créer le dossier logs : %v", err)
	}

	var err error
	logFile, err = os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Impossible d'ouvrir le fichier de log %s : %v", filename, err)
	}

	var level slog.Level
	if env == "prod" {
		level = slog.LevelError
	} else {
		level = slog.LevelInfo
	}

	handler := slog.NewJSONHandler(logFile, &slog.HandlerOptions{
		Level:     level,
		AddSource: true,
	})

	l := slog.New(handler).With("service", serviceName, "env", env)
	slog.SetDefault(l)

	slog.Info("Logger initialisé", "fichier", filename)
}

func Close() {
	if logFile != nil {
		logFile.Close()
	}
}

func Info(format string, v ...interface{}) {
	slog.Info(fmt.Sprintf(format, v...))
}

func Error(format string, v ...interface{}) {
	slog.Error(fmt.Sprintf(format, v...))
}

func Fatal(format string, v ...interface{}) {
	slog.Error(fmt.Sprintf(format, v...))
	os.Exit(1)
}
