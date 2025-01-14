package discovery

import (
	"context"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_Do(t *testing.T) {
	p1 := &ProviderMock{
		EventsFunc: func(ctx context.Context) <-chan struct{} {
			res := make(chan struct{}, 1)
			res <- struct{}{}
			return res
		},
		ListFunc: func() ([]URLMapper, error) {
			return []URLMapper{
				{Server: "*", SrcMatch: *regexp.MustCompile("^/api/svc1/(.*)"), Dst: "http://127.0.0.1:8080/blah1/$1"},
				{Server: "*", SrcMatch: *regexp.MustCompile("^/api/svc2/(.*)"), Dst: "http://127.0.0.2:8080/blah2/$1/abc"},
			}, nil
		},
		IDFunc: func() ProviderID {
			return PIFile
		},
	}
	p2 := &ProviderMock{
		EventsFunc: func(ctx context.Context) <-chan struct{} {
			return make(chan struct{}, 1)
		},
		ListFunc: func() ([]URLMapper, error) {
			return []URLMapper{
				{Server: "localhost", SrcMatch: *regexp.MustCompile("/api/svc3/xyz"), Dst: "http://127.0.0.3:8080/blah3/xyz"},
			}, nil
		},
		IDFunc: func() ProviderID {
			return PIDocker
		},
	}
	svc := NewService([]Provider{p1, p2})

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	err := svc.Run(ctx)
	require.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
	mappers := svc.Mappers()
	assert.Equal(t, 3, len(mappers))
	assert.Equal(t, PIFile, mappers[0].ProviderID)
	assert.Equal(t, "*", mappers[0].Server)
	assert.Equal(t, "^/api/svc1/(.*)", mappers[0].SrcMatch.String())
	assert.Equal(t, "http://127.0.0.1:8080/blah1/$1", mappers[0].Dst)

	assert.Equal(t, 1, len(p1.EventsCalls()))
	assert.Equal(t, 1, len(p2.EventsCalls()))

	assert.Equal(t, 1, len(p1.ListCalls()))
	assert.Equal(t, 1, len(p2.ListCalls()))

	assert.Equal(t, 2, len(p1.IDCalls()))
	assert.Equal(t, 1, len(p2.IDCalls()))
}

func TestService_Match(t *testing.T) {
	p1 := &ProviderMock{
		EventsFunc: func(ctx context.Context) <-chan struct{} {
			res := make(chan struct{}, 1)
			res <- struct{}{}
			return res
		},
		ListFunc: func() ([]URLMapper, error) {
			return []URLMapper{
				{SrcMatch: *regexp.MustCompile("^/api/svc1/(.*)"), Dst: "http://127.0.0.1:8080/blah1/$1"},
				{Server: "m.example.com", SrcMatch: *regexp.MustCompile("^/api/svc2/(.*)"),
					Dst: "http://127.0.0.2:8080/blah2/$1/abc"},
			}, nil
		},
		IDFunc: func() ProviderID {
			return PIFile
		},
	}
	p2 := &ProviderMock{
		EventsFunc: func(ctx context.Context) <-chan struct{} {
			return make(chan struct{}, 1)
		},
		ListFunc: func() ([]URLMapper, error) {
			return []URLMapper{
				{SrcMatch: *regexp.MustCompile("/api/svc3/xyz"), Dst: "http://127.0.0.3:8080/blah3/xyz"},
			}, nil
		},
		IDFunc: func() ProviderID {
			return PIDocker
		},
	}
	svc := NewService([]Provider{p1, p2})

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	err := svc.Run(ctx)
	require.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
	assert.Equal(t, 3, len(svc.mappers))

	tbl := []struct {
		server, src string
		dest        string
		ok          bool
	}{
		{"example.com", "/api/svc3/xyz", "http://127.0.0.3:8080/blah3/xyz", true},
		{"abc.example.com", "/api/svc1/1234", "http://127.0.0.1:8080/blah1/1234", true},
		{"zzz.example.com", "/aaa/api/svc1/1234", "/aaa/api/svc1/1234", false},
		{"m.example.com", "/api/svc2/1234", "http://127.0.0.2:8080/blah2/1234/abc", true},
		{"m1.example.com", "/api/svc2/1234", "/api/svc2/1234", false},
	}

	for i, tt := range tbl {
		tt := tt
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			res, ok := svc.Match(tt.server, tt.src)
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.dest, res)
		})
	}
}

func TestService_Servers(t *testing.T) {
	p1 := &ProviderMock{
		EventsFunc: func(ctx context.Context) <-chan struct{} {
			res := make(chan struct{}, 1)
			res <- struct{}{}
			return res
		},
		ListFunc: func() ([]URLMapper, error) {
			return []URLMapper{
				{SrcMatch: *regexp.MustCompile("^/api/svc1/(.*)"), Dst: "http://127.0.0.1:8080/blah1/$1"},
				{Server: "m.example.com", SrcMatch: *regexp.MustCompile("^/api/svc2/(.*)"),
					Dst: "http://127.0.0.2:8080/blah2/$1/abc"},
			}, nil
		},
		IDFunc: func() ProviderID {
			return PIFile
		},
	}
	p2 := &ProviderMock{
		EventsFunc: func(ctx context.Context) <-chan struct{} {
			return make(chan struct{}, 1)
		},
		ListFunc: func() ([]URLMapper, error) {
			return []URLMapper{
				{Server: "xx.reproxy.io", SrcMatch: *regexp.MustCompile("/api/svc3/xyz"), Dst: "http://127.0.0.3:8080/blah3/xyz"},
			}, nil
		},
		IDFunc: func() ProviderID {
			return PIDocker
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	svc := NewService([]Provider{p1, p2})
	err := svc.Run(ctx)
	require.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
	assert.Equal(t, 3, len(svc.mappers))

	servers := svc.Servers()
	assert.Equal(t, []string{"m.example.com", "xx.reproxy.io"}, servers)

}

func TestService_extendRule(t *testing.T) {

	tbl := []struct {
		inp URLMapper
		out URLMapper
	}{
		{
			URLMapper{SrcMatch: *regexp.MustCompile("/")},
			URLMapper{SrcMatch: *regexp.MustCompile("^/(.*)"), Dst: "/$1"},
		},
		{
			URLMapper{Server: "m.example.com", PingURL: "http://example.com/ping", ProviderID: "docker",
				SrcMatch: *regexp.MustCompile("/api/blah/"), Dst: "http://localhost:8080/"},
			URLMapper{Server: "m.example.com", PingURL: "http://example.com/ping", ProviderID: "docker",
				SrcMatch: *regexp.MustCompile("^/api/blah/(.*)"), Dst: "http://localhost:8080/$1"},
		},
		{
			URLMapper{Server: "m.example.com", PingURL: "http://example.com/ping", ProviderID: "docker",
				SrcMatch: *regexp.MustCompile("/api/blah/(.*)/xxx/(.*_)"), Dst: "http://localhost:8080/$1/$2"},
			URLMapper{Server: "m.example.com", PingURL: "http://example.com/ping", ProviderID: "docker",
				SrcMatch: *regexp.MustCompile("/api/blah/(.*)/xxx/(.*_)"), Dst: "http://localhost:8080/$1/$2"},
		},
		{
			URLMapper{Server: "m.example.com", PingURL: "http://example.com/ping", ProviderID: "docker",
				SrcMatch: *regexp.MustCompile("/api/blah"), Dst: "http://localhost:8080/xxx"},
			URLMapper{Server: "m.example.com", PingURL: "http://example.com/ping", ProviderID: "docker",
				SrcMatch: *regexp.MustCompile("/api/blah"), Dst: "http://localhost:8080/xxx"},
		},
	}

	svc := &Service{}
	for i, tt := range tbl {
		tt := tt
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			res := svc.extendRule(tt.inp)
			assert.Equal(t, tt.out, res)
		})
	}

}
