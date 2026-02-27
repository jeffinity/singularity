package kratosx

import (
	"reflect"

	"github.com/bytedance/sonic"
	perr "github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

var (
	// MarshalOptions is a configurable JSON format marshaller
	MarshalOptions = protojson.MarshalOptions{
		UseProtoNames:  true,
		UseEnumNumbers: true,
	}

	// MarshalOptionsEnumAsString 针对 proto 的枚举使用字符串（name）
	MarshalOptionsEnumAsString = protojson.MarshalOptions{
		UseProtoNames:  true,
		UseEnumNumbers: false,
	}

	// UnmarshalOptions is a configurable JSON format parser
	UnmarshalOptions = protojson.UnmarshalOptions{
		DiscardUnknown: true,
	}

	Codec = codec{} // 优先使用 proto json

	EnumStringCodec = enumStringCodec{} // proto enum 使用字符串

	JSONCodec = jsonCodec{} // 纯 json codec
)

// ProtoMarshal 使用默认的 MarshalOptions (枚举为数字) 编码 proto.Message
func ProtoMarshal(m proto.Message) ([]byte, error) {
	return MarshalOptions.Marshal(m)
}

// ProtoMarshalEnumAsString 使用 MarshalOptionsEnumAsString (枚举为字符串) 编码 proto.Message
func ProtoMarshalEnumAsString(m proto.Message) ([]byte, error) {
	return MarshalOptionsEnumAsString.Marshal(m)
}

// ProtoUnmarshal 使用 UnmarshalOptions 解析到目标 v，支持 *T / **T 形式的 proto.Message。
func ProtoUnmarshal(data []byte, v interface{}) error {
	handled, err := protoUnmarshalWithOptions(data, v)
	if !handled {
		return perr.Errorf("ProtoUnmarshal: target is not proto.Message, got %T", v)
	}
	return err
}

func protoMarshalIfProto(v interface{}, opts protojson.MarshalOptions) ([]byte, bool, error) {
	if m, ok := v.(proto.Message); ok {
		bs, err := opts.Marshal(m)
		return bs, true, err
	}
	return nil, false, nil
}

func protoUnmarshalWithOptions(data []byte, v interface{}) (bool, error) {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return false, perr.Errorf("codec.Unmarshal: expect non-nil pointer, got %T", v)
	}

	if pm, ok := v.(proto.Message); ok {
		return true, UnmarshalOptions.Unmarshal(data, pm)
	}

	// 处理 **T 等多级指针情况
	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			rv.Set(reflect.New(rv.Type().Elem()))
		}
		rv = rv.Elem()
	}

	// rv 现在是底层的 T，尝试取地址后判断是否是 proto.Message
	if !rv.CanAddr() {
		return false, perr.Errorf("codec.Unmarshal: value %T is not addressable", v)
	}
	if addr := rv.Addr().Interface(); addr != nil {
		if pm, ok := addr.(proto.Message); ok {
			return true, UnmarshalOptions.Unmarshal(data, pm)
		}
	}

	// 不是 proto.Message
	return false, nil
}

// codec is a Codec implementation with json.
//   - proto.Message => protojson + 数字枚举
//   - 其他 => sonic
type codec struct{}

func (codec) Marshal(v interface{}) ([]byte, error) {
	if bs, handled, err := protoMarshalIfProto(v, MarshalOptions); handled {
		return bs, err
	}
	return sonic.ConfigFastest.Marshal(v)
}

func (c codec) Unmarshal(data []byte, v interface{}) error {
	// 先尝试当作 proto 解析（支持 *T / **T）
	if handled, err := protoUnmarshalWithOptions(data, v); handled {
		return err
	}
	// fallback sonic
	return sonic.ConfigFastest.Unmarshal(data, v)
}

func (codec) Name() string {
	return "ProtoAndJsonCodec"
}

func (c codec) MustMarshal(v any) string {
	bs, _ := c.Marshal(v)
	return string(bs)
}

func (c codec) MustMarshalB(v any) []byte {
	bs, _ := c.Marshal(v)
	return bs
}

type enumStringCodec struct{}

func (enumStringCodec) Marshal(v interface{}) ([]byte, error) {
	if bs, handled, err := protoMarshalIfProto(v, MarshalOptionsEnumAsString); handled {
		return bs, err
	}
	return sonic.ConfigFastest.Marshal(v)
}

func (c enumStringCodec) Unmarshal(data []byte, v interface{}) error {
	// Unmarshal 行为与默认 codec 相同，protojson 对数字/字符串枚举都能兼容解析
	if handled, err := protoUnmarshalWithOptions(data, v); handled {
		return err
	}
	return sonic.ConfigFastest.Unmarshal(data, v)
}

func (enumStringCodec) Name() string {
	return "ProtoAndJsonEnumStringCodec"
}

func (c enumStringCodec) MustMarshal(v any) string {
	bs, _ := c.Marshal(v)
	return string(bs)
}

func (c enumStringCodec) MustMarshalB(v any) []byte {
	bs, _ := c.Marshal(v)
	return bs
}

type jsonCodec struct{}

func (jsonCodec) Marshal(v interface{}) ([]byte, error) {
	return sonic.ConfigFastest.Marshal(v)
}

func (jsonCodec) Unmarshal(data []byte, v interface{}) error {
	return sonic.ConfigFastest.Unmarshal(data, v)
}

func (jsonCodec) Name() string {
	return "JSONCodec"
}

func (c jsonCodec) MustMarshal(v any) string {
	bs, _ := c.Marshal(v)
	return string(bs)
}

func (c jsonCodec) MustMarshalB(v any) []byte {
	bs, _ := c.Marshal(v)
	return bs
}
