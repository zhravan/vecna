package sftp

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// scpPath returns the path to the scp binary. On Windows, if scp is not in PATH, tries OpenSSH install path.
func scpPath() string {
	if path, err := exec.LookPath("scp"); err == nil {
		return path
	}
	if runtime.GOOS == "windows" {
		// Windows 10+ optional OpenSSH client default location
		const winOpenSSH = "C:\\Windows\\System32\\OpenSSH\\scp.exe"
		if _, err := os.Stat(winOpenSSH); err == nil {
			return winOpenSSH
		}
	}
	return "scp"
}

// RunSCP runs scp for push (local → remote) or pull (remote → local). Returns combined output.
func RunSCP(h HostConnection, direction string, localPath, remotePath string) (output string, err error) {
	port := h.Port
	if port == 0 {
		port = 22
	}
	args := []string{"-P", fmt.Sprintf("%d", port)}
	if h.IdentityFile != "" {
		args = append(args, "-i", expandPath(h.IdentityFile))
	}
	remote := fmt.Sprintf("%s@%s:%s", h.User, h.Hostname, remotePath)
	if direction == "push" {
		args = append(args, localPath, remote)
	} else {
		args = append(args, remote, localPath)
	}
	cmd := exec.Command(scpPath(), args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// RunSCPHostToHost copies from source host to dest host (stream goes through local machine).
// Both source and dest are remote paths: user@host:path.
func RunSCPHostToHost(source HostConnection, sourcePath string, dest HostConnection, destPath string) (output string, err error) {
	srcPort := source.Port
	if srcPort == 0 {
		srcPort = 22
	}
	args := []string{"-P", fmt.Sprintf("%d", srcPort)}
	if source.IdentityFile != "" {
		args = append(args, "-i", expandPath(source.IdentityFile))
	}
	src := fmt.Sprintf("%s@%s:%s", source.User, source.Hostname, sourcePath)
	destStr := fmt.Sprintf("%s@%s:%s", dest.User, dest.Hostname, destPath)
	args = append(args, src, destStr)
	cmd := exec.Command(scpPath(), args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// HostConnection holds the connection details for building an sftp command.
type HostConnection struct {
	User         string
	Hostname     string
	Port         int
	IdentityFile string
}

// BuildCommand returns the sftp command and args for the given connection.
// Caller can run it (e.g. in a new terminal) or display it.
func BuildCommand(h HostConnection) (name string, args []string) {
	port := h.Port
	if port == 0 {
		port = 22
	}
	args = append(args, "-P", fmt.Sprintf("%d", port))
	if h.IdentityFile != "" {
		path := expandPath(h.IdentityFile)
		args = append(args, "-i", path)
	}
	args = append(args, fmt.Sprintf("%s@%s", h.User, h.Hostname))
	return "sftp", args
}

func expandPath(p string) string {
	if strings.HasPrefix(p, "~") {
		home, _ := os.UserHomeDir()
		rest := strings.TrimLeft(p[1:], `/\`)
		return filepath.Join(home, rest)
	}
	return p
}

// RunInNewTerminal runs the sftp command in a new terminal window so the user can interact with it.
// On macOS uses Terminal.app via osascript; on Linux tries gnome-terminal or xterm; on Windows uses cmd start.
// Returns the command string and any error (e.g. if no terminal could be launched).
func RunInNewTerminal(h HostConnection) (cmdStr string, err error) {
	name, args := BuildCommand(h)
	cmdStr = name + " " + strings.Join(quoteArgs(args), " ")

	switch runtime.GOOS {
	case "darwin":
		// Tell Terminal.app to run the command in a new window
		script := `tell application "Terminal" to do script "` + escapeForAppleScript(strings.Join(args, " ")) + `"`
		// Run: osascript -e 'tell application "Terminal" to do script "sftp -P 22 user@host"'
		cmd := exec.Command("osascript", "-e", script)
		if err := cmd.Start(); err != nil {
			return cmdStr, err
		}
		go cmd.Wait()
		return cmdStr, nil
	case "linux":
		// Try gnome-terminal first, then xterm
		if cmd := exec.Command("gnome-terminal", append([]string{"--", name}, args...)...); cmd.Start() == nil {
			go cmd.Wait()
			return cmdStr, nil
		}
		xtermCmd := name + " " + strings.Join(quoteArgs(args), " ")
		if cmd := exec.Command("xterm", "-e", xtermCmd); cmd.Start() == nil {
			go cmd.Wait()
			return cmdStr, nil
		}
		return cmdStr, fmt.Errorf("could not open terminal (tried gnome-terminal, xterm)")
	case "windows":
		// Open new console window with sftp (requires OpenSSH client, e.g. built-in on Windows 10+)
		startArgs := append([]string{"/c", "start", "Vecna SFTP", name}, args...)
		cmd := exec.Command("cmd", startArgs...)
		if err := cmd.Start(); err != nil {
			return cmdStr, err
		}
		go cmd.Wait()
		return cmdStr, nil
	default:
		return cmdStr, fmt.Errorf("open SFTP in new terminal not supported on %s", runtime.GOOS)
	}
}

func quoteArgs(args []string) []string {
	out := make([]string, len(args))
	for i, a := range args {
		if strings.ContainsAny(a, " \t'\"") {
			out[i] = fmt.Sprintf("%q", a)
		} else {
			out[i] = a
		}
	}
	return out
}

func escapeForAppleScript(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}
