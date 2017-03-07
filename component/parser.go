package component

import (
	"io"
	"sort"
	"strings"

	git "gopkg.in/src-d/go-git.v3"
)

// Parser is an helper, creates a tree from the repo
type Parser struct {
	index      map[string]int
	categories []*Category
	assets     []*Asset
}

// Parse executes the parsing on a repo
func (p *Parser) Parse(t *git.Tree) error {
	p.index = make(map[string]int)
	p.categories = make([]*Category, 0)
	if err := p.parse(t, filterCat); err != nil {
		return err
	}
	if err := p.parse(t, filterRes); err != nil {
		return err
	}
	sort.Sort(catSorter(p.categories))
	for i := range p.categories {
		sort.Sort(subSorter(p.categories[i].subcategories))
		for j := range p.categories[i].subcategories {
			sort.Sort(itemSorter(p.categories[i].subcategories[j].items))
		}
	}
	return nil
}

func (p *Parser) parse(t *git.Tree, fn func(name string) bool) error {
	for iter := t.Files(); ; {
		f, err := iter.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if !fn(f.Name) {
			continue
		}
		if err := p.parseFile(f); err != nil {
			return err
		}
	}
	return nil
}

func (p *Parser) parseFile(f *git.File) error {
	contents, err := f.Contents()
	if err != nil {
		return parseError{f.Name, "read", err}
	}
	cmp, err := newCmp("/" + f.Name)
	if err != nil {
		return parseError{f.Name, "cmp", err}
	}
	parts := strings.Split(f.Name, "/")
	if err := cmp.SetPath("/" + f.Name); err != nil {
		return parseError{f.Name, "path", err}
	}
	switch c := cmp.(type) {
	case *Category:
		p.index[parts[1]] = len(p.categories)
		p.categories = append(p.categories, c)
	case *Subcategory:
		p.categories[p.index[parts[1]]].Add(c)
	case *Item:
		p.categories[p.index[parts[1]]].Sub(parts[2]).AddItem(c)
	case *Checklist:
		p.categories[p.index[parts[1]]].Sub(parts[2]).SetChecks(c)
	case *Asset:
		p.assets = append(p.assets, c)
	default:
		return parseError{f.Name, "type", "Invalid Path"}
	}
	if err := cmp.SetContents(strings.TrimSpace(contents)); err != nil {
		return parseError{f.Name, "contents", err}
	}
	return nil
}

func (p *Parser) Categories() map[string][]*Category {
	var res = make(map[string][]*Category)
	for _, cat := range p.categories {
		res[cat.Locale] = append(res[cat.Locale], cat)
	}
	return res
}

func (p *Parser) Assets() []*Asset {
	return p.assets
}
