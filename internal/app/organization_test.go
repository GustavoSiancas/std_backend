package app

import (
	"std/internal/model"
	"testing"
)

func TestBuildOrganization(t *testing.T) {
	root, child, leaf, other := int64(1), int64(2), int64(3), int64(4)
	deps := []model.Department{
		{Base: model.Base{ID: root}, Name: "Root"},
		{Base: model.Base{ID: child}, Name: "Child", ParentID: &root},
		{Base: model.Base{ID: leaf}, Name: "Leaf", ParentID: &child},
		{Base: model.Base{ID: other}, Name: "Other"},
	}
	chart, err := buildOrganization(deps)
	if err != nil {
		t.Fatal(err)
	}
	if chart.Total != 4 || chart.RootCount != 2 || chart.Roots[0].Children[0].Children[0].ID != leaf {
		t.Fatal("incorrect hierarchy")
	}
	empty, err := buildOrganization(nil)
	if err != nil || empty.Roots == nil || len(empty.Roots) != 0 {
		t.Fatal("invalid empty chart")
	}
	deps[0].ParentID = &leaf
	if _, err := buildOrganization(deps); err == nil {
		t.Fatal("cycle accepted")
	}
	missing := int64(99)
	deps[0].ParentID = &missing
	if _, err := buildOrganization(deps); err == nil {
		t.Fatal("missing parent accepted")
	}
}
