package nacosx

import (
	"github.com/go-kratos/kratos/v2/config"
	"github.com/google/wire"
	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"github.com/pkg/errors"

	"github.com/jeffinity/singularity/friendly"
)

var ProviderSet = wire.NewSet(
	NewNamingClient,
	NewConfigClient,
	NewRegistryEngine,
)

// Conf holds the Nacos configuration.
// Addr, Port, Username, Password specify the server connection.
// ClusterName serves as both the key prefix and cluster name for service discovery/registration.
// NamespaceId isolates environments or tenants; GroupId logically groups configs and services.
// DataId identifies specific config items in Nacos.
type Conf struct {
	Addr        string // Nacos server address
	Port        uint64 // Nacos server port
	Username    string // optional auth
	Password    string // optional auth
	ClusterName string // Prefix and cluster name for Nacos registrations
	NamespaceId string // for both config and naming
	GroupId     string // naming group / config group
	DataId      string // config data ID
	LogDir      string // config data ID
	Weight      int32  // config data ID
	Scheme      string // http / https (optional; auto set to https when TLS enabled)
	// TLS / mTLS
	EnableTLS   bool   // enable TLS (set true for https)
	TLSTrustAll bool   // if true, skip verifying server cert (NOT recommended for production)
	TLSCAFile   string // PEM CA bundle path for verifying server cert
	TLSCertFile string // PEM client certificate path (mTLS)
	TLSKeyFile  string // PEM client private key path (mTLS)
}

func newNacosClientConfig(cfg Conf) *constant.ClientConfig {

	cc := &constant.ClientConfig{
		NamespaceId:         cfg.NamespaceId,
		TimeoutMs:           5000,
		NotLoadCacheAtStart: true,
		CacheDir:            cfg.LogDir,
		LogDir:              cfg.LogDir,
		UpdateThreadNum:     20,
		LogLevel:            "error",
		LogRollingConfig: &constant.ClientLogRollingConfig{
			MaxSize:    100,
			MaxAge:     7,
			MaxBackups: 7,
			Compress:   true,
		},
	}

	if cfg.Username != "" {
		cc.Username = cfg.Username
		cc.Password = cfg.Password
	}

	// TLS / mTLS
	if cfg.EnableTLS {
		cc.TLSCfg = constant.TLSConfig{
			Enable:    true,
			Appointed: true,
			TrustAll:  cfg.TLSTrustAll,
			CaFile:    cfg.TLSCAFile,
			CertFile:  cfg.TLSCertFile,
			KeyFile:   cfg.TLSKeyFile,
		}
	}

	return cc
}

func newNacosServerConfig(cfg Conf) []constant.ServerConfig {
	sc := []constant.ServerConfig{
		*constant.NewServerConfig(cfg.Addr, cfg.Port),
	}

	// prefer explicit cfg.Scheme, otherwise infer from TLS
	if cfg.Scheme != "" {
		sc[0].Scheme = cfg.Scheme
	} else if cfg.EnableTLS {
		sc[0].Scheme = "https"
	}

	return sc
}

func NewNacosConfigSource(cfg Conf, cc config_client.IConfigClient) (config.Source, error) {

	if cfg.Addr == "" || cfg.Port == 0 {
		return nil, nil
	}
	source := NewConfigSource(cc, WithGroup(cfg.GroupId), WithDataID(cfg.DataId))
	return source, nil
}

// NewRegistryEngine
//
//	@Description: 如果 cfg.Addr == "" || cfg.Port == 0 则返回 nil, 需要外部兼容
//	@param cfg
//	@param nc
//	@return *Registry
//	@return error
func NewRegistryEngine(cfg Conf, nc naming_client.INamingClient) (*Registry, error) {

	if cfg.Addr == "" || cfg.Port == 0 {
		return nil, nil
	}

	return New(nc,
		WithPrefix("/"+cfg.ClusterName),                             // key prefix
		WithWeight(float64(friendly.GetOrDefault(cfg.Weight, 100))), // default weight
		WithCluster(cfg.ClusterName),                                // cluster name
		WithGroup(cfg.GroupId),                                      // group
	), nil
}

// NewRegistryEngineSimple
//
//	@Description: 手动调用简化创建流程
//	@param cfg
//	@param nc
//	@return *Registry
//	@return error
func NewRegistryEngineSimple(cfg Conf) (*Registry, error) {

	if cfg.Addr == "" || cfg.Port == 0 {
		return nil, nil
	}

	nc, err := NewNamingClient(cfg)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return New(nc,
		WithPrefix("/"+cfg.ClusterName),                             // key prefix
		WithWeight(float64(friendly.GetOrDefault(cfg.Weight, 100))), // default weight
		WithCluster(cfg.ClusterName),                                // cluster name
		WithGroup(cfg.GroupId),                                      // group
	), nil
}

// NewNamingClient
//
//	@Description: 如果 cfg.Addr == "" || cfg.Port == 0 则返回 nil, 需要外部兼容
//	@param cfg
//	@return naming_client.INamingClient
//	@return error
func NewNamingClient(cfg Conf) (naming_client.INamingClient, error) {

	if cfg.Addr == "" || cfg.Port == 0 {
		return nil, nil
	}

	cc, sc := newNacosClientConfig(cfg), newNacosServerConfig(cfg)
	namingClient, err := clients.NewNamingClient(vo.NacosClientParam{
		ClientConfig:  cc,
		ServerConfigs: sc,
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return namingClient, nil
}

// NewConfigClient
//
//	@Description: 如果 cfg.Addr == "" || cfg.Port == 0 则返回 nil, 需要外部兼容
//	@param cfg
//	@return iClient
//	@return err
func NewConfigClient(cfg Conf) (iClient config_client.IConfigClient, err error) {
	if cfg.Addr == "" || cfg.Port == 0 {
		return nil, nil
	}

	cc, sc := newNacosClientConfig(cfg), newNacosServerConfig(cfg)
	return clients.NewConfigClient(
		vo.NacosClientParam{
			ClientConfig:  cc,
			ServerConfigs: sc,
		},
	)
}
