package provider

import (
	"context"
	"testing"
	"time"

	dc "github.com/fsouza/go-dockerclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDocker_List(t *testing.T) {
	dclient := &DockerClientMock{
		ListContainersFunc: func(opts dc.ListContainersOptions) ([]dc.APIContainers, error) {
			return []dc.APIContainers{
				{Names: []string{"c1"}, State: "running",
					Networks: dc.NetworkList{
						Networks: map[string]dc.ContainerNetwork{"bridge": {IPAddress: "127.0.0.2"}},
					},
					Ports: []dc.APIPort{
						{PrivatePort: 12345},
					},
					Labels: map[string]string{"reproxy.route": "^/api/123/(.*)", "reproxy.dest": "/blah/$1",
						"reproxy.server": "example.com", "reproxy.ping": "/ping"},
				},
				{Names: []string{"c2"}, State: "running",
					Networks: dc.NetworkList{
						Networks: map[string]dc.ContainerNetwork{"bridge": {IPAddress: "127.0.0.3"}},
					},
					Ports: []dc.APIPort{
						{PrivatePort: 12346},
					},
				},
				{Names: []string{"c3"}, State: "stopped"},
				{Names: []string{"c4"}, State: "running",
					Networks: dc.NetworkList{
						Networks: map[string]dc.ContainerNetwork{"other": {IPAddress: "127.0.0.2"}},
					},
					Ports: []dc.APIPort{
						{PrivatePort: 12345},
					},
				},
			}, nil
		},
	}

	d := Docker{DockerClient: dclient, Network: "bridge"}
	res, err := d.List()
	require.NoError(t, err)
	require.Equal(t, 2, len(res))

	assert.Equal(t, "^/api/123/(.*)", res[0].SrcMatch.String())
	assert.Equal(t, "http://127.0.0.2:12345/blah/$1", res[0].Dst)
	assert.Equal(t, "example.com", res[0].Server)
	assert.Equal(t, "http://127.0.0.2:12345/ping", res[0].PingURL)

	assert.Equal(t, "^/api/c2/(.*)", res[1].SrcMatch.String())
	assert.Equal(t, "http://127.0.0.3:12346/$1", res[1].Dst)
	assert.Equal(t, "http://127.0.0.3:12346/ping", res[1].PingURL)
	assert.Equal(t, "*", res[1].Server)

}

func TestDocker_Events(t *testing.T) {
	dclient := &DockerClientMock{
		AddEventListenerWithOptionsFunc: func(options dc.EventsOptions, listener chan<- *dc.APIEvents) error {
			go func() {
				time.Sleep(30 * time.Millisecond)
				listener <- &dc.APIEvents{Type: "container", Status: "start",
					Actor: dc.APIActor{Attributes: map[string]string{"name": "/c1"}}}
				time.Sleep(30 * time.Millisecond)
				listener <- &dc.APIEvents{Type: "container", Status: "start",
					Actor: dc.APIActor{Attributes: map[string]string{"name": "/c2"}}}
			}()
			return nil
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	d := Docker{DockerClient: dclient}
	ch := d.Events(ctx)

	events := 0
	for range ch {
		t.Log("event")
		events++
	}
	assert.Equal(t, 2+1, events, "initial event plus 2 more")
}
