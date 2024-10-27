package http

import (
	"bufio"
	"io"
	"strings"
)

type RouteHandler struct {
	pattern  string
	segments []PathSegment
	handle   HandlerFunction
	method   string
}

type HandlerFunction func(req HttpRequest, res HttpResponse)

func NewRouteHandler(method string, routePattern string, h HandlerFunction) (RouteHandler, error) {
	segments, err := getPathSegments(routePattern)
	if err != nil {
		return RouteHandler{}, err
	}

	return RouteHandler{
		pattern:  routePattern,
		segments: segments,
		handle:   h,
		method:   strings.ToLower(method),
	}, nil
}

type PathSegment struct {
	Part  string
	IsVar bool
}

func match(route string, segments []PathSegment) (bool, map[string]string) {
	reader := bufio.NewReader(strings.NewReader(strings.ToLower(route)))
	routeParams := make(map[string]string)
	n := 0
	for {
		c, err := reader.Peek(1)
		// fmt.Printf("n=%d, c=%v, err=%v\n", n, c, err)

		if n == len(segments) {
			if err == io.EOF {
				// fmt.Printf("hit complete match\n")
				return true, routeParams
			} else {
				// fmt.Println("end of segments but did not complete the route")
				break
			}
		}

		if err != nil {
			// fmt.Printf("hit error %v\n", err)
			break
		}

		if c[0] == '/' {
			// fmt.Printf("discarding /\n")
			reader.Discard(1)
		}

		expected := segments[n]

		got, err := reader.ReadString('/')
		if err != nil && err != io.EOF {
			// fmt.Printf("hit unexpected error reading rest of path %v\n", err)
			break
		}

		got = strings.TrimSuffix(got, "/")

		if expected.IsVar {
			// fmt.Printf("part is a variable\n")
			routeParams[expected.Part] = got
		} else {
			if got != expected.Part {
				// fmt.Printf("part didn't match (%s!=%s)\n", expected.Part, got)
				break // no match on route
			}
			// fmt.Printf("part matched (%s==%s)\n", expected.Part, got)
		}

		n++
	}

	// fmt.Println("found no match")

	return false, make(map[string]string)
}

func getPathSegments(route string) ([]PathSegment, error) {
	reader := bufio.NewReader(strings.NewReader(route))

	parts := make([]PathSegment, 0)
	for {
		c, err := reader.Peek(1)
		if err != nil {
			if err == io.EOF {
				break
			}
		}

		if c[0] == '/' {
			reader.Discard(1)
		}

		part, err := reader.ReadString('/')
		if err != nil && err != io.EOF {
			return make([]PathSegment, 0), err
		}

		part = strings.TrimSuffix(part, "/")

		isvar := false
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			isvar = true
			part = strings.TrimSuffix(strings.TrimPrefix(part, "{"), "}")
		} else {
			part = strings.ToLower(part) // only lowercase non-vars
		}

		parts = append(parts, PathSegment{
			Part:  part,
			IsVar: isvar,
		})
	}

	return parts, nil
}
