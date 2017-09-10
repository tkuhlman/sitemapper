package mapper

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"sort"
	"testing"
)

func TestJSON(t *testing.T) {
	wantSM := smJSON{
		Nodes: []nodeJSON{
			nodeJSON{ID: "/", Label: "/"},
			nodeJSON{ID: "/hello-world", Label: "/hello-world"},
			nodeJSON{ID: "/values", Label: "/values"},
			nodeJSON{ID: "/variables", Label: "/variables"},
			nodeJSON{ID: "/constants", Label: "/constants", Color: failColor},
		},
		Edges: []edgeJSON{
			edgeJSON{ID: "/->/hello-world", Source: "/", Target: "/hello-world"},
			edgeJSON{ID: "/->/values", Source: "/", Target: "/values"},
			edgeJSON{ID: "/->/variables", Source: "/", Target: "/variables"},
			edgeJSON{ID: "/hello-world->/", Source: "/hello-world", Target: "/"},
			edgeJSON{ID: "/hello-world->/values", Source: "/hello-world", Target: "/values"},
			edgeJSON{ID: "/values->/", Source: "/values", Target: "/"},
			edgeJSON{ID: "/values->/variables", Source: "/values", Target: "/variables"},
			edgeJSON{ID: "/variables->/", Source: "/variables", Target: "/"},
			edgeJSON{ID: "/variables->/constants", Source: "/variables", Target: "/constants"},
		},
	}

	server := httptest.NewServer(http.FileServer(http.Dir("testdata")))
	defer server.Close()

	u, err := url.Parse(server.URL + "/hello-world")
	if err != nil {
		t.Fatal(err)
	}

	sm, err := NewSiteMap(u.String(), 2)
	if err != nil {
		t.Fatal(err)
	}

	if err := sm.Start(); err != nil {
		t.Errorf("Start error: %v", err)
	}

	ln, err := net.Listen("tcp", "")
	http.Handle("/json", sm)
	go http.Serve(ln, nil)
	resp, err := http.Get(fmt.Sprintf("http://%s/json", ln.Addr()))
	if err != nil {
		t.Fatalf("Failed to retrieve JSON: %v", err)
	}

	gotSM := &smJSON{}
	d := json.NewDecoder(resp.Body)
	if err := d.Decode(gotSM); err != nil {
		t.Fatal(err)
	}

	if got, want := len(gotSM.Edges), len(wantSM.Edges); got != want {
		t.Errorf("Got %d edges, want %d", got, want)
	}
	sort.Slice(gotSM.Edges, func(i, j int) bool { return gotSM.Edges[i].ID < gotSM.Edges[j].ID })
	sort.Slice(wantSM.Edges, func(i, j int) bool { return wantSM.Edges[i].ID < wantSM.Edges[j].ID })
	if !reflect.DeepEqual(gotSM.Edges, wantSM.Edges) {
		t.Errorf("Got edges\n%v\nwant\n%v\n", gotSM.Edges, wantSM.Edges)
	}

	if got, want := len(gotSM.Nodes), len(wantSM.Nodes); got != want {
		t.Fatalf("Got %d nodes, want %d", got, want)
	}
	sort.Slice(gotSM.Nodes, func(i, j int) bool { return gotSM.Nodes[i].ID < gotSM.Nodes[j].ID })
	sort.Slice(wantSM.Nodes, func(i, j int) bool { return wantSM.Nodes[i].ID < wantSM.Nodes[j].ID })
	for i, got := range gotSM.Nodes {
		want := wantSM.Nodes[i]
		// note this does not validate x/y positions as they are randomized
		if got.ID != want.ID || got.Label != want.Label || got.Color != want.Color {
			t.Errorf("Node %d - got %#v, want %#v", i, got, want)
		}
	}
}
