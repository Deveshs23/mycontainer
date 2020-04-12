package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/docker/docker/pkg/reexec"
)

func init() {
	reexec.Register("nsInitialisation", nsInitialisation)
	if reexec.Init() {
		os.Exit(0)
	}
}
func nsInitialisation() {
	newrootPath := os.Args[1]
	if err := pivotRoot(newrootPath); err != nil {
		fmt.Printf("Error running pivot_root %s\n", err)
		os.Exit(1)
	}

	if err := mountProc(newrootPath); err != nil {
		fmt.Printf("Error Mountung")
	}
	fmt.Printf("\n>> namespace setup code goes here <<\n\n")
	nsRun()
}
func nsRun() {
	cmd := exec.Command("/bin/bash")

	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout

	cmd.Env = []string{"PS1=-[ns-process]- #"}
	if err := cmd.Run(); err != nil {
		fmt.Printf("Error runing the command /bin/bash %s\n", err)
		os.Exit(1)
	}
}

// Call three system call clone, setns, unshare
func main() {
	type SysProcIDMap struct {
		ContainerID int
		HostID      int
		Size        int
	}
	var rootfsPath string

	cmd := reexec.Command("nsInitialisation", rootfsPath)
	cmd = exec.Command("/bin/bash")
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr

	cmd.Env = []string{"PS1=-[ns-process]- # "}
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS |
			syscall.CLONE_NEWNS |
			syscall.CLONE_NEWIPC |
			syscall.CLONE_NEWNET |
			syscall.CLONE_NEWPID |
			syscall.CLONE_NEWUSER,
		UidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: 0,
				HostID:      os.Getuid(),
				Size:        1,
			},
		},
		GidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: 0,
				HostID:      os.Getgid(),
				Size:        1,
			},
		},
	}

	if err := cmd.Run(); err != nil {
		fmt.Printf("Error running the /bin/bash command %s\n", err)
		os.Exit(1)
	}
}

func pivotRoot(newroot string) error {
	putold := filepath.Join(newroot, "/.pivot_root")
	if err := syscall.Mount(
		newroot,
		newroot,
		"",
		syscall.MS_BIND|syscall.MS_REC,
		"",
	); err != nil {
		return err
	}
	// Create old put Directory
	if err := os.MkdirAll(putold, 0700); err != nil {
		return err
	}

	if err := syscall.PivotRoot(newroot, putold); err != nil {
		return err
	}

	if err := os.Chdir("/"); err != nil {
		return err
	}

	putold = "/.pivot_root"
	if err := syscall.Unmount(
		putold,
		syscall.MNT_DETACH,
	); err != nil {
		return err
	}

	if err := os.RemoveAll(putold); err != nil {
		return err
	}

	return nil
}

func mountProc(newroot string) error {
	source := "proc"
	target := filepath.Join(newroot, "/proc")
	fstype := "proc"
	flag := 0
	data := ""

	os.MkdirAll(target, 0755)
	if err := syscall.Mount(
		source,
		target,
		fstype,
		uintptr(flag),
		data,
	); err != nil {
		return err
	}
	return nil
}
