package storage

import (
	"regexp"
	"strings"
)

var (
	// Removes package name and method arguments for Java method names.
	// See tests for examples.
	javaRegExp = regexp.MustCompile(`^(?:[a-z]\w*\.)*([A-Z][\w\$]*\.(?:<init>|[a-z][\w\$]*(?:\$\d+)?))(?:(?:\()|$)`)
	// Removes package name and method arguments for Go function names.
	// See tests for examples.
	goRegExp = regexp.MustCompile(`^(?:[\w\-\.]+\/)+(.+)`)
	// Removes potential module versions in a package path.
	goVerRegExp = regexp.MustCompile(`^(.*?)/v(?:[2-9]|[1-9][0-9]+)([./].*)$`)
	// Strips C++ namespace prefix from a C++ function / method name.
	// NOTE: Make sure to keep the template parameters in the name. Normally,
	// template parameters are stripped from the C++ names but when
	// -symbolize=demangle=templates flag is used, they will not be.
	// See tests for examples.
	cppRegExp                = regexp.MustCompile(`^(?:[_a-zA-Z]\w*::)+(_*[A-Z]\w*::~?[_a-zA-Z]\w*(?:<.*>)?)`)
	cppAnonymousPrefixRegExp = regexp.MustCompile(`^\(anonymous namespace\)::`)
)

// ShortenFunctionName returns a shortened version of a function's name.
func ShortenFunctionName(f string) string {
	f = cppAnonymousPrefixRegExp.ReplaceAllString(f, "")
	f = goVerRegExp.ReplaceAllString(f, `${1}${2}`)
	for _, re := range []*regexp.Regexp{goRegExp, javaRegExp, cppRegExp} {
		if matches := re.FindStringSubmatch(f); len(matches) >= 2 {
			return strings.Join(matches[1:], "")
		}
	}
	return f
}
