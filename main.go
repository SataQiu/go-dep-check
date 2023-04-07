package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
)

var (
	GOPATH = os.Getenv("GOPATH")
)

func main() {
	file, err := os.OpenFile("missing_dep.txt", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	do("", file)
}

func do(dirPath string, out *os.File) {
	modFile := filepath.Join(dirPath, "go.mod")
	if _, err := os.Stat(modFile); err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(os.Stdout, "can not find go.mod from %v, skip\n", dirPath)
			return
		}
	}

	file, err := ioutil.ReadFile(modFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	f, err := modfile.Parse("go.mod", file, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing file: %v\n", err)
		os.Exit(1)
	}

	replace := map[string]module.Version{}
	for _, r := range f.Replace {
		replace[r.Old.Path] = r.New
	}

	for _, r := range f.Require {
		m := r.Mod
		if p, ok := replace[m.Path]; ok {
			m.Path = p.Path
			m.Version = p.Version
		}

		dep := fmt.Sprintf("%s@%s", m.Path, m.Version)
		cmd := exec.Command("go", "mod", "download", dep)
		output, err := cmd.Output()
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to download %v, error %v, output %v\n", dep, err, string(output))
			out.WriteString(fmt.Sprintf("%s\n", m.Path))
		} else {
			fmt.Fprintf(os.Stdout, "download %v succeed\n", dep)
			subDir := filepath.Join(GOPATH, "pkg", "mod", dep)
			do(subDir, out)
		}
	}
}
