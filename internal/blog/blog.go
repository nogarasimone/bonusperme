package blog

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/yuin/goldmark"
	"gopkg.in/yaml.v3"
)

// Post represents a blog article.
type Post struct {
	Slug        string   `yaml:"slug"`
	Title       string   `yaml:"title"`
	Description string   `yaml:"description"`
	Date        time.Time `yaml:"date"`
	Category    string   `yaml:"category"`
	Tags        []string `yaml:"tags"`
	Author      string   `yaml:"author"`
	HTMLContent string   `yaml:"-"`
}

var (
	posts []Post
	mu    sync.RWMutex
)

// LoadAll reads all .md files from dir, parses YAML frontmatter + markdown body,
// and stores them sorted by date descending.
func LoadAll(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	md := goldmark.New()
	var loaded []Post

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}

		p, err := parsePost(data, md)
		if err != nil {
			continue
		}
		loaded = append(loaded, p)
	}

	sort.Slice(loaded, func(i, j int) bool {
		return loaded[i].Date.After(loaded[j].Date)
	})

	mu.Lock()
	posts = loaded
	mu.Unlock()

	return nil
}

func parsePost(data []byte, md goldmark.Markdown) (Post, error) {
	content := string(data)

	// Strip leading BOM if present
	content = strings.TrimPrefix(content, "\xef\xbb\xbf")

	// Split on "---" delimiters
	parts := strings.SplitN(content, "---", 3)
	if len(parts) < 3 {
		return Post{}, fmt.Errorf("invalid frontmatter")
	}

	var p Post
	if err := yaml.Unmarshal([]byte(parts[1]), &p); err != nil {
		return Post{}, err
	}

	// Render markdown to HTML
	var buf bytes.Buffer
	if err := md.Convert([]byte(strings.TrimSpace(parts[2])), &buf); err != nil {
		return Post{}, err
	}
	p.HTMLContent = buf.String()

	return p, nil
}

// GetAll returns all posts sorted by date descending.
func GetAll() []Post {
	mu.RLock()
	defer mu.RUnlock()
	result := make([]Post, len(posts))
	copy(result, posts)
	return result
}

// GetBySlug returns a post by its slug, or nil if not found.
func GetBySlug(slug string) *Post {
	mu.RLock()
	defer mu.RUnlock()
	for i := range posts {
		if posts[i].Slug == slug {
			p := posts[i]
			return &p
		}
	}
	return nil
}

// GetByCategory returns all posts matching a category.
func GetByCategory(cat string) []Post {
	mu.RLock()
	defer mu.RUnlock()
	var result []Post
	for _, p := range posts {
		if p.Category == cat {
			result = append(result, p)
		}
	}
	return result
}
