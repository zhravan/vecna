package safety

import "regexp"

var dangerous = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\brm\s+(-[rf]+\s+)?/`),
	regexp.MustCompile(`(?i)\bmkfs(?:\.|\s)`),
	regexp.MustCompile(`(?i)\bdd\s+.*\bof=/dev/`),
	regexp.MustCompile(`(?i)\bshutdown\b|\breboot\b|\bpoweroff\b`),
	regexp.MustCompile(`(?i)\bsystemctl\s+(disable|mask|stop)\b`),
	regexp.MustCompile(`(?i)\btruncate\s+.*--size\s+0`),
}

func IsDangerous(command string) bool { for _, re := range dangerous { if re.MatchString(command) { return true } }; return false }
