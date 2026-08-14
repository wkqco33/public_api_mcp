package tools

import (
	"testing"
	"time"
)

func TestKmaBaseTimeNcstAtBoundary(t *testing.T) {
	kst := time.FixedZone("KST", 9*60*60)
	date, base := kmaBaseTimeNcstAt(time.Date(2024, 3, 7, 10, 9, 0, 0, kst))
	if date != "20240307" || base != "0900" {
		t.Fatalf("got %s %s", date, base)
	}
	date, base = kmaBaseTimeNcstAt(time.Date(2024, 3, 7, 10, 10, 0, 0, kst))
	if date != "20240307" || base != "1000" {
		t.Fatalf("got %s %s", date, base)
	}
}

func TestKmaBaseTimeVilageAtMidnightBoundary(t *testing.T) {
	kst := time.FixedZone("KST", 9*60*60)
	date, base := kmaBaseTimeVilageAt(time.Date(2024, 3, 7, 1, 5, 0, 0, kst))
	if date != "20240306" || base != "2300" {
		t.Fatalf("got %s %s", date, base)
	}
	date, base = kmaBaseTimeVilageAt(time.Date(2024, 3, 7, 11, 10, 0, 0, kst))
	if date != "20240307" || base != "1100" {
		t.Fatalf("got %s %s", date, base)
	}
}
