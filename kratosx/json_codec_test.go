package kratosx

import (
	"strings"
	"testing"

	"google.golang.org/protobuf/types/descriptorpb"
)

type jsonOnly struct {
	Name string `json:"name"`
}

func TestProtoMarshalAndUnmarshal(t *testing.T) {
	msg := &descriptorpb.FieldDescriptorProto{
		Type: descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum(),
	}

	t.Run("marshal enum as number", func(t *testing.T) {
		bs, err := ProtoMarshal(msg)
		if err != nil {
			t.Fatalf("ProtoMarshal failed: %v", err)
		}
		if !strings.Contains(string(bs), `"type":5`) {
			t.Fatalf("unexpected marshal result: %s", string(bs))
		}
	})

	t.Run("marshal enum as string", func(t *testing.T) {
		bs, err := ProtoMarshalEnumAsString(msg)
		if err != nil {
			t.Fatalf("ProtoMarshalEnumAsString failed: %v", err)
		}
		if !strings.Contains(string(bs), `"type":"TYPE_INT32"`) {
			t.Fatalf("unexpected marshal result: %s", string(bs))
		}
	})

	t.Run("unmarshal to proto pointer", func(t *testing.T) {
		var dst descriptorpb.FieldDescriptorProto
		if err := ProtoUnmarshal([]byte(`{"type":"TYPE_STRING"}`), &dst); err != nil {
			t.Fatalf("ProtoUnmarshal failed: %v", err)
		}
		if dst.GetType() != descriptorpb.FieldDescriptorProto_TYPE_STRING {
			t.Fatalf("unexpected enum value: %v", dst.GetType())
		}
	})

	t.Run("unmarshal to proto double pointer", func(t *testing.T) {
		var dst *descriptorpb.FieldDescriptorProto
		if err := ProtoUnmarshal([]byte(`{"type":"TYPE_BOOL"}`), &dst); err != nil {
			t.Fatalf("ProtoUnmarshal failed: %v", err)
		}
		if dst == nil || dst.GetType() != descriptorpb.FieldDescriptorProto_TYPE_BOOL {
			t.Fatalf("unexpected decoded value: %#v", dst)
		}
	})
}

func TestProtoUnmarshalRejectsNonProto(t *testing.T) {
	var dst map[string]any
	err := ProtoUnmarshal([]byte(`{"a":1}`), &dst)
	if err == nil {
		t.Fatal("expected error for non-proto target")
	}
}

func TestCodecFallbackJSON(t *testing.T) {
	in := jsonOnly{Name: "alice"}
	bs, err := Codec.Marshal(in)
	if err != nil {
		t.Fatalf("Codec.Marshal failed: %v", err)
	}

	var out jsonOnly
	if err := Codec.Unmarshal(bs, &out); err != nil {
		t.Fatalf("Codec.Unmarshal failed: %v", err)
	}
	if out.Name != "alice" {
		t.Fatalf("unexpected value: %#v", out)
	}
}

func TestCodecNames(t *testing.T) {
	if Codec.Name() != "ProtoAndJsonCodec" {
		t.Fatalf("unexpected Codec name: %s", Codec.Name())
	}
	if EnumStringCodec.Name() != "ProtoAndJsonEnumStringCodec" {
		t.Fatalf("unexpected EnumStringCodec name: %s", EnumStringCodec.Name())
	}
	if JSONCodec.Name() != "JSONCodec" {
		t.Fatalf("unexpected JSONCodec name: %s", JSONCodec.Name())
	}
}
