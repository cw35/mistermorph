package skills

import (
	"fmt"
	"sort"
	"strings"

	"github.com/quailyquaily/mistermorph/internal/markdown"
)

type Frontmatter struct {
	Name         string   `yaml:"name"`
	Description  string   `yaml:"description"`
	AuthProfiles []string `yaml:"auth_profiles"`
	Requirements []string `yaml:"-"`
}

func ParseFrontmatter(contents string) (Frontmatter, bool) {
	type rawFrontmatter struct {
		Name         string   `yaml:"name"`
		Description  string   `yaml:"description"`
		AuthProfiles []string `yaml:"auth_profiles"`
		Requirements any      `yaml:"requirements"`
	}
	parsed, _, ok := markdown.ParseFrontmatter[rawFrontmatter](contents)
	if !ok {
		return Frontmatter{}, false
	}
	raw := parsed

	fm := Frontmatter{
		Name:         strings.TrimSpace(raw.Name),
		Description:  strings.TrimSpace(raw.Description),
		AuthProfiles: normalizeAuthProfiles(raw.AuthProfiles),
		Requirements: normalizeRequirements(raw.Requirements),
	}
	return fm, true
}

func normalizeAuthProfiles(in []string) []string {
	uniq := make(map[string]bool, len(in))
	out := make([]string, 0, len(in))
	for _, p := range in {
		p = strings.TrimSpace(p)
		if p == "" || uniq[p] {
			continue
		}
		uniq[p] = true
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

func normalizeRequirements(raw any) []string {
	if raw == nil {
		return nil
	}
	var items []string
	switch v := raw.(type) {
	case string:
		items = append(items, v)
	case []string:
		items = append(items, v...)
	case []any:
		for _, item := range v {
			items = append(items, stringifyRequirementItem(item)...)
		}
	default:
		items = append(items, fmt.Sprint(v))
	}
	uniq := make(map[string]bool, len(items))
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" || uniq[item] {
			continue
		}
		uniq[item] = true
		out = append(out, item)
	}
	return out
}

func stringifyRequirementItem(v any) []string {
	switch item := v.(type) {
	case nil:
		return nil
	case string:
		return []string{item}
	case map[string]any:
		keys := make([]string, 0, len(item))
		for k := range item {
			keys = append(keys, strings.TrimSpace(k))
		}
		sort.Strings(keys)
		out := make([]string, 0, len(keys))
		for _, k := range keys {
			out = append(out, fmt.Sprintf("%s: %s", k, strings.TrimSpace(fmt.Sprint(item[k]))))
		}
		return out
	case map[any]any:
		tmp := make(map[string]any, len(item))
		for k, val := range item {
			key := strings.TrimSpace(fmt.Sprint(k))
			if key == "" {
				continue
			}
			tmp[key] = val
		}
		return stringifyRequirementItem(tmp)
	default:
		return []string{fmt.Sprint(item)}
	}
}
