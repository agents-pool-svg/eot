package eot

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// LLMConfig holds the credentials and default model for the LLM gateway.
type LLMConfig struct {
	APIBase      string
	APIKey       string
	DefaultModel string
}

// Preferred (new) and legacy env var names.
var (
	envAPIBase = []string{"EOT_API_BASE", "CODEMATRIX_LLM_API_BASE", "OPENAI_BASE_URL"}
	envAPIKey  = []string{"EOT_API_KEY", "CODEMATRIX_LLM_API_KEY", "OPENAI_API_KEY"}
	envModel   = []string{"EOT_MODEL", "CODEMATRIX_LLM_MODEL", "OPENAI_MODEL"}
)

var kvLineRE = regexp.MustCompile(`^\s*([A-Z_][A-Z0-9_]*)\s*=\s*(.+?)\s*$`)

// ConfigOption configures LoadConfig.
type ConfigOption func(*configOpts)

type configOpts struct {
	apiBase     string
	apiKey      string
	model       string
	projectRoot string
}

// WithAPIBase sets the API base URL explicitly (highest priority).
func WithAPIBase(v string) ConfigOption { return func(o *configOpts) { o.apiBase = v } }

// WithAPIKey sets the API key explicitly (highest priority).
func WithAPIKey(v string) ConfigOption { return func(o *configOpts) { o.apiKey = v } }

// WithModel sets the default model explicitly (highest priority).
func WithModel(v string) ConfigOption { return func(o *configOpts) { o.model = v } }

// WithProjectRoot overrides the directory used to look up .env / ReadMe.md.
func WithProjectRoot(v string) ConfigOption { return func(o *configOpts) { o.projectRoot = v } }

// LoadConfig builds an LLMConfig by resolving, in order:
//
//  1. Explicit options (WithAPIBase / WithAPIKey / WithModel)
//  2. Environment variables (EOT_API_BASE / EOT_API_KEY / EOT_MODEL and legacy aliases)
//  3. KEY=VALUE lines in `.env`, then `ReadMe.md` / `README.md` at project root
//
// It returns an error if api_base or api_key cannot be resolved.
func LoadConfig(opts ...ConfigOption) (*LLMConfig, error) {
	o := &configOpts{}
	for _, f := range opts {
		f(o)
	}

	base, key, model := o.apiBase, o.apiKey, o.model

	// env
	if base == "" {
		base = firstEnv(envAPIBase)
	}
	if key == "" {
		key = firstEnv(envAPIKey)
	}
	if model == "" {
		model = firstEnv(envModel)
	}

	// files
	if base == "" || key == "" || model == "" {
		root := o.projectRoot
		if root == "" {
			cwd, err := os.Getwd()
			if err == nil {
				root = cwd
			}
		}
		if root != "" {
			kv := map[string]string{}
			for _, name := range []string{".env", "ReadMe.md", "README.md"} {
				mergeKV(kv, filepath.Join(root, name))
			}
			if base == "" {
				base = firstKV(kv, envAPIBase)
			}
			if key == "" {
				key = firstKV(kv, envAPIKey)
			}
			if model == "" {
				model = firstKV(kv, envModel)
			}
		}
	}

	if base == "" || key == "" {
		return nil, fmt.Errorf("missing LLM credentials: provide APIBase/APIKey explicitly, " +
			"or set EOT_API_BASE / EOT_API_KEY (env, .env, or ReadMe.md)")
	}
	if model == "" {
		model = "gpt-4o-mini"
	}

	return &LLMConfig{
		APIBase:      normalizeBase(base),
		APIKey:       key,
		DefaultModel: model,
	}, nil
}

func firstEnv(names []string) string {
	for _, n := range names {
		if v := os.Getenv(n); v != "" && !isPlaceholder(v) {
			return v
		}
	}
	return ""
}

func firstKV(kv map[string]string, names []string) string {
	for _, n := range names {
		if v, ok := kv[n]; ok && v != "" && !isPlaceholder(v) {
			return v
		}
	}
	return ""
}

// isPlaceholder returns true for obvious doc placeholders like "<your-api-key>",
// "sk-xxxxxxxx", "your-token-here" etc. This prevents LoadConfig from silently
// accepting unfilled example values copied from README/.env templates.
func isPlaceholder(v string) bool {
	v = strings.TrimSpace(v)
	if v == "" {
		return true
	}
	// <...> angle-bracket placeholders: <your-xxx>, <api-key>, etc.
	if strings.HasPrefix(v, "<") && strings.HasSuffix(v, ">") {
		return true
	}
	lower := strings.ToLower(v)
	markers := []string{
		"your-", "your_", "xxxxxx", "xxxxxxxx",
		"changeme", "replace-me", "todo",
		"example.com", "placeholder",
	}
	for _, m := range markers {
		if strings.Contains(lower, m) {
			return true
		}
	}
	return false
}

func mergeKV(dst map[string]string, path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "#") || strings.HasPrefix(t, "//") || strings.HasPrefix(t, "```") {
			continue
		}
		m := kvLineRE.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		dst[m[1]] = m[2]
	}
}

func normalizeBase(b string) string {
	b = strings.TrimRight(b, "/")
	if !strings.HasSuffix(b, "/v1") {
		b += "/v1"
	}
	return b
}
