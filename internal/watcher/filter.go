package watcher

import (
	"fmt"
	"regexp"
)

type Filter struct {
	patterns []*regexp.Regexp
}

func NewFilter(patterns []string) (*Filter, error) {
	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, p := range patterns {
		re, err := regexp.Compile(p)
		if err != nil {
			return nil, fmt.Errorf("invalid exclude pattern %q: %w", p, err)
		}
		compiled = append(compiled, re)
	}
	return &Filter{patterns: compiled}, nil
}

func (f *Filter) ShouldExclude(relPath string) bool {
	for _, re := range f.patterns {
		if re.MatchString(relPath) {
			return true
		}
	}
	return false
}
