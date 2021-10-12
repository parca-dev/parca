package symbol

import (
	"strings"

	"github.com/google/pprof/profile"
	"github.com/ianlancetaylor/demangle"
)

// Demangle updates the function names in a profile with demangled C++
// names, simplified according to demanglerMode. If force is set,
// overwrite any names that appear already demangled.
func Demangle(prof *profile.Profile, force bool, demanglerMode string) {
	//var functionName string
	//var err error
	//if doDemangle {
	//	// TODO(kakkoyun): Why?
	//	fName := symbolsMap[fAddress].Name
	//	if len(fName) > 7 && fName[:7] == "__dl__Z" {
	//		fName = fName[5:]
	//	}
	//	functionName = demangle.Filter(fName)
	//	//functionName, err = demangle.ToString(fName)
	//	//if err == demangle.ErrNotMangledName {
	//	//	functionName = symbolsMap[fAddress].Name
	//	//}
	//} else {
	//	functionName = symbolsMap[fAddress].Name
	//}

	if force {
		// Remove the current demangled names to force demangling
		for _, f := range prof.Function {
			if f.Name != "" && f.SystemName != "" {
				f.Name = f.SystemName
			}
		}
	}

	var options []demangle.Option
	switch demanglerMode {
	case "": // demangled, simplified: no parameters, no templates, no return type
		options = []demangle.Option{demangle.NoParams, demangle.NoTemplateParams}
	case "templates": // demangled, simplified: no parameters, no return type
		options = []demangle.Option{demangle.NoParams}
	case "full":
		options = []demangle.Option{demangle.NoClones}
	case "none": // no demangling
		return
	}

	// Copy the options because they may be updated by the call.
	o := make([]demangle.Option, len(options))
	for _, fn := range prof.Function {
		if fn.Name != "" && fn.SystemName != fn.Name {
			continue // Already demangled.
		}
		copy(o, options)
		if demangled := demangle.Filter(fn.SystemName, o...); demangled != fn.SystemName {
			fn.Name = demangled
			continue
		}
		// Could not demangle. Apply heuristics in case the name is
		// already demangled.
		name := fn.SystemName
		if looksLikeDemangledCPlusPlus(name) {
			if demanglerMode == "" || demanglerMode == "templates" {
				name = removeMatching(name, '(', ')')
			}
			if demanglerMode == "" {
				name = removeMatching(name, '<', '>')
			}
		}
		fn.Name = name
	}
}

// looksLikeDemangledCPlusPlus is a heuristic to decide if a name is
// the result of demangling C++. If so, further heuristics will be
// applied to simplify the name.
func looksLikeDemangledCPlusPlus(demangled string) bool {
	if strings.Contains(demangled, ".<") { // Skip java names of the form "class.<init>"
		return false
	}
	return strings.ContainsAny(demangled, "<>[]") || strings.Contains(demangled, "::")
}

// removeMatching removes nested instances of start..end from name.
func removeMatching(name string, start, end byte) string {
	s := string(start) + string(end)
	var nesting, first, current int
	for index := strings.IndexAny(name[current:], s); index != -1; index = strings.IndexAny(name[current:], s) {
		switch current += index; name[current] {
		case start:
			nesting++
			if nesting == 1 {
				first = current
			}
		case end:
			nesting--
			switch {
			case nesting < 0:
				return name // Mismatch, abort
			case nesting == 0:
				name = name[:first] + name[current+1:]
				current = first - 1
			}
		}
		current++
	}
	return name
}
