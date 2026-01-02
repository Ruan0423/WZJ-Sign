package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

var MountAllowed = map[string]bool{}

func LoadMountAllowed() error {
	MountAllowed = map[string]bool{}
	if _, err := os.Stat("mount_allow.txt"); os.IsNotExist(err) {
		return nil
	}
	f, err := os.Open("mount_allow.txt")
	if err != nil {
		return err
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line != "" {
			MountAllowed[line] = true
		}
	}
	if err := s.Err(); err != nil {
		return err
	}
	return nil
}

func SaveMountAllowed() error {
	f, err := os.Create("mount_allow.txt")
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	for name := range MountAllowed {
		fmt.Fprintln(w, name)
	}
	return w.Flush()
}

func ToggleMountAllow(name string) (enabled bool, err error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return false, fmt.Errorf("name empty")
	}
	if MountAllowed[name] {
		delete(MountAllowed, name)
		err = SaveMountAllowed()
		return false, err
	}
	MountAllowed[name] = true
	err = SaveMountAllowed()
	return true, err
}