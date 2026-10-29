package app

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"std/internal/model"
)

type departmentSummary struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Code     string `json:"code"`
	IsActive bool   `json:"is_active"`
}
type departmentWithParent struct {
	model.Department
	Parent *departmentSummary `json:"parent"`
}
type OrganizationNode struct {
	model.Department
	Children []*OrganizationNode `json:"children"`
}
type OrganizationChart struct {
	Roots     []*OrganizationNode `json:"roots"`
	Total     int                 `json:"total"`
	RootCount int                 `json:"root_count"`
}

var errHierarchy = errors.New("department hierarchy contains a cycle or missing parent")

func buildOrganization(departments []model.Department) (OrganizationChart, error) {
	chart := OrganizationChart{Roots: make([]*OrganizationNode, 0), Total: len(departments)}
	nodes := make(map[int64]*OrganizationNode, len(departments))
	for _, d := range departments {
		nodes[d.ID] = &OrganizationNode{Department: d, Children: make([]*OrganizationNode, 0)}
	}
	// Iterative parent traversal validates cycles without recursion on deep hierarchies.
	done := make(map[int64]bool, len(nodes))
	for _, d := range departments {
		path := make(map[int64]bool)
		cursor := d.ID
		for !done[cursor] {
			if path[cursor] {
				return OrganizationChart{}, errHierarchy
			}
			path[cursor] = true
			n, ok := nodes[cursor]
			if !ok {
				return OrganizationChart{}, errHierarchy
			}
			if n.ParentID == nil {
				break
			}
			cursor = *n.ParentID
		}
		for id := range path {
			done[id] = true
		}
	}
	for _, d := range departments {
		node := nodes[d.ID]
		if d.ParentID == nil {
			chart.Roots = append(chart.Roots, node)
		} else {
			parent := nodes[*d.ParentID]
			parent.Children = append(parent.Children, node)
		}
	}
	chart.RootCount = len(chart.Roots)
	return chart, nil
}

// GetOrganizationChart reads the complete hierarchy, including inactive areas,
// so children never lose their parents due to filtering or pagination.
func (a *App) GetOrganizationChart(ctx context.Context) (OrganizationChart, error) {
	var departments []model.Department
	if err := a.DB.WithContext(ctx).Order("name ASC,id ASC").Find(&departments).Error; err != nil {
		return OrganizationChart{}, err
	}
	return buildOrganization(departments)
}
func (a *App) organizationChart(c *gin.Context) {
	chart, err := a.GetOrganizationChart(c.Request.Context())
	if errors.Is(err, errHierarchy) {
		fail(c, 409, err)
		return
	}
	if err != nil {
		accountError(c, err)
		return
	}
	c.JSON(200, chart)
}
