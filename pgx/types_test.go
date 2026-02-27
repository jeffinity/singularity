package pgx

import "testing"

type profile struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func TestJSONBValue(t *testing.T) {
	t.Run("nil value", func(t *testing.T) {
		j := JSONB[profile]{Val: nil}
		v, err := j.Value()
		if err != nil {
			t.Fatalf("Value failed: %v", err)
		}
		if v != nil {
			t.Fatalf("expected nil, got %#v", v)
		}
	})

	t.Run("marshal value", func(t *testing.T) {
		j := JSONB[profile]{Val: &profile{Name: "alice", Age: 18}}
		v, err := j.Value()
		if err != nil {
			t.Fatalf("Value failed: %v", err)
		}
		bs, ok := v.([]byte)
		if !ok {
			t.Fatalf("expected []byte, got %T", v)
		}
		got := string(bs)
		if got != `{"name":"alice","age":18}` {
			t.Fatalf("unexpected json: %s", got)
		}
	})
}

func TestJSONBScan(t *testing.T) {
	t.Run("scan nil", func(t *testing.T) {
		var j JSONB[profile]
		if err := j.Scan(nil); err != nil {
			t.Fatalf("Scan failed: %v", err)
		}
		if j.Val != nil {
			t.Fatalf("expected nil Val, got %#v", j.Val)
		}
	})

	t.Run("scan bytes", func(t *testing.T) {
		var j JSONB[profile]
		if err := j.Scan([]byte(`{"name":"bob","age":20}`)); err != nil {
			t.Fatalf("Scan failed: %v", err)
		}
		if j.Val == nil || j.Val.Name != "bob" || j.Val.Age != 20 {
			t.Fatalf("unexpected value: %#v", j.Val)
		}
	})

	t.Run("scan string", func(t *testing.T) {
		var j JSONB[profile]
		if err := j.Scan(`{"name":"carol","age":21}`); err != nil {
			t.Fatalf("Scan failed: %v", err)
		}
		if j.Val == nil || j.Val.Name != "carol" || j.Val.Age != 21 {
			t.Fatalf("unexpected value: %#v", j.Val)
		}
	})

	t.Run("unsupported type", func(t *testing.T) {
		var j JSONB[profile]
		if err := j.Scan(123); err == nil {
			t.Fatal("expected error for unsupported type")
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		var j JSONB[profile]
		if err := j.Scan([]byte(`{"name"`)); err == nil {
			t.Fatal("expected unmarshal error")
		}
	})
}
