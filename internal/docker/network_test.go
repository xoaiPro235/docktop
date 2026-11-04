package docker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/moby/moby/client"
)

func TestClient_ListNetworks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/networks"):
			_, _ = w.Write([]byte(`[
				{"Id":"system-id","Name":"bridge","Driver":"bridge","Scope":"local"},
				{"Id":"empty-id","Name":"unused","Driver":"bridge","Scope":"local"},
				{"Id":"used-id","Name":"app","Driver":"bridge","Scope":"local"}
			]`))
		case strings.HasSuffix(r.URL.Path, "/containers/json"):
			_, _ = w.Write([]byte(`[
				{
					"Id":"container-id",
					"Names":["/web"],
					"State":"running",
					"Ports":[],
					"NetworkSettings":{"Networks":{
						"app":{
							"NetworkID":"used-id",
							"IPAddress":"172.18.0.2",
							"IPPrefixLen":16,
							"MacAddress":"02:42:ac:12:00:02"
						}
					}}
				}
			]`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	cli, err := client.New(client.WithHost(server.URL), client.WithAPIVersion("1.52"))
	if err != nil {
		t.Fatal(err)
	}
	defer cli.Close() // nolint:errcheck

	got, err := (&Client{cli: cli}).ListNetworks(context.Background())
	if err != nil {
		t.Fatalf("ListNetworks() error = %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("ListNetworks() returned %d networks, want 3", len(got))
	}
	if got[0].Name != "app" || got[1].Name != "unused" || got[2].Name != "bridge" {
		t.Errorf("network order = [%q, %q, %q], want [app, unused, bridge]", got[0].Name, got[1].Name, got[2].Name)
	}
	if got[0].ConnectedCount != 1 || got[0].IsSystem {
		t.Errorf("connected user network metadata = %+v, want one endpoint and non-system", got[0])
	}
	wantEndpoint := NetworkEndpointInfo{
		ContainerID: "container-id",
		Name:        "web",
		IP:          "172.18.0.2/16",
		MacAddress:  "02:42:ac:12:00:02",
		Ports:       "-",
	}
	if len(got[0].Containers) != 1 || got[0].Containers[0] != wantEndpoint {
		t.Errorf("connected network endpoints = %+v, want [%+v]", got[0].Containers, wantEndpoint)
	}
	if got[1].ConnectedCount != 0 || got[1].IsSystem {
		t.Errorf("unused user network metadata = %+v, want no endpoints and non-system", got[1])
	}
	if got[2].ConnectedCount != 0 || !got[2].IsSystem {
		t.Errorf("system network metadata = %+v, want no endpoints and system", got[2])
	}
}

func Test_networkScore(t *testing.T) {
	tests := []struct {
		name string
		n    NetworkInfo
		want int
	}{
		{
			name: "system network gets score 3",
			n:    NetworkInfo{IsSystem: true, ConnectedCount: 5},
			want: 3,
		},
		{
			name: "user network with connected containers gets score 1",
			n:    NetworkInfo{IsSystem: false, ConnectedCount: 2},
			want: 1,
		},
		{
			name: "empty user network gets score 2",
			n:    NetworkInfo{IsSystem: false, ConnectedCount: 0},
			want: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := networkScore(tt.n); got != tt.want {
				t.Errorf("networkScore() = %v, want %v", got, tt.want)
			}
		})
	}
}
