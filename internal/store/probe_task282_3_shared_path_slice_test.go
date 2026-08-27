package store_test

import (
	"path/filepath"
	"testing"

	"task282-ednadiag/internal/model"
	"task282-ednadiag/internal/store"
)

func TestListPathsByChainReturnsIndependentCopy(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "p.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	p := &model.ContamPath{ID: "c:p:1", ChainID: "c", Taxon: "carp", ViaBlankID: "b", ViaBatchID: "B1", Status: model.PathCandidate, Score: 0.5, Evidence: "e"}
	if err := st.CreatePath(p); err != nil {
		t.Fatal(err)
	}
	a, err := st.ListPathsByChain("c")
	if err != nil || len(a) != 1 {
		t.Fatalf("a=%v err=%v", a, err)
	}
	b, err := st.ListPathsByChain("c")
	if err != nil || len(b) != 1 {
		t.Fatal(err)
	}
	a[0].Taxon = "mutated"
	if b[0].Taxon == "mutated" {
		t.Fatal("shared backing array between ListPathsByChain calls")
	}
}
