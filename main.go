package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
)

var (
	logger       *log.Logger
	debugLogger  *log.Logger
	repoPath     string
	versionMajor int = 0
	versionMinor int = 1
	versionPatch int = 0
)

func main() {
	// Need to send the result code to the OS but also need to support 'defer'
	// os.Exit would finish before any defers, so wrap everything in mainImpl()
	os.Exit(MainImpl())

}

func MainImpl() int {

	// Generic panic handler so we get stack trace
	defer func() {
		if e := recover(); e != nil {
			outputf("Panic: %v\n", e)
			outputf(string(debug.Stack()))
			os.Exit(99)
		}

	}()

	// Get set up
	cfg := LoadConfig()
	err := initLogging(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "git-lfs-ssh-serve was unable to initialise logging: %v (continuing anyway)\n", err)
	}

	if cfg.BasePath == "" {
		outputf("Missing required configuration setting: base-path\n")
		return 12
	}
	if !dirExists(cfg.BasePath) {
		outputf("Invalid value for base-path: %v\nDirectory must exist.\n", cfg.BasePath)
		return 14
	}
	// Change to the base path directory so filepath.Clean() can work with relative dirs
	os.Chdir(cfg.BasePath)

	if cfg.DeltaCachePath != "" && !dirExists(cfg.DeltaCachePath) {
		// Create delta cache if doesn't exist, use same permissions as base path
		s, err := os.Stat(cfg.BasePath)
		if err != nil {
			outputf("Invalid value for base-path: %v\nCannot stat: %v\n", cfg.BasePath, err.Error())
			return 16
		}
		err = os.MkdirAll(cfg.DeltaCachePath, s.Mode())
		if err != nil {
			outputf("Error creating delta cache path %v: %v\n", cfg.DeltaCachePath, err.Error())
			return 16
		}
	}

	// Get path argument
	if len(os.Args) < 2 {
		outputf("Path argument missing, cannot continue\n")
		return 18
	}
	repoPath = filepath.Clean(os.Args[1])
	if filepath.IsAbs(repoPath) && !cfg.AllowAbsolutePaths {
		outputf("Path argument %v invalid, absolute paths are not allowed by this server\n", repoPath)
		return 18
	}

	return Serve(os.Stdin, os.Stdout, os.Stderr, cfg, repoPath)
}

func dirExists(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		return false
	}

	return fi.IsDir()
}

func initLogging(cfg *Config) error {
	if cfg.LogFile != "" {

		// O_APPEND is safe to use for multiple processes, make sure writeable by all though
		logf, err := os.OpenFile(cfg.LogFile, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
		if err != nil {
			return err
		}
		logger = log.New(logf, "", log.Ldate|log.Ltime)
		if cfg.DebugLog {
			debugLogger = logger
		}
	}
	return nil
}

// Helper function to log (no need to worry about nil loggers, prefixing etc)
func logPrintf(l *log.Logger, format string, v ...interface{}) {
	if l != nil {
		// Prefix message with repo root (this is cached for efficiency)
		// We don't add this to the Logger prefix in New() because this prefixes before the timestamp & other
		// flag-based fields, which means things don't line up nicely in the log
		newformat := `[%d][%v]: ` + format
		newargs := []interface{}{os.Getpid(), repoPath}
		newargs = append(newargs, v...)

		l.Printf(newformat, newargs...)
	}
}

// Helper function to log a regular message AND output to stderr
func outputf(format string, v ...interface{}) {
	fmt.Fprintf(os.Stderr, format, v...)
	logPrintf(logger, format, v...)
}

// Helper function to log a regular message
func logf(format string, v ...interface{}) {
	logPrintf(logger, format, v...)
}

// Helper function to log a debug message
func debugf(format string, v ...interface{}) {
	logPrintf(debugLogger, format, v...)
}
