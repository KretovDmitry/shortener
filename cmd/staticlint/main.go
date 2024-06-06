package main

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/KretovDmitry/shortener/pkg/exitinmain"
	"github.com/kisielk/errcheck/errcheck"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/appends"
	"golang.org/x/tools/go/analysis/passes/asmdecl"
	"golang.org/x/tools/go/analysis/passes/assign"
	"golang.org/x/tools/go/analysis/passes/atomic"
	"golang.org/x/tools/go/analysis/passes/atomicalign"
	"golang.org/x/tools/go/analysis/passes/bools"
	"golang.org/x/tools/go/analysis/passes/buildtag"
	"golang.org/x/tools/go/analysis/passes/cgocall"
	"golang.org/x/tools/go/analysis/passes/composite"
	"golang.org/x/tools/go/analysis/passes/copylock"
	"golang.org/x/tools/go/analysis/passes/deepequalerrors"
	"golang.org/x/tools/go/analysis/passes/defers"
	"golang.org/x/tools/go/analysis/passes/directive"
	"golang.org/x/tools/go/analysis/passes/errorsas"
	"golang.org/x/tools/go/analysis/passes/framepointer"
	"golang.org/x/tools/go/analysis/passes/httpresponse"
	"golang.org/x/tools/go/analysis/passes/ifaceassert"
	"golang.org/x/tools/go/analysis/passes/loopclosure"
	"golang.org/x/tools/go/analysis/passes/lostcancel"
	"golang.org/x/tools/go/analysis/passes/nilfunc"
	"golang.org/x/tools/go/analysis/passes/nilness"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/shadow"
	"golang.org/x/tools/go/analysis/passes/shift"
	"golang.org/x/tools/go/analysis/passes/sigchanyzer"
	"golang.org/x/tools/go/analysis/passes/slog"
	"golang.org/x/tools/go/analysis/passes/sortslice"
	"golang.org/x/tools/go/analysis/passes/stdmethods"
	"golang.org/x/tools/go/analysis/passes/stdversion"
	"golang.org/x/tools/go/analysis/passes/stringintconv"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"golang.org/x/tools/go/analysis/passes/testinggoroutine"
	"golang.org/x/tools/go/analysis/passes/tests"
	"golang.org/x/tools/go/analysis/passes/timeformat"
	"golang.org/x/tools/go/analysis/passes/unmarshal"
	"golang.org/x/tools/go/analysis/passes/unreachable"
	"golang.org/x/tools/go/analysis/passes/unsafeptr"
	"golang.org/x/tools/go/analysis/passes/unusedresult"
	"golang.org/x/tools/go/analysis/passes/unusedwrite"
	"honnef.co/go/tools/quickfix"
	"honnef.co/go/tools/simple"
	"honnef.co/go/tools/staticcheck"
	"honnef.co/go/tools/stylecheck"
)

// Config — name of the configuration file.
const Config = `config.json`

// ConfigData describes configuration file structure.
type ConfigData struct {
	Staticcheck []string
}

func main() {
	// Get the path name for the executable that started the current process.
	appfile, err := os.Executable()
	if err != nil {
		panic(err)
	}

	// Read configuration file.
	data, err := os.ReadFile(filepath.Join(filepath.Dir(appfile), Config))
	if err != nil {
		panic(err)
	}

	// Decode file.
	var cfg ConfigData
	if err = json.Unmarshal(data, &cfg); err != nil {
		panic(err)
	}

	checks := []*analysis.Analyzer{
		/* x/tools/go/anaisys package analizers */

		// detects if there is only one variable in append.
		appends.Analyzer,
		// reports mismatches between assembly files and Go declarations.
		asmdecl.Analyzer,
		// detects useless assignments.
		assign.Analyzer,
		// checks for common mistakes using the sync/atomic package.
		atomic.Analyzer,
		// checks for non-64-bit-aligned arguments to sync/atomic functions.
		atomicalign.Analyzer,
		// detects common mistakes involving boolean operators.
		bools.Analyzer,
		// checks build tags.
		buildtag.Analyzer,
		// detects some violations of the cgo pointer passing rules.
		cgocall.Analyzer,
		// checks for unkeyed composite literals.
		composite.Analyzer,
		// checks for locks erroneously passed by value.
		copylock.Analyzer,
		// checks for the use of reflect.DeepEqual with error values.
		deepequalerrors.Analyzer,
		// checks for common mistakes in defer statements.
		defers.Analyzer,
		// checks known Go toolchain directives.
		directive.Analyzer,
		// checks that the second argument to errors.As
		// is a pointer to a type implementing error.
		errorsas.Analyzer,
		// reports assembly code that clobbers
		// the frame pointer before saving it.
		framepointer.Analyzer,
		// checks for mistakes using HTTP responses.
		httpresponse.Analyzer,
		// reports impossible interface-interface type assertions.
		ifaceassert.Analyzer,
		// checks for references to enclosing loop variables
		// from within nested functions.
		loopclosure.Analyzer,
		// checks for failure to call a context cancellation function.
		lostcancel.Analyzer,
		// checks for useless comparisons against nil.
		nilfunc.Analyzer,
		// inspects the control-flow graph of an SSA
		// (requires buildssa analyzer) function and reports errors
		// such as nil pointer dereferences and
		// degenerate nil pointer comparisons.
		nilness.Analyzer,
		// checks consistency of Printf format strings and arguments.
		printf.Analyzer,
		// checks for shadowed variables.
		shadow.Analyzer,
		// checks for shifts that exceed the width of an integer.
		shift.Analyzer,
		// detects misuse of unbuffered signal as argument to signal.Notify.
		sigchanyzer.Analyzer,
		// checks for mismatched key-value pairs in log/slog calls.
		slog.Analyzer,
		// checks for calls to sort.Slice that do not use a slice type
		// as first argument.
		sortslice.Analyzer,
		// checks for misspellings in the signatures of methods
		// similar to well-known interfaces.
		stdmethods.Analyzer,
		// reports uses of standard library symbols that are "too new"
		// for the Go version in force in the referring file.
		stdversion.Analyzer,
		// reports type conversions from integers to strings.
		stringintconv.Analyzer,
		// checks struct field tags are well formed.
		structtag.Analyzer,
		// reports calls to Fatal from a test goroutine.
		testinggoroutine.Analyzer,
		// checks for common mistaken usages of tests and examples.
		tests.Analyzer,
		// checks for the use of time.Format
		// or time.Parse calls with a bad format.
		timeformat.Analyzer,
		// checks for passing non-pointer or non-interface types
		// to unmarshal and decode functions.
		unmarshal.Analyzer,
		// checks for unreachable code.
		unreachable.Analyzer,
		// checks for invalid conversions of uintptr to unsafe.Pointer.
		unsafeptr.Analyzer,
		// checks for unused results of calls to certain pure functions.
		unusedresult.Analyzer,
		// checks for unused writes to the elements of a struct or array object.
		unusedwrite.Analyzer,

		/* Custom checkers. */

		// reports os.Exit call inside main function of the main package
		exitinmain.Analyzer,

		/* External checkers. */

		// checks that you checked errors.
		errcheck.Analyzer,
	}

	configChecks := make(map[string]bool, len(cfg.Staticcheck))
	for _, v := range cfg.Staticcheck {
		configChecks[v] = true
	}

	// Add staticcheck's analyzers specified in the configuration file.
	for _, v := range staticcheck.Analyzers {
		if configChecks[v.Analyzer.Name] {
			checks = append(checks, v.Analyzer)
		}
	}

	// Add code simplifications analyzers specified in the configuration file.
	for _, v := range simple.Analyzers {
		if configChecks[v.Analyzer.Name] {
			checks = append(checks, v.Analyzer)
		}
	}

	// Add stylistic issues analyzers specified in the configuration file.
	for _, v := range stylecheck.Analyzers {
		if configChecks[v.Analyzer.Name] {
			checks = append(checks, v.Analyzer)
		}
	}

	// Add quickfixes analyzers specified in the configuration file.
	for _, v := range quickfix.Analyzers {
		if configChecks[v.Analyzer.Name] {
			checks = append(checks, v.Analyzer)
		}
	}

	multichecker.Main(checks...)
}
