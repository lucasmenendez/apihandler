package apihandler

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

// argName type is a string that represents the name of a named argument
// from a request URI.
type argName string

const (
	// uriSeparator contains a string with the backslash character to split
	// the URI for sanity checks
	uriSeparator = "/"
	// argsToRgxSub constant contains the regex pattern to match a named
	// argument in a request URI, includes the interpolation of the name of
	// the argument
	argsToRgxSub = "(?P<$arg_name>.+)"
)

// argsToRgx variable is a regex that allows to detect named arguments from
// a route path, helping to build a regex to match requests URIs with the
// route supporting named args.
var argsToRgx = regexp.MustCompile(`(?U)\{(?P<arg_name>.+)\}`)

// route struct contains the parameters of a valid route, which contains the
// method, the path, a regex to match request URIs with paths that use named
// arguments, and the route handler.
type route struct {
	method  string
	path    string
	rgx     *regexp.Regexp
	handler func(http.ResponseWriter, *http.Request)
}

// parse function transforms the provided path into a regex to match with
// the URI of incoming requests. The resulting regex will be stored into
// current route and will be used to match named arguments from a request
// URI.
func (r *route) parse() error {
	rgx := argsToRgx.ReplaceAllString(r.path, argsToRgxSub)
	escapedRgx := strings.ReplaceAll(rgx, "/", "\\/")
	var err error
	if r.rgx, err = regexp.Compile(fmt.Sprintf("%s$", escapedRgx)); err != nil {
		return fmt.Errorf("error parsing path: %w", err)
	}
	return nil
}

// match function returns if the requestURI provided matches with the
// current route regex. It also checks if both arguments have the same
// number of URI parts to ensure that is the same level of depth.
func (r *route) match(requestURI string) bool {
	uri, _ := strings.CutSuffix(requestURI, uriSeparator)
	lenURI := strings.Count(uri, uriSeparator)
	lenRgx := strings.Count(r.rgx.String(), uriSeparator)
	return lenURI == lenRgx && r.rgx.MatchString(requestURI)
}

// decodeArgs function returns if the request URI matches with the route
// regex provided and the named arguments that the URI could contain.
func (r *route) decodeArgs(requestURI string) (map[string]string, bool) {
	// check if matches
	if !r.match(requestURI) {
		return nil, false
	}
	// find named arguments
	args := make(map[string]string)
	uri, _ := strings.CutSuffix(requestURI, uriSeparator)
	matches := r.rgx.FindStringSubmatch(uri)
	if len(matches) < 1 {
		return nil, false
	}
	for i, name := range r.rgx.SubexpNames()[0:] {
		args[name] = matches[i]
	}
	return args, true
}
