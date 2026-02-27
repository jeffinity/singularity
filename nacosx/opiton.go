package nacosx

type options struct {
	prefix  string
	weight  float64
	cluster string
	group   string
	kind    string
	dataID  string
}

// Option is a nacos registry option.
type Option func(o *options)

// WithPrefix sets the key prefix.
func WithPrefix(prefix string) Option {
	return func(o *options) { o.prefix = prefix }
}

// WithDataID With nacos config data id.
func WithDataID(dataID string) Option {
	return func(o *options) {
		o.dataID = dataID
	}
}

// WithWeight sets the default instance weight.
func WithWeight(weight float64) Option {
	return func(o *options) { o.weight = weight }
}

// WithCluster sets the cluster name.
func WithCluster(cluster string) Option {
	return func(o *options) { o.cluster = cluster }
}

// WithGroup sets the group name.
func WithGroup(group string) Option {
	return func(o *options) { o.group = group }
}

// WithDefaultKind sets the default protocol kind.
func WithDefaultKind(kind string) Option {
	return func(o *options) { o.kind = kind }
}
