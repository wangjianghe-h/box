package main

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var keyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

type config struct {
	values map[string]string
	arrays map[string][]string
	home   string
	file   string
}

func loadSettings(path, home string) (*config, error) {
	c := &config{
		values: map[string]string{},
		arrays: map[string][]string{},
		home:   home,
		file:   path,
	}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			c.resolveVars()
			return c, nil
		}
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(strings.TrimPrefix(scanner.Text(), "\ufeff"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		eq := strings.IndexByte(line, '=')
		if eq <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:eq])
		if !keyPattern.MatchString(key) {
			continue
		}
		value := strings.TrimSpace(line[eq+1:])
		if strings.HasPrefix(value, "(") && strings.HasSuffix(value, ")") {
			c.arrays[key] = splitWords(value[1 : len(value)-1])
			continue
		}
		if strings.HasPrefix(value, "(") {
			// Multi-line arrays are not used by the module settings.
			continue
		}
		c.values[key] = unquote(stripComment(value))
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	c.resolveVars()
	return c, nil
}

func (c *config) get(key, fallback string) string {
	if value, ok := c.values[key]; ok && value != "" {
		return value
	}
	return fallback
}

func (c *config) array(key string) []string {
	values, ok := c.arrays[key]
	if !ok {
		return nil
	}
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func (c *config) boolean(key string, fallback bool) bool {
	value, ok := c.values[key]
	if !ok {
		return fallback
	}
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true", "1", "yes", "on", "enable", "enabled":
		return true
	case "false", "0", "no", "off", "disable", "disabled":
		return false
	default:
		return fallback
	}
}

func (c *config) resolveVars() {
	builtins := map[string]string{
		"box_dir":       c.home,
		"box_run":       filepath.Join(c.home, "run"),
		"box_run_state": filepath.Join(c.home, "run", "state"),
		"bin_dir":       filepath.Join(c.home, "bin"),
	}
	for key, value := range c.values {
		c.values[key] = expandVars(value, c.values, builtins)
	}
	for key, values := range c.arrays {
		for i, value := range values {
			c.arrays[key][i] = expandVars(value, c.values, builtins)
		}
	}
}

func expandVars(value string, values map[string]string, builtins map[string]string) string {
	for pass := 0; pass < 4; pass++ {
		changed := false
		var b strings.Builder
		for i := 0; i < len(value); {
			if value[i] == '$' && i+1 < len(value) && value[i+1] == '{' {
				end := strings.IndexByte(value[i+2:], '}')
				if end >= 0 {
					key := value[i+2 : i+2+end]
					replacement, ok := builtins[key]
					if !ok {
						replacement, ok = values[key]
					}
					if ok {
						b.WriteString(replacement)
						i += 3 + end
						changed = true
						continue
					}
				}
			}
			b.WriteByte(value[i])
			i++
		}
		value = b.String()
		if !changed {
			break
		}
	}
	return value
}

func stripComment(value string) string {
	var quote byte
	escaped := false
	for i := 0; i < len(value); i++ {
		ch := value[i]
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' && quote == '"' {
			escaped = true
			continue
		}
		if quote != 0 {
			if ch == quote {
				quote = 0
			}
			continue
		}
		if ch == '"' || ch == '\'' {
			quote = ch
			continue
		}
		if ch == '#' && (i == 0 || value[i-1] == ' ' || value[i-1] == '\t') {
			return strings.TrimSpace(value[:i])
		}
	}
	return strings.TrimSpace(value)
}

func unquote(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
			value = value[1 : len(value)-1]
			value = strings.ReplaceAll(value, `\"`, `"`)
			value = strings.ReplaceAll(value, `\'`, `'`)
			value = strings.ReplaceAll(value, `\\`, `\`)
		}
	}
	return value
}

func splitWords(input string) []string {
	var words []string
	var b strings.Builder
	var quote byte
	escaped := false
	flush := func() {
		if b.Len() > 0 {
			words = append(words, b.String())
			b.Reset()
		}
	}
	for i := 0; i < len(input); i++ {
		ch := input[i]
		if escaped {
			b.WriteByte(ch)
			escaped = false
			continue
		}
		if ch == '\\' && quote != '\'' {
			escaped = true
			continue
		}
		if quote != 0 {
			if ch == quote {
				quote = 0
			} else {
				b.WriteByte(ch)
			}
			continue
		}
		switch ch {
		case '\'', '"':
			quote = ch
		case ' ', '\t', '\r', '\n':
			flush()
		default:
			b.WriteByte(ch)
		}
	}
	flush()
	return words
}

func normalizeIPV6Mode(c *config) string {
	if mode := strings.ToLower(c.get("ipv6_mode", "")); mode != "" {
		switch mode {
		case "enable", "enabled", "true", "1":
			return "enable"
		case "disable", "disabled", "system_disable", "off":
			return "disable"
		case "bypass", "bypassed", "false", "0":
			return "bypass"
		}
	}
	if c.boolean("ipv6", true) {
		return "enable"
	}
	return "bypass"
}

func normalizeChoice(value, fallback string, allowed ...string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	for _, item := range allowed {
		if value == item {
			return value
		}
	}
	return fallback
}

func defaultConfigName(core string, c *config) string {
	switch core {
	case "mihomo":
		return c.get("name_mihomo_config", "config.yaml")
	case "sing-box":
		return c.get("name_sing_config", "config.json")
	case "xray":
		return c.get("name_xray_config", "config.json")
	case "v2fly":
		return c.get("name_v2fly_config", "config.json")
	case "hysteria":
		return c.get("name_hysteria_config", "config.yaml")
	default:
		return "config.yaml"
	}
}

func readPackages(path string) map[string]int {
	packages := map[string]int{}
	f, err := os.Open(path)
	if err != nil {
		return packages
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}
		uid, err := parseInt(fields[1])
		if err != nil {
			continue
		}
		packages[fields[0]] = uid
	}
	return packages
}

func parseInt(value string) (int, error) {
	return strconv.Atoi(value)
}

func readPackageList(path string) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	var entries []string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		entries = append(entries, line)
	}
	return entries
}

func readAPList(path string) (allow, ignore []string) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}
		switch fields[0] {
		case "allow":
			allow = appendUnique(allow, fields[1])
		case "ignore":
			ignore = appendUnique(ignore, fields[1])
		}
	}
	return allow, ignore
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func readGidList(path string) []int {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	var gids []int
	seen := map[int]bool{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		gid, err := parseInt(line)
		if err != nil || gid < 0 || seen[gid] {
			continue
		}
		seen[gid] = true
		gids = append(gids, gid)
	}
	return gids
}

func resolveEntry(entry string, packages map[string]int) (int, string, bool) {
	prefix := 0
	name := entry
	if colon := strings.IndexByte(entry, ':'); colon >= 0 {
		prefixText := entry[:colon]
		name = entry[colon+1:]
		if prefixText != "" {
			value, err := parseInt(prefixText)
			if err == nil && value >= 0 {
				prefix = value
			}
		}
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return 0, "", false
	}
	if uid, err := parseInt(name); err == nil && uid >= 0 && !strings.Contains(name, ".") {
		return uid, "", true
	}
	base, ok := packages[name]
	if !ok {
		return 0, "", false
	}
	return base + prefix*100000, name, true
}
