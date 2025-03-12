package cmd

import (
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/creack/pty"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	// Shell command
	shellCmd = &cobra.Command{
		Use:   "shell [process-name]",
		Short: "Attach to process shell",
		Long:  `Attach to an interactive shell for a process.`,
		Run:   runShell,
	}
)

func runShell(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		logrus.Fatal("Process name is required")
	}

	name := args[0]

	// Attach shell to process
	ptmx, err := processManager.AttachShell(name)
	if err != nil {
		logrus.Fatalf("Failed to attach shell: %v", err)
	}
	defer processManager.DetachShell(name)

	// Set up terminal
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		logrus.Fatalf("Failed to set terminal to raw mode: %v", err)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	// Handle window size changes
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)
	go func() {
		for range ch {
			if err := pty.InheritSize(os.Stdin, ptmx); err != nil {
				logrus.Warnf("Failed to resize pty: %v", err)
			}
		}
	}()
	ch <- syscall.SIGWINCH // Initial resize

	// Set up bidirectional communication
	go func() {
		_, err := io.Copy(ptmx, os.Stdin)
		if err != nil {
			logrus.Warnf("Error copying stdin to pty: %v", err)
		}
	}()

	_, err = io.Copy(os.Stdout, ptmx)
	if err != nil {
		logrus.Warnf("Error copying pty to stdout: %v", err)
	}
}
