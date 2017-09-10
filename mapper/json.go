package mapper

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
)

const failColor = "#ec5148"

type nodeJSON struct {
	Color string `json:"color"`
	ID    string `json:"id"`
	Label string `json:"label"`
	Size  int    `json:"size"`
	X     int    `json:"x"`
	Y     int    `json:"y"`
}

type edgeJSON struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
}

type smJSON struct {
	Nodes []nodeJSON `json:"nodes"`
	Edges []edgeJSON `json:"edges"`
}

// MarshalJSON outputs the JSON representaion of sm needed for use by sigmajs
// to display a site map. It implements the json.Marshaller interface.
func (sm *SiteMap) MarshalJSON() ([]byte, error) {
	j := smJSON{Nodes: []nodeJSON{}, Edges: []edgeJSON{}}

	for id, p := range sm.pages {
		n := nodeJSON{ID: id, Label: id, X: rand.Intn(1000), Y: rand.Intn(1000)}
		if p.broken {
			n.Color = failColor
		}
		j.Nodes = append(j.Nodes, n)
		for path := range p.links {
			j.Edges = append(j.Edges, edgeJSON{ID: fmt.Sprintf("%s->%s", id, path), Source: id, Target: path})
		}
	}
	return json.Marshal(j)
}

// ServeHTTP implments the http.Handler interface responding with sm marshaled
// as JSON.
func (sm *SiteMap) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	enc := json.NewEncoder(w)
	if err := enc.Encode(sm); err != nil {
		http.Error(w, fmt.Sprintf("failed to marshal sitemap as JSON: %v", err), http.StatusInternalServerError)
	}
}
