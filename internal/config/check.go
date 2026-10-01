package config

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Problem is a single issue found in a config file.
type Problem struct {
	Line int // 0 if unknown
	Msg  string
}

func (p Problem) String() string {
	if p.Line > 0 {
		return fmt.Sprintf("line %d: %s", p.Line, p.Msg)
	}
	return p.Msg
}

// Result is the outcome of Check. Config is nil when there are problems.
type Result struct {
	Config   *Config
	Problems []Problem
}

var varName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

var (
	topLevelKeys = []string{"files", "packs", "vars"}
	entryKeys    = []string{"filename", "url", "vars", "raw"}
)

// Check validates config data and reports every problem it finds, including
// unknown keys (with a "did you mean" suggestion when one is close). An empty
// config is valid.
func Check(data []byte) Result {
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return Result{Problems: []Problem{{Msg: "syntax error: " + err.Error()}}}
	}

	c := &checker{}
	doc := &root
	if doc.Kind == yaml.DocumentNode && len(doc.Content) > 0 {
		doc = doc.Content[0]
	}
	switch {
	case doc.Kind == 0 || isNull(doc):
		return Result{Config: &Config{}}
	case doc.Kind != yaml.MappingNode:
		c.add(doc, "top level must be a mapping with %q, %q and/or %q", "files", "packs", "vars")
		return Result{Problems: c.problems}
	}

	var filesNode, packsNode, varsNode *yaml.Node
	for i := 0; i < len(doc.Content); i += 2 {
		k, v := doc.Content[i], doc.Content[i+1]
		switch k.Value {
		case "files":
			filesNode = v
		case "packs":
			packsNode = v
		case "vars":
			varsNode = v
		default:
			c.unknown(k, "key", "", topLevelKeys)
		}
	}

	fileKeys := map[string]bool{}
	if filesNode != nil {
		c.checkFiles(filesNode, fileKeys)
	}
	if varsNode != nil {
		c.checkVars("top level", varsNode)
	}
	if packsNode != nil {
		c.checkPacks(packsNode, fileKeys)
	}
	if len(c.problems) > 0 {
		sort.SliceStable(c.problems, func(i, j int) bool { return c.problems[i].Line < c.problems[j].Line })
		return Result{Problems: c.problems}
	}

	var cfg Config
	if err := doc.Decode(&cfg); err != nil {
		return Result{Problems: []Problem{{Msg: err.Error()}}}
	}
	return Result{Config: &cfg}
}

type checker struct{ problems []Problem }

func (c *checker) add(n *yaml.Node, format string, args ...any) {
	c.problems = append(c.problems, Problem{Line: n.Line, Msg: fmt.Sprintf(format, args...)})
}

// unknown reports an unrecognised key, suggesting the closest allowed one.
func (c *checker) unknown(k *yaml.Node, what, in string, allowed []string) {
	msg := fmt.Sprintf("unknown %s %q%s", what, k.Value, in)
	if s := suggest(k.Value, allowed); s != "" {
		msg += fmt.Sprintf(", did you mean %q?", s)
	}
	c.add(k, "%s", msg)
}

func (c *checker) checkFiles(n *yaml.Node, fileKeys map[string]bool) {
	if isNull(n) {
		return
	}
	if n.Kind != yaml.MappingNode {
		c.add(n, "%q must be a mapping of key to {filename, url}", "files")
		return
	}
	for i := 0; i < len(n.Content); i += 2 {
		k, v := n.Content[i], n.Content[i+1]
		fileKeys[k.Value] = true
		if v.Kind != yaml.MappingNode {
			c.add(k, "file %q must be a mapping with filename and url", k.Value)
			continue
		}
		set := map[string]bool{}
		var raw, vars *yaml.Node
		for j := 0; j < len(v.Content); j += 2 {
			f, val := v.Content[j], v.Content[j+1]
			if !contains(entryKeys, f.Value) {
				c.unknown(f, "field", fmt.Sprintf(" in file %q", k.Value), entryKeys)
				continue
			}
			switch f.Value {
			case "vars":
				vars = f
				c.checkVars(fmt.Sprintf("file %q", k.Value), val)
				continue
			case "raw":
				if val.Kind != yaml.ScalarNode || val.Tag != "!!bool" {
					c.add(f, "file %q: raw must be true or false", k.Value)
				} else if val.Value == "true" {
					raw = f
				}
				continue
			}
			if val.Kind == yaml.ScalarNode && val.Value != "" {
				set[f.Value] = true
			}
		}
		for _, name := range []string{"filename", "url"} {
			if !set[name] {
				c.add(k, "file %q is missing %s", k.Value, name)
			}
		}
		if raw != nil && vars != nil {
			c.add(vars, "file %q: vars has no effect with raw: true", k.Value)
		}
	}
}

// checkVars validates a vars mapping; where says whose it is, for messages.
func (c *checker) checkVars(where string, n *yaml.Node) {
	if isNull(n) {
		return
	}
	if n.Kind != yaml.MappingNode {
		c.add(n, "%s: vars must be a mapping of NAME to value", where)
		return
	}
	for i := 0; i < len(n.Content); i += 2 {
		k, v := n.Content[i], n.Content[i+1]
		if !varName.MatchString(k.Value) {
			c.add(k, "%s: invalid variable name %q (use letters, digits and _)", where, k.Value)
		}
		if v.Kind != yaml.ScalarNode || isNull(v) {
			c.add(k, "%s: variable %q needs a string value", where, k.Value)
		}
	}
}

func (c *checker) checkPacks(n *yaml.Node, fileKeys map[string]bool) {
	if isNull(n) {
		return
	}
	if n.Kind != yaml.MappingNode {
		c.add(n, "%q must be a mapping of pack name to a list of file keys", "packs")
		return
	}
	known := sortedKeys(fileKeys)
	for i := 0; i < len(n.Content); i += 2 {
		k, v := n.Content[i], n.Content[i+1]
		if isNull(v) {
			continue
		}
		if v.Kind != yaml.SequenceNode {
			c.add(k, "pack %q must be a list of file keys", k.Value)
			continue
		}
		for _, item := range v.Content {
			if item.Kind != yaml.ScalarNode {
				c.add(item, "pack %q: entries must be file keys", k.Value)
				continue
			}
			if !fileKeys[item.Value] {
				msg := fmt.Sprintf("pack %q references unknown file %q", k.Value, item.Value)
				if s := suggest(item.Value, known); s != "" {
					msg += fmt.Sprintf(", did you mean %q?", s)
				}
				c.add(item, "%s", msg)
			}
		}
	}
}

func isNull(n *yaml.Node) bool { return n.Kind == yaml.ScalarNode && n.Tag == "!!null" }

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// suggest returns the allowed value closest to s, or "" if none is close.
func suggest(s string, allowed []string) string {
	s = strings.ToLower(s)
	best, bestDist := "", 3
	for _, a := range allowed {
		if d := levenshtein(s, strings.ToLower(a)); d < bestDist {
			best, bestDist = a, d
		}
	}
	return best
}

func levenshtein(a, b string) int {
	prev := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur := make([]int, len(b)+1)
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(b)]
}
