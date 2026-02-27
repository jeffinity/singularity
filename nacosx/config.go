package nacosx

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/go-kratos/kratos/v2/config"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"github.com/pkg/errors"
)

// ConfigSource implements kratos config.Source using a provided Nacos SDK v2 client.
type ConfigSource struct {
	client config_client.IConfigClient
	opts   options
}

// NewConfigSource creates a Nacos config source with the given client and shared options.
// Default DataID is "application.yaml" and Group is constant.DEFAULT_GROUP.
// Use shared Option functions (WithDataID, WithGroup, WithCluster, WithPrefix, etc.) to customize.
func NewConfigSource(cli config_client.IConfigClient, opts ...Option) config.Source {
	// default options
	_opts := options{
		dataID: "application.yaml",
		group:  constant.DEFAULT_GROUP,
	}
	// apply shared options
	for _, o := range opts {
		o(&_opts)
	}
	return &ConfigSource{
		client: cli,
		opts:   _opts,
	}
}

// Load pulls the current configuration from Nacos.
func (s *ConfigSource) Load() ([]*config.KeyValue, error) {
	data, err := s.client.GetConfig(vo.ConfigParam{
		DataId: s.opts.dataID,
		Group:  s.opts.group,
	})
	if err != nil {
		return nil, errors.WithMessage(err, "GetConfig failed")
	}
	return []*config.KeyValue{
		{
			Key:    s.opts.dataID,
			Value:  []byte(data),
			Format: strings.TrimPrefix(filepath.Ext(s.opts.dataID), "."),
		},
	}, nil
}

// Watch listens for config changes and returns a Watcher.
func (s *ConfigSource) Watch() (config.Watcher, error) {
	ctx, cancel := context.WithCancel(context.Background())
	w := &configWatcher{
		opts:       s.opts,
		ctx:        ctx,
		cancel:     cancel,
		ch:         make(chan string, 1),
		cancelFunc: s.client.CancelListenConfig,
	}
	// register listener
	err := s.client.ListenConfig(vo.ConfigParam{
		DataId: s.opts.dataID,
		Group:  s.opts.group,
		OnChange: func(_, grp, dataID, data string) {
			if dataID == s.opts.dataID && grp == s.opts.group {
				w.ch <- data
			}
		},
	})
	if err != nil {
		cancel()
		return nil, errors.WithMessage(err, "ListenConfig failed")
	}
	return w, nil
}

// configWatcher implements config.Watcher for Nacos.
type configWatcher struct {
	opts       options
	ctx        context.Context
	cancel     context.CancelFunc
	ch         chan string
	cancelFunc func(vo.ConfigParam) error
}

// Next returns the next configuration snapshot.
func (w *configWatcher) Next() ([]*config.KeyValue, error) {
	select {
	case <-w.ctx.Done():
		return nil, w.ctx.Err()
	case data := <-w.ch:
		return []*config.KeyValue{
			{
				Key:    w.opts.dataID,
				Value:  []byte(data),
				Format: strings.TrimPrefix(filepath.Ext(w.opts.dataID), "."),
			},
		}, nil
	}
}

// Stop stops listening and closes the Watcher.
func (w *configWatcher) Stop() error {
	err := w.cancelFunc(vo.ConfigParam{DataId: w.opts.dataID, Group: w.opts.group})
	w.cancel()
	if err != nil {
		return errors.WithMessage(err, "CancelListenConfig failed")
	}
	return nil
}

// Close is an alias for Stop.
func (w *configWatcher) Close() error {
	return w.Stop()
}
