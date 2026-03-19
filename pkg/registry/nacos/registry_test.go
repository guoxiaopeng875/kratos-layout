package nacos

import (
	"context"
	"errors"
	"testing"

	"github.com/go-kratos/kratos/v2/registry"
	"github.com/nacos-group/nacos-sdk-go/model"
	"github.com/nacos-group/nacos-sdk-go/vo"
)

func TestNew(t *testing.T) {
	r := New(nil)
	if r == nil {
		t.Fatal("New returned nil")
	}
	if r.opts.prefix != "/microservices" {
		t.Errorf("expected prefix /microservices, got %s", r.opts.prefix)
	}
	if r.opts.cluster != "DEFAULT" {
		t.Errorf("expected cluster DEFAULT, got %s", r.opts.cluster)
	}
	if r.opts.weight != 100 {
		t.Errorf("expected weight 100, got %f", r.opts.weight)
	}
	if r.opts.kind != "grpc" {
		t.Errorf("expected kind grpc, got %s", r.opts.kind)
	}
}

func TestNew_WithOptions(t *testing.T) {
	r := New(nil,
		func(o *options) { o.cluster = "CUSTOM" },
		func(o *options) { o.weight = 50 },
		func(o *options) { o.kind = "http" },
	)
	if r.opts.cluster != "CUSTOM" {
		t.Errorf("expected cluster CUSTOM, got %s", r.opts.cluster)
	}
	if r.opts.weight != 50 {
		t.Errorf("expected weight 50, got %f", r.opts.weight)
	}
	if r.opts.kind != "http" {
		t.Errorf("expected kind http, got %s", r.opts.kind)
	}
}

func TestParseEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		wantHost string
		wantPort uint64
		wantErr  bool
	}{
		{
			name:     "grpc endpoint",
			endpoint: "grpc://127.0.0.1:9000",
			wantHost: "127.0.0.1",
			wantPort: 9000,
		},
		{
			name:     "http endpoint",
			endpoint: "http://localhost:8080",
			wantHost: "localhost",
			wantPort: 8080,
		},
		{
			name:     "invalid url",
			endpoint: "://invalid",
			wantErr:  true,
		},
		{
			name:     "no port",
			endpoint: "grpc://localhost",
			wantErr:  true,
		},
		{
			name:     "invalid port",
			endpoint: "grpc://localhost:abc",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, host, port, err := parseEndpoint(tt.endpoint)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if host != tt.wantHost {
				t.Errorf("host = %s, want %s", host, tt.wantHost)
			}
			if port != tt.wantPort {
				t.Errorf("port = %d, want %d", port, tt.wantPort)
			}
			if u == nil {
				t.Error("url should not be nil")
			}
		})
	}
}

func TestBuildMetadata(t *testing.T) {
	r := New(nil)

	t.Run("nil metadata", func(t *testing.T) {
		si := &registry.ServiceInstance{
			Version: "v1.0",
		}
		md, weight := r.buildMetadata(si, "grpc")
		if md["kind"] != "grpc" {
			t.Errorf("kind = %s, want grpc", md["kind"])
		}
		if md["version"] != "v1.0" {
			t.Errorf("version = %s, want v1.0", md["version"])
		}
		if weight != 100 {
			t.Errorf("weight = %f, want 100", weight)
		}
	})

	t.Run("with metadata", func(t *testing.T) {
		si := &registry.ServiceInstance{
			Version: "v2.0",
			Metadata: map[string]string{
				"env": "test",
			},
		}
		md, weight := r.buildMetadata(si, "http")
		if md["kind"] != "http" {
			t.Errorf("kind = %s, want http", md["kind"])
		}
		if md["version"] != "v2.0" {
			t.Errorf("version = %s, want v2.0", md["version"])
		}
		if md["env"] != "test" {
			t.Errorf("env = %s, want test", md["env"])
		}
		if weight != 100 {
			t.Errorf("weight = %f, want 100", weight)
		}
	})

	t.Run("with weight in metadata", func(t *testing.T) {
		si := &registry.ServiceInstance{
			Version: "v1.0",
			Metadata: map[string]string{
				"weight": "50",
			},
		}
		_, weight := r.buildMetadata(si, "grpc")
		if weight != 50 {
			t.Errorf("weight = %f, want 50", weight)
		}
	})

	t.Run("with invalid weight in metadata", func(t *testing.T) {
		si := &registry.ServiceInstance{
			Version: "v1.0",
			Metadata: map[string]string{
				"weight": "invalid",
			},
		}
		_, weight := r.buildMetadata(si, "grpc")
		if weight != 100 {
			t.Errorf("weight = %f, want 100 (default)", weight)
		}
	})
}

func TestRegister_EmptyName(t *testing.T) {
	r := New(nil)
	si := &registry.ServiceInstance{Name: ""}
	err := r.Register(context.TODO(), si)
	if err == nil {
		t.Fatal("expected error for empty name")
	}
	if !errors.Is(err, ErrServiceInstanceNameEmpty) {
		t.Errorf("expected ErrServiceInstanceNameEmpty, got %v", err)
	}
}

func TestRegister_InvalidEndpoint(t *testing.T) {
	r := New(nil)
	si := &registry.ServiceInstance{
		Name:      "test-service",
		Endpoints: []string{"://invalid"},
	}
	err := r.Register(context.TODO(), si)
	if err == nil {
		t.Fatal("expected error for invalid endpoint")
	}
}

func TestDeregister_InvalidEndpoint(t *testing.T) {
	r := New(nil)
	si := &registry.ServiceInstance{
		Endpoints: []string{"://invalid"},
	}
	err := r.Deregister(context.TODO(), si)
	if err == nil {
		t.Fatal("expected error for invalid endpoint")
	}
}

func TestRegister_Success(t *testing.T) {
	var captured vo.RegisterInstanceParam
	mock := &mockNamingClient{
		registerInstanceFn: func(param vo.RegisterInstanceParam) (bool, error) {
			captured = param
			return true, nil
		},
	}
	r := New(mock)
	si := &registry.ServiceInstance{
		Name:      "test-service",
		Version:   "v1.0",
		Endpoints: []string{"grpc://10.0.0.1:9000"},
	}

	err := r.Register(context.Background(), si)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if captured.Ip != "10.0.0.1" {
		t.Errorf("Ip = %s, want 10.0.0.1", captured.Ip)
	}
	if captured.Port != 9000 {
		t.Errorf("Port = %d, want 9000", captured.Port)
	}
	if captured.ServiceName != "test-service.grpc" {
		t.Errorf("ServiceName = %s, want test-service.grpc", captured.ServiceName)
	}
	if !captured.Enable {
		t.Error("Enable should be true")
	}
	if !captured.Healthy {
		t.Error("Healthy should be true")
	}
	if !captured.Ephemeral {
		t.Error("Ephemeral should be true")
	}
}

func TestRegister_MultipleEndpoints(t *testing.T) {
	var count int
	mock := &mockNamingClient{
		registerInstanceFn: func(param vo.RegisterInstanceParam) (bool, error) {
			count++
			return true, nil
		},
	}
	r := New(mock)
	si := &registry.ServiceInstance{
		Name:      "svc",
		Endpoints: []string{"grpc://10.0.0.1:9000", "http://10.0.0.1:8080"},
	}

	err := r.Register(context.Background(), si)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 registrations, got %d", count)
	}
}

func TestRegister_ClientError(t *testing.T) {
	mock := &mockNamingClient{
		registerInstanceFn: func(param vo.RegisterInstanceParam) (bool, error) {
			return false, errors.New("register failed")
		},
	}
	r := New(mock)
	si := &registry.ServiceInstance{
		Name:      "svc",
		Endpoints: []string{"grpc://10.0.0.1:9000"},
	}

	err := r.Register(context.Background(), si)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, errors.Unwrap(err)) && err.Error() == "" {
		t.Error("expected wrapped error")
	}
}

func TestDeregister_Success(t *testing.T) {
	var captured vo.DeregisterInstanceParam
	mock := &mockNamingClient{
		deregisterInstanceFn: func(param vo.DeregisterInstanceParam) (bool, error) {
			captured = param
			return true, nil
		},
	}
	r := New(mock)
	si := &registry.ServiceInstance{
		Name:      "test-service",
		Endpoints: []string{"grpc://10.0.0.1:9000"},
	}

	err := r.Deregister(context.Background(), si)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if captured.Ip != "10.0.0.1" {
		t.Errorf("Ip = %s, want 10.0.0.1", captured.Ip)
	}
	if captured.Port != 9000 {
		t.Errorf("Port = %d, want 9000", captured.Port)
	}
	if captured.ServiceName != "test-service.grpc" {
		t.Errorf("ServiceName = %s, want test-service.grpc", captured.ServiceName)
	}
}

func TestDeregister_ClientError(t *testing.T) {
	mock := &mockNamingClient{
		deregisterInstanceFn: func(param vo.DeregisterInstanceParam) (bool, error) {
			return false, errors.New("deregister failed")
		},
	}
	r := New(mock)
	si := &registry.ServiceInstance{
		Name:      "svc",
		Endpoints: []string{"grpc://10.0.0.1:9000"},
	}

	err := r.Deregister(context.Background(), si)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetService_Success(t *testing.T) {
	mock := &mockNamingClient{
		selectInstancesFn: func(param vo.SelectInstancesParam) ([]model.Instance, error) {
			return []model.Instance{
				{
					InstanceId: "inst-1",
					Ip:         "10.0.0.1",
					Port:       9000,
					Weight:     80,
					Metadata: map[string]string{
						"kind":    "grpc",
						"version": "v1.0",
					},
					ServiceName: "test-svc",
				},
				{
					InstanceId:  "inst-2",
					Ip:          "10.0.0.2",
					Port:        9000,
					Weight:      0, // zero weight, should use default
					Metadata:    map[string]string{},
					ServiceName: "test-svc",
				},
			}, nil
		},
	}
	r := New(mock)

	items, err := r.GetService(context.Background(), "test-svc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	// First instance has kind from metadata
	if items[0].ID != "inst-1" {
		t.Errorf("ID = %s, want inst-1", items[0].ID)
	}
	if items[0].Version != "v1.0" {
		t.Errorf("Version = %s, want v1.0", items[0].Version)
	}
	if items[0].Endpoints[0] != "grpc://10.0.0.1:9000" {
		t.Errorf("Endpoint = %s, want grpc://10.0.0.1:9000", items[0].Endpoints[0])
	}

	// Second instance uses default kind and weight
	if items[1].Endpoints[0] != "grpc://10.0.0.2:9000" {
		t.Errorf("Endpoint = %s, want grpc://10.0.0.2:9000", items[1].Endpoints[0])
	}
}

func TestGetService_Error(t *testing.T) {
	mock := &mockNamingClient{
		selectInstancesFn: func(param vo.SelectInstancesParam) ([]model.Instance, error) {
			return nil, errors.New("select failed")
		},
	}
	r := New(mock)

	_, err := r.GetService(context.Background(), "test-svc")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestWatch(t *testing.T) {
	mock := &mockNamingClient{}
	r := New(mock)

	w, err := r.Watch(context.Background(), "test-svc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w == nil {
		t.Fatal("watcher should not be nil")
	}
	if err := w.Stop(); err != nil {
		t.Fatalf("Stop error: %v", err)
	}
}

func TestWatcher_Next_ContextCanceled(t *testing.T) {
	mock := &mockNamingClient{}
	ctx, cancel := context.WithCancel(context.Background())

	w, err := newWatcher(ctx, mock, "test-svc", "DEFAULT_GROUP", "grpc", []string{"DEFAULT"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// First call to Next should succeed (watchChan has initial signal)
	mock.getServiceFn = func(param vo.GetServiceParam) (model.Service, error) {
		return model.Service{
			Name: "test-svc",
			Hosts: []model.Instance{
				{
					InstanceId: "inst-1",
					Ip:         "10.0.0.1",
					Port:       9000,
					Metadata: map[string]string{
						"kind":    "grpc",
						"version": "v1.0",
					},
				},
			},
		}, nil
	}

	items, err := w.Next()
	if err != nil {
		t.Fatalf("unexpected error on first Next: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Endpoints[0] != "grpc://10.0.0.1:9000" {
		t.Errorf("Endpoint = %s, want grpc://10.0.0.1:9000", items[0].Endpoints[0])
	}

	// Cancel context, next call should return error
	cancel()
	_, err = w.Next()
	if err == nil {
		t.Fatal("expected error after cancel")
	}
}

func TestWatcher_Next_GetServiceError(t *testing.T) {
	mock := &mockNamingClient{
		getServiceFn: func(param vo.GetServiceParam) (model.Service, error) {
			return model.Service{}, errors.New("get service failed")
		},
	}

	w, err := newWatcher(context.Background(), mock, "test-svc", "DEFAULT_GROUP", "grpc", []string{"DEFAULT"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer w.Stop()

	_, err = w.Next()
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestWatcher_Next_WithKindFromMetadata(t *testing.T) {
	mock := &mockNamingClient{
		getServiceFn: func(param vo.GetServiceParam) (model.Service, error) {
			return model.Service{
				Name: "test-svc",
				Hosts: []model.Instance{
					{
						InstanceId: "inst-1",
						Ip:         "10.0.0.1",
						Port:       8080,
						Metadata: map[string]string{
							"kind":    "http",
							"version": "v2.0",
						},
					},
				},
			}, nil
		},
	}

	w, err := newWatcher(context.Background(), mock, "test-svc", "DEFAULT_GROUP", "grpc", []string{"DEFAULT"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer w.Stop()

	items, err := w.Next()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if items[0].Endpoints[0] != "http://10.0.0.1:8080" {
		t.Errorf("Endpoint = %s, want http://10.0.0.1:8080", items[0].Endpoints[0])
	}
	if items[0].Version != "v2.0" {
		t.Errorf("Version = %s, want v2.0", items[0].Version)
	}
}

func TestWatcher_Stop(t *testing.T) {
	var unsubscribed bool
	mock := &mockNamingClient{
		unsubscribeFn: func(param *vo.SubscribeParam) error {
			unsubscribed = true
			return nil
		},
	}

	w, err := newWatcher(context.Background(), mock, "test-svc", "DEFAULT_GROUP", "grpc", []string{"DEFAULT"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = w.Stop()
	if err != nil {
		t.Fatalf("Stop error: %v", err)
	}
	if !unsubscribed {
		t.Error("expected Unsubscribe to be called")
	}
}

func TestWatcher_Stop_Error(t *testing.T) {
	mock := &mockNamingClient{
		unsubscribeFn: func(param *vo.SubscribeParam) error {
			return errors.New("unsubscribe failed")
		},
	}

	w, err := newWatcher(context.Background(), mock, "test-svc", "DEFAULT_GROUP", "grpc", []string{"DEFAULT"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = w.Stop()
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNewWatcher_SubscribeError(t *testing.T) {
	mock := &mockNamingClient{
		subscribeFn: func(param *vo.SubscribeParam) error {
			return errors.New("subscribe failed")
		},
	}

	w, err := newWatcher(context.Background(), mock, "test-svc", "DEFAULT_GROUP", "grpc", []string{"DEFAULT"})
	if err == nil {
		t.Fatal("expected error")
	}
	// Even on error, watcher is returned (SDK behavior)
	_ = w
}

func TestRegister_NoEndpoints(t *testing.T) {
	mock := &mockNamingClient{}
	r := New(mock)
	si := &registry.ServiceInstance{
		Name:      "test-service",
		Endpoints: []string{},
	}

	err := r.Register(context.Background(), si)
	if err != nil {
		t.Fatalf("unexpected error with empty endpoints: %v", err)
	}
}

func TestDeregister_NoEndpoints(t *testing.T) {
	mock := &mockNamingClient{}
	r := New(mock)
	si := &registry.ServiceInstance{
		Name:      "test-service",
		Endpoints: []string{},
	}

	err := r.Deregister(context.Background(), si)
	if err != nil {
		t.Fatalf("unexpected error with empty endpoints: %v", err)
	}
}
