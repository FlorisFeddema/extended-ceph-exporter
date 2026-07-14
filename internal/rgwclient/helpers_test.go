package rgwclient

import (
	"testing"

	cephadmin "github.com/ceph/go-ceph/rgw/admin"
)

func TestNormalizeStore(t *testing.T) {
	tests := []struct {
		name     string
		backends []cephadmin.StorageBackend
		expected string
	}{
		{name: "none", expected: unknownLabelValue},
		{name: "single", backends: []cephadmin.StorageBackend{{Name: "beast"}}, expected: "beast"},
		{name: "empty names", backends: []cephadmin.StorageBackend{{Name: ""}}, expected: unknownLabelValue},
		{name: "mixed", backends: []cephadmin.StorageBackend{{Name: "beast"}, {Name: "civetweb"}}, expected: mixedLabelValue},
	}

	for _, tt := range tests {
		if got := normalizeStore(tt.backends); got != tt.expected {
			t.Fatalf("%s: got %q want %q", tt.name, got, tt.expected)
		}
	}
}

func TestZonegroupRealm(t *testing.T) {
	if got := zonegroupRealm(""); got != unknownLabelValue {
		t.Fatalf("empty zonegroup = %q", got)
	}
	if got := zonegroupRealm("realm-a"); got != "realm-a" {
		t.Fatalf("unexpected zonegroup mapping: %q", got)
	}
}

func TestUserRealm(t *testing.T) {
	if got := userRealm(nil); got != unknownLabelValue {
		t.Fatalf("nil buckets = %q", got)
	}
	if got := userRealm([]cephadmin.Bucket{{Zonegroup: "a"}, {Zonegroup: "a"}}); got != "a" {
		t.Fatalf("same zonegroup = %q", got)
	}
	if got := userRealm([]cephadmin.Bucket{{Zonegroup: "a"}, {Zonegroup: "b"}}); got != mixedLabelValue {
		t.Fatalf("mixed zonegroup = %q", got)
	}
}

func TestPointerConversions(t *testing.T) {
	u64 := uint64(4)
	i64 := int64(8)
	i := 2
	bTrue := true
	bFalse := false

	if uint64PtrFloat(nil) != 0 || uint64PtrFloat(&u64) != 4 {
		t.Fatal("unexpected uint64PtrFloat")
	}
	if int64PtrFloat(nil) != 0 || int64PtrFloat(&i64) != 8 {
		t.Fatal("unexpected int64PtrFloat")
	}
	if intPtrFloat(nil) != 0 || intPtrFloat(&i) != 2 {
		t.Fatal("unexpected intPtrFloat")
	}
	if intPtrBool(nil) || !intPtrBool(&i) {
		t.Fatal("unexpected intPtrBool")
	}
	if boolPtrValue(nil) || !boolPtrValue(&bTrue) || boolPtrValue(&bFalse) {
		t.Fatal("unexpected boolPtrValue")
	}
}

func TestQuotaLimitHelpers(t *testing.T) {
	enabled := true
	disabled := false
	i64 := int64(10)
	i := 2
	objects := int64(9)

	if got := quotaSizeLimit(&enabled, &i64, &i); got == nil || *got != 10 {
		t.Fatalf("preferred max size = %v", got)
	}
	if got := quotaSizeLimit(&enabled, nil, &i); got == nil || *got != 2048 {
		t.Fatalf("max size kb conversion = %v", got)
	}
	if got := quotaSizeLimit(&enabled, nil, nil); got != nil {
		t.Fatalf("nil quota size = %v", got)
	}
	if got := quotaSizeLimit(&disabled, &i64, &i); got != nil {
		t.Fatalf("disabled quota size = %v", got)
	}
	if got := quotaObjectsLimit(&enabled, &objects); got == nil || *got != 9 {
		t.Fatalf("quota objects = %v", got)
	}
	if got := quotaObjectsLimit(&enabled, nil); got != nil {
		t.Fatalf("nil quota objects = %v", got)
	}
}
