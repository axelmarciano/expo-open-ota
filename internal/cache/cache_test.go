package cache

import (
	"reflect"
	"testing"
)

func TestParseSentinelAddrsTrimsWhitespaceAndDropsEmptyEntries(t *testing.T) {
	got := parseSentinelAddrs(" sentinel-0:26379, sentinel-1:26379,,\t sentinel-2:26379 , ")
	want := []string{"sentinel-0:26379", "sentinel-1:26379", "sentinel-2:26379"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseSentinelAddrs() = %#v, want %#v", got, want)
	}
}

func TestParseSentinelAddrsReturnsEmptyForBlankInput(t *testing.T) {
	got := parseSentinelAddrs(" , \t,")

	if len(got) != 0 {
		t.Fatalf("parseSentinelAddrs() = %#v, want empty", got)
	}
}
