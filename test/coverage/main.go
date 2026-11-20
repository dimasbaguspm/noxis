// Command coverage is the gate: it reads `go test -cover` from stdin and refuses a package that
// dropped below the floor committed for it. A floor is a ratchet, not a target — it only ever goes
// up, and it goes up with the tests that earn the raise, in the same pull request.
package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
)

// minimum is the number every package in the tree is held to before it has a floor of its own.
const minimum = 0.0

func main() {
	floors := flag.String("floors", "test/coverage/floors.txt", "file of `package floor` lines")
	flag.Parse()

	if err := run(os.Stdin, os.Stdout, *floors); err != nil {
		fmt.Fprintln(os.Stderr, "coverage:", err)
		os.Exit(1)
	}
}

func run(in io.Reader, out io.Writer, floorsPath string) error {
	limits, err := readFloors(floorsPath)
	if err != nil {
		return err
	}
	measured, err := parse(in)
	if err != nil {
		return err
	}
	if len(measured) == 0 {
		return errors.New("no coverage lines on stdin: run `go test ./... -cover` and pipe it in")
	}

	var (
		failed  []string
		missing []string
		pkgs    = make([]string, 0, len(measured))
	)
	for pkg := range measured {
		pkgs = append(pkgs, pkg)
	}
	sort.Strings(pkgs)

	_, _ = fmt.Fprintf(out, "%-40s %8s %8s\n", "package", "covered", "floor")
	for _, pkg := range pkgs {
		have := measured[pkg]
		floor, declared := limits[pkg]
		if !declared {
			floor = minimum
			missing = append(missing, pkg)
		}
		mark := ""
		if have+0.05 < floor {
			mark = "  BELOW"
			failed = append(failed, fmt.Sprintf("%s: %.1f%% < %.1f%%", pkg, have, floor))
		}
		_, _ = fmt.Fprintf(out, "%-40s %7.1f%% %7.1f%%%s\n", pkg, have, floor, mark)
	}

	if len(failed) > 0 {
		return fmt.Errorf("below the committed floor:\n  %s", strings.Join(failed, "\n  "))
	}
	if len(missing) > 0 {
		// Not a failure: a new package has to exist before it can have a number. Saying so keeps the
		// floors file an accurate description of the tree instead of a wish.
		_, _ = fmt.Fprintf(out, "\nno floor yet (add one to %s): %s\n", floorsPath,
			strings.Join(missing, ", "))
	}
	return nil
}

// readFloors reads `package percentage` lines. A comment or a blank line is skipped.
func readFloors(path string) (map[string]float64, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer func() { _ = file.Close() }()

	limits := map[string]float64{}
	scanner := bufio.NewScanner(file)
	for line := 1; scanner.Scan(); line++ {
		text := strings.TrimSpace(scanner.Text())
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		fields := strings.Fields(text)
		if len(fields) != 2 {
			return nil, fmt.Errorf("%s:%d: want `package floor`, got %q", path, line, text)
		}
		value, err := strconv.ParseFloat(strings.TrimSuffix(fields[1], "%"), 64)
		if err != nil {
			return nil, fmt.Errorf("%s:%d: %w", path, line, err)
		}
		limits[fields[0]] = value
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return limits, nil
}

// parse reads `go test -cover` output. Lines with no coverage, a build failure or a cached result
// that reported nothing are skipped rather than guessed at.
func parse(in io.Reader) (map[string]float64, error) {
	measured := map[string]float64{}
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 || fields[0] != "ok" {
			continue
		}
		pkg := fields[1]
		for i, field := range fields {
			if field != "coverage:" || i+2 >= len(fields) {
				continue
			}
			percent := strings.TrimSuffix(fields[i+1], "%")
			value, err := strconv.ParseFloat(percent, 64)
			if err != nil {
				// `coverage: [no statements]` is a package with no code to cover, not a mistake.
				continue
			}
			measured[pkg] = value
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return measured, nil
}
