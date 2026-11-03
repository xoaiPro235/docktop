package docker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/moby/moby/client"
)

func TestClient_ListVolumes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/volumes"):
			_, _ = w.Write([]byte(`{"Volumes":[{"Name":"unused","Driver":"local"},{"Name":"shared","Driver":"local"},{"Name":"other","Driver":"local"}]}`))
		case strings.HasSuffix(r.URL.Path, "/containers/json"):
			_, _ = w.Write([]byte(`[
				{"Id":"ctr-z","Names":["/zulu"],"State":"running","Mounts":[{"Type":"volume","Name":"shared","Destination":"/data","Mode":"rw","RW":true}]},
				{"Id":"ctr-a","Names":["/alpha"],"State":"exited","Mounts":[{"Type":"volume","Name":"shared","Destination":"/cache","Mode":"ro","RW":false},{"Type":"volume","Name":"other","Destination":"/other"}]}
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

	got, err := (&Client{cli: cli}).ListVolumes(context.Background())
	if err != nil {
		t.Fatalf("ListVolumes() error = %v", err)
	}
	want := []VolumeInfo{
		{
			Name: "shared", Driver: "local",
			InUse:  true,
			RefCnt: 2,
			Containers: []VolumeContainerMount{
				{ContainerID: "ctr-a", ContainerName: "alpha", State: "exited", Destination: "/cache", Mode: "ro", RW: false},
				{ContainerID: "ctr-z", ContainerName: "zulu", State: "running", Destination: "/data", Mode: "rw", RW: true},
			},
		},
		{
			Name: "other", Driver: "local",
			InUse:      true,
			RefCnt:     1,
			Containers: []VolumeContainerMount{{ContainerID: "ctr-a", ContainerName: "alpha", State: "exited", Destination: "/other"}},
		},
		{Name: "unused", Driver: "local"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ListVolumes() = %#v, want %#v", got, want)
	}
}
