package hud

import "github.com/waxb/kubernites/pkg/view"

var (
	shelfCategories = view.Categories
	ShelfBranches   = view.Branches
)

func pageCategory(p view.Page) string { return p.Category }

func pageBranch(p view.Page) string { return p.Branch }
