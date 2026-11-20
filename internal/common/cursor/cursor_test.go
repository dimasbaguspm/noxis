package cursor

import (
	"testing"

	"noxis/internal/common/ref"
)

func TestLimit(t *testing.T) {
	for _, tc := range []struct {
		name                 string
		n, fallback, ceiling int
		want                 int
	}{
		{"unset", 0, 50, 200, 50},
		{"negative", -5, 50, 200, 50},
		{"inside", 20, 50, 200, 20},
		{"over", 500, 50, 200, 200},
		{"at ceiling", 200, 50, 200, 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := Limit(tc.n, tc.fallback, tc.ceiling); got != tc.want {
				t.Fatalf("Limit(%d) = %d, want %d", tc.n, got, tc.want)
			}
		})
	}
}

func TestRef(t *testing.T) {
	good := ref.New()
	got, err := Ref(good)
	if err != nil || got != good {
		t.Fatalf("Ref(%q) = %q, %v", good, got, err)
	}
	if got, err := Ref(""); err != nil || got != "" {
		t.Fatalf("Ref(\"\") = %q, %v; want empty, nil", got, err)
	}
	if _, err := Ref("42"); err == nil {
		t.Fatal("Ref(\"42\") accepted an integer id")
	}
}

func TestNext(t *testing.T) {
	r := ref.New()
	if got := Next(r, true); got != r {
		t.Fatalf("Next(full) = %q, want %q", got, r)
	}
	if got := Next(r, false); got != "" {
		t.Fatalf("Next(short) = %q, want empty", got)
	}
	if got := Next("", true); got != "" {
		t.Fatalf("Next(no rows) = %q, want empty", got)
	}
}
