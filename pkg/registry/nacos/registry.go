package nacos

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"math"
	"net"
	"net/url"
	"strconv"

	"github.com/nacos-group/nacos-sdk-go/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/vo"

	"github.com/go-kratos/kratos/v2/registry"
)

var (
	ErrServiceInstanceNameEmpty = errors.New("kratos/nacos: ServiceInstance.Name can not be empty")
	ErrInvalidPort              = errors.New("kratos/nacos: invalid port number")
)

var (
	_ registry.Registrar = (*Registry)(nil)
	_ registry.Discovery = (*Registry)(nil)
)

type options struct {
	prefix  string
	weight  float64
	cluster string
	group   string
	kind    string
}

// Option is nacos option.
type Option func(o *options)

// WithPrefix with prefix path.
func WithPrefix(prefix string) Option {
	return func(o *options) { o.prefix = prefix }
}

// WithWeight with weight option.
func WithWeight(weight float64) Option {
	return func(o *options) { o.weight = weight }
}

// WithCluster with cluster option.
func WithCluster(cluster string) Option {
	return func(o *options) { o.cluster = cluster }
}

// WithGroup with group option.
func WithGroup(group string) Option {
	return func(o *options) { o.group = group }
}

// WithDefaultKind with default kind option.
func WithDefaultKind(kind string) Option {
	return func(o *options) { o.kind = kind }
}

// Registry is nacos registry.
type Registry struct {
	opts options
	cli  naming_client.INamingClient
}

// New new a nacos registry.
func New(cli naming_client.INamingClient, opts ...Option) (r *Registry) {
	op := options{
		prefix:  "/microservices",
		cluster: "DEFAULT",
		group:   constant.DEFAULT_GROUP,
		weight:  100,
		kind:    "grpc",
	}
	for _, option := range opts {
		option(&op)
	}
	return &Registry{
		opts: op,
		cli:  cli,
	}
}

// buildMetadata builds the metadata map for registration.
func (r *Registry) buildMetadata(si *registry.ServiceInstance, scheme string) (metadata map[string]string, weight float64) {
	weight = r.opts.weight
	if si.Metadata == nil {
		return map[string]string{
			"kind":    scheme,
			"version": si.Version,
		}, weight
	}
	rmd := maps.Clone(si.Metadata)
	rmd["kind"] = scheme
	rmd["version"] = si.Version
	if w, ok := si.Metadata["weight"]; ok {
		if parsed, err := strconv.ParseFloat(w, 64); err == nil {
			weight = parsed
		}
	}
	return rmd, weight
}

// parseEndpoint parses an endpoint string and returns host and port.
func parseEndpoint(endpoint string) (u *url.URL, host string, port uint64, err error) {
	u, err = url.Parse(endpoint)
	if err != nil {
		return nil, "", 0, err
	}
	var portStr string
	host, portStr, err = net.SplitHostPort(u.Host)
	if err != nil {
		return nil, "", 0, err
	}
	port, err = strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		return nil, "", 0, err
	}
	return u, host, port, nil
}

// Register the registration.
func (r *Registry) Register(_ context.Context, si *registry.ServiceInstance) error {
	if si.Name == "" {
		return ErrServiceInstanceNameEmpty
	}
	for _, endpoint := range si.Endpoints {
		u, host, p, err := parseEndpoint(endpoint)
		if err != nil {
			return err
		}
		rmd, weight := r.buildMetadata(si, u.Scheme)
		_, e := r.cli.RegisterInstance(vo.RegisterInstanceParam{
			Ip:          host,
			Port:        p,
			ServiceName: si.Name + "." + u.Scheme,
			Weight:      weight,
			Enable:      true,
			Healthy:     true,
			Ephemeral:   true,
			Metadata:    rmd,
			ClusterName: r.opts.cluster,
			GroupName:   r.opts.group,
		})
		if e != nil {
			return fmt.Errorf("RegisterInstance err %w, endpoint: %s", e, endpoint)
		}
	}
	return nil
}

// Deregister the registration.
func (r *Registry) Deregister(_ context.Context, service *registry.ServiceInstance) error {
	for _, endpoint := range service.Endpoints {
		u, host, p, err := parseEndpoint(endpoint)
		if err != nil {
			return err
		}
		if _, err = r.cli.DeregisterInstance(vo.DeregisterInstanceParam{
			Ip:          host,
			Port:        p,
			ServiceName: service.Name + "." + u.Scheme,
			GroupName:   r.opts.group,
			Cluster:     r.opts.cluster,
			Ephemeral:   true,
		}); err != nil {
			return err
		}
	}
	return nil
}

// Watch creates a watcher according to the service name.
func (r *Registry) Watch(ctx context.Context, serviceName string) (registry.Watcher, error) {
	return newWatcher(ctx, r.cli, serviceName, r.opts.group, r.opts.kind, []string{r.opts.cluster})
}

// GetService return the service instances in memory according to the service name.
func (r *Registry) GetService(_ context.Context, serviceName string) ([]*registry.ServiceInstance, error) {
	res, err := r.cli.SelectInstances(vo.SelectInstancesParam{
		ServiceName: serviceName,
		GroupName:   r.opts.group,
		HealthyOnly: true,
	})
	if err != nil {
		return nil, err
	}
	items := make([]*registry.ServiceInstance, 0, len(res))
	for _, in := range res {
		kind := r.opts.kind
		weight := r.opts.weight
		if k, ok := in.Metadata["kind"]; ok {
			kind = k
		}
		if in.Weight > 0 {
			weight = in.Weight
		}

		r := &registry.ServiceInstance{
			ID:        in.InstanceId,
			Name:      in.ServiceName,
			Version:   in.Metadata["version"],
			Metadata:  in.Metadata,
			Endpoints: []string{kind + "://" + net.JoinHostPort(in.Ip, strconv.FormatUint(in.Port, 10))},
		}
		r.Metadata["weight"] = strconv.FormatInt(int64(math.Ceil(weight)), 10)
		items = append(items, r)
	}
	return items, nil
}
