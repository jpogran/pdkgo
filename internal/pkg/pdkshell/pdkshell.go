package pdkshell

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/rs/zerolog/log"
)

type PDKInfo struct {
	RubyVersion      string
	InstallDirectory string
	RubyExecutable   string
	PDKExecutable    string
	CertDirectory    string
	CertPemFile      string
}

func Execute(args []string) {
	i := getPDKInfo()
	executable := buildExecutable(i.RubyExecutable)
	args = buildCommandArgs(args, i.RubyExecutable, i.PDKExecutable)
	env := os.Environ()
	env = append(env, fmt.Sprintf("SSL_CERT_DIR=%s", i.CertDirectory), fmt.Sprintf("SSL_CERT_FILE=%s", i.CertPemFile))
	cmd := &exec.Cmd{
		Path:   executable,
		Args:   args,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Env:    env,
	}

	log.Trace().Msgf("args: %s", args)
	if err := cmd.Run(); err != nil {
		log.Fatal().Msgf("pdk failed with '%s'\n", err)
	}
}

func getPDKInfo() *PDKInfo {
	rubyVersion := "2.4.10"
	installDir, err := getPDKInstallDirectory(true)
	if err != nil {
		log.Fatal().Msgf("error: %v", err)
	}

	i := &PDKInfo{
		RubyVersion:      rubyVersion,
		InstallDirectory: installDir,
		RubyExecutable:   filepath.Join(installDir, "private", "ruby", rubyVersion, "bin", "ruby"),
		PDKExecutable:    filepath.Join(installDir, "private", "ruby", rubyVersion, "bin", "pdk"),
		CertDirectory:    filepath.Join(installDir, "ssl", "certs"),
		CertPemFile:      filepath.Join(installDir, "ssl", "cert.pem"),
	}
	return i
}

func buildExecutable(rubyexe string) (executable string) {
	executable = rubyexe
	if runtime.GOOS == "windows" {
		exe, _ := exec.LookPath("cmd.exe")
		executable = exe
	}
	return executable
}

func buildCommandArgs(args []string, rubyexe, pdkexe string) []string {
	var a []string
	if runtime.GOOS == "windows" {
		a = append(a, "/c")
	}
	a = append(a, rubyexe, "-S", "--", pdkexe)
	a = append(a, args...)
	return a
}
