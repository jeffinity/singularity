package nacosx

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strconv"

	"github.com/go-kratos/kratos/v2/registry"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"github.com/pkg/errors"
)

var ErrServiceInstanceNameEmpty = errors.New("kratos/nacos: ServiceInstance.Name can not be empty")

var (
	_ registry.Registrar = (*Registry)(nil)
	_ registry.Discovery = (*Registry)(nil)
	_ registry.Watcher   = (*registryWatcher)(nil)
)

// Registry is a Nacos registry.
type Registry struct {
	opts options
	cli  naming_client.INamingClient
}

// New creates a new Nacos registry.
func New(cli naming_client.INamingClient, opts ...Option) *Registry {
	op := options{
		prefix:  "/microservices",
		cluster: "DEFAULT",
		group:   constant.DEFAULT_GROUP,
		weight:  100,
		kind:    "grpc",
	}
	for _, o := range opts {
		o(&op)
	}
	return &Registry{opts: op, cli: cli}
}

// Register registers a service instance.
func (r *Registry) Register(_ context.Context, si *registry.ServiceInstance) error {
	if si.Name == "" {
		return ErrServiceInstanceNameEmpty
	}
	for _, endpoint := range si.Endpoints {
		u, err := url.Parse(endpoint)
		if err != nil {
			return err
		}
		host, port, err := net.SplitHostPort(u.Host)
		if err != nil {
			return err
		}
		p, err := strconv.Atoi(port)
		if err != nil {
			return err
		}
		// build metadata
		weight := r.opts.weight
		var md map[string]string
		if si.Metadata == nil {
			md = map[string]string{
				"kind":    u.Scheme,
				"version": si.Version,
			}
		} else {
			md = make(map[string]string, len(si.Metadata)+2)
			for k, v := range si.Metadata {
				md[k] = v
			}
			md["kind"] = u.Scheme
			md["version"] = si.Version
			if wv, ok := si.Metadata["weight"]; ok {
				if w2, err := strconv.ParseFloat(wv, 64); err == nil {
					weight = w2
				}
			}
		}

		// register instance
		if _, err := r.cli.RegisterInstance(vo.RegisterInstanceParam{
			Ip:          host,
			Port:        uint64(p),
			ServiceName: si.Name + "." + u.Scheme,
			Weight:      weight,
			Enable:      true,
			Healthy:     true,
			Ephemeral:   true,
			Metadata:    md,
			ClusterName: r.opts.cluster,
			GroupName:   r.opts.group,
		}); err != nil {
			return errors.WithMessage(err, "register instance failed:")
		}
	}
	return nil
}

// Deregister deregisters a service instance.
func (r *Registry) Deregister(_ context.Context, si *registry.ServiceInstance) error {
	for _, endpoint := range si.Endpoints {
		u, err := url.Parse(endpoint)
		if err != nil {
			return err
		}
		host, port, err := net.SplitHostPort(u.Host)
		if err != nil {
			return err
		}
		p, err := strconv.Atoi(port)
		if err != nil {
			return err
		}
		if _, err := r.cli.DeregisterInstance(vo.DeregisterInstanceParam{
			Ip:          host,
			Port:        uint64(p),
			ServiceName: si.Name + "." + u.Scheme,
			Cluster:     r.opts.cluster,
			GroupName:   r.opts.group,
			Ephemeral:   true,
		}); err != nil {
			return err
		}
	}
	return nil
}

// Watch creates a service registryWatcher.
func (r *Registry) Watch(ctx context.Context, serviceName string) (registry.Watcher, error) {
	return newRegistryWatcher(ctx, r.cli, serviceName, r.opts.group, r.opts.kind, []string{r.opts.cluster})
}

// GetService retrieves instances of a service.
func (r *Registry) GetService(_ context.Context, serviceName string) ([]*registry.ServiceInstance, error) {
	insts, err := r.cli.SelectInstances(vo.SelectInstancesParam{
		ServiceName: serviceName,
		GroupName:   r.opts.group,
		HealthyOnly: true,
	})
	if err != nil {
		return nil, err
	}
	var items []*registry.ServiceInstance
	for _, in := range insts {
		kind := r.opts.kind
		if k, ok := in.Metadata["kind"]; ok {
			kind = k
		}
		items = append(items, &registry.ServiceInstance{
			ID:        in.InstanceId,
			Name:      in.ServiceName,
			Version:   in.Metadata["version"],
			Metadata:  in.Metadata,
			Endpoints: []string{fmt.Sprintf("%s://%s:%d", kind, in.Ip, in.Port)},
		})
	}
	return items, nil
}

// registryWatcher watches for instance changes.
type registryWatcher struct {
	serviceName string
	clusters    []string
	groupName   string
	ctx         context.Context
	cancel      context.CancelFunc
	watchChan   chan struct{}
	cli         naming_client.INamingClient
	kind        string
	subParam    *vo.SubscribeParam
}

func newRegistryWatcher(ctx context.Context, cli naming_client.INamingClient, serviceName, groupName, kind string, clusters []string) (*registryWatcher, error) {
	w := &registryWatcher{
		serviceName: serviceName,
		clusters:    clusters,
		groupName:   groupName,
		cli:         cli,
		kind:        kind,
		watchChan:   make(chan struct{}, 1),
	}
	w.ctx, w.cancel = context.WithCancel(ctx)

	sub := &vo.SubscribeParam{
		ServiceName: serviceName,
		Clusters:    clusters,
		GroupName:   groupName,
		SubscribeCallback: func(instances []model.Instance, err error) {
			select {
			case w.watchChan <- struct{}{}:
			default:
			}
		},
	}
	w.subParam = sub
	if err := cli.Subscribe(sub); err != nil {
		return nil, err
	}
	// initial trigger
	select {
	case w.watchChan <- struct{}{}:
	default:
	}
	return w, nil
}

// Next returns updated instances when changes occur.
func (w *registryWatcher) Next() ([]*registry.ServiceInstance, error) {
	select {
	case <-w.ctx.Done():
		return nil, w.ctx.Err()
	case <-w.watchChan:
	}
	svc, err := w.cli.GetService(vo.GetServiceParam{
		ServiceName: w.serviceName,
		Clusters:    w.clusters,
		GroupName:   w.groupName,
	})
	if err != nil {
		return nil, err
	}
	var items []*registry.ServiceInstance
	for _, in := range svc.Hosts {
		kind := w.kind
		if k, ok := in.Metadata["kind"]; ok {
			kind = k
		}
		items = append(items, &registry.ServiceInstance{
			ID:        in.InstanceId,
			Name:      svc.Name,
			Version:   in.Metadata["version"],
			Metadata:  in.Metadata,
			Endpoints: []string{fmt.Sprintf("%s://%s:%d", kind, in.Ip, in.Port)},
		})
	}
	return items, nil
}

// Stop stops the registryWatcher.
func (w *registryWatcher) Stop() error {
	err := w.cli.Unsubscribe(w.subParam)
	w.cancel()
	return err
}
