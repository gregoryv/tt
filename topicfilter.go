package tt

import "strings"

func NewTopicFilter(filter string) *TopicFilter {
	r := &TopicFilter{
		filter: filter,
		levels: strings.Split(filter, "/"),
		always: filter == "#",
	}
	if !r.always {
		r.hasMulti = strings.HasSuffix(filter, "/#")
		r.hasSingle = strings.Contains(filter, "+")
	}
	return r
}

type TopicFilter struct {
	filter    string
	levels    []string // topicFilter split into words a/# becomes "a", "#"
	always    bool
	hasMulti  bool
	hasSingle bool
}

// Match topic name and return any wildcard words.
func (r *TopicFilter) Match(name string) ([]string, bool) {
	// special case always
	if r.always {
		return nil, true
	}
	// without wildcards
	if !r.hasMulti && !r.hasSingle {
		return nil, name == r.filter
	}

	// + matches are saved for easy access by handlers
	var words []string

	// wip don't like this algorithm one bit

	var j int // index in name
	for _, f := range r.levels {
		switch f {
		case "#":
			if r.hasMulti {
				return words, true
			}
			// todo maybe warn on bad filter, eg. a/#/b

		case "+":
			// wip what does j mean here, next position? weird
			w := word(name, j)
			words = append(words, w)
			j += len(w) + 1

		case name[j : j+len(f)]: // word match
			j += len(f) + 1

		default:
			return nil, false
		}
	}

	// if name not consumed by filter
	if j < len(name) {
		return nil, false
	}

	return words, true
}

func word(name string, i int) string {
	width := strings.Index(name[i:], "/")
	if width > 0 {
		return name[i : i+width]
	}
	return name[i:]

}
