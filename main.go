package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
)

var outputFile = flag.String("o", "", "the file path to output result")
var filterDomains = flag.String("filters", "", "set filter domains for the checker")

var gopath = os.Getenv("GOPATH")

func main() {
	flag.Parse()

	fmt.Printf("Detected GOPATH: %v\n", gopath)

	filters := []string{}
	if len(*filterDomains) > 0 {
		for _, s := range strings.Split(*filterDomains, ",") {
			s = strings.TrimSpace(s)
			if len(s) > 0 {
				filters = append(filters, s)
			}
		}
	}

	result := map[string]struct{}{}
	curDir, _ := os.Getwd()
	process(curDir, filters, result)

	if len(result) > 0 {
		fmt.Println("Found Missing Dependencies:")
		for path := range result {
			fmt.Println(path)
		}
	} else {
		fmt.Println("Congratulations, No missing dependencies found.")
	}

	if len(*outputFile) > 0 {
		file, err := os.OpenFile(*outputFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			panic(err)
		}
		defer file.Close()
		for path := range result {
			file.WriteString(fmt.Sprintf("%s\n", path))
		}
	}
}

func process(dirPath string, filters []string, result map[string]struct{}) error {
	return filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		if filepath.Base(path) == "go.mod" {
			content, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}

			f, err := modfile.ParseLax(path, content, nil)
			if err != nil {
				return err
			}

			replace := map[string]module.Version{}
			for _, r := range f.Replace {
				replace[r.Old.Path] = r.New
			}

			for _, r := range f.Require {
				m := r.Mod
				if p, ok := replace[m.Path]; ok {
					m = p
				}

				if _, ok := result[m.Path]; ok {
					continue
				}

				related := !(len(filters) > 0)
				if !related {
					for _, domain := range filters {
						if strings.Contains(m.Path, domain) {
							related = true
						}
					}
				}
				if !related {
					continue
				}

				dep := fmt.Sprintf("%s@%s", m.Path, m.Version)
				cmd := exec.Command("go", "mod", "download", dep)
				fmt.Printf("downloading %v\n", dep)
				output, err := cmd.Output()
				if err != nil {
					fmt.Printf("failed to download %v, error %v, output %v\n", dep, err, string(output))
					result[m.Path] = struct{}{}
				} else {
					fmt.Printf("downloaded %v\n", dep)
					subDir := filepath.Join(gopath, "pkg", "mod", dep)
					process(subDir, filters, result)
				}
			}
		}
		return nil
	})
}
