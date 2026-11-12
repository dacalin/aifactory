package main

import (
	"fmt"
	"io"
	"sort"
)

type checkFunc func(*repository) error

var checks = map[string]checkFunc{}

func register(name string, check checkFunc) {
	if _, exists := checks[name]; exists {
		panic("duplicate check " + name)
	}
	checks[name] = check
}

func checkNames() []string {
	names := make([]string, 0, len(checks))
	for name := range checks {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func runChecks(repo *repository, names []string, output io.Writer) error {
	failures := 0
	for _, name := range names {
		if err := checks[name](repo); err != nil {
			fmt.Fprintf(output, "FAIL %s\n%s\nSee %s/%s.md\n", name, err, repo.rulesDir, name)
			failures++
		} else {
			fmt.Fprintf(output, "PASS %s\n", name)
		}
	}
	if failures > 0 {
		return fmt.Errorf("%d of %d checks failed", failures, len(names))
	}
	fmt.Fprintf(output, "All %d checks passed.\n", len(names))
	return nil
}
