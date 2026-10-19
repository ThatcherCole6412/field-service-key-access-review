package fieldaudit

import (
	"reflect"
	"testing"
	"time"
)

func TestBuildClassifiesFieldServiceReach(t *testing.T) {
	tests := []struct {
		name   string
		scopes []string
		want   Access
	}{
		{name: "photos only", scopes: []string{PhotosScope}, want: Access{Photos: true}},
		{name: "dispatcher", scopes: []string{DispatchScope, FollowUpScope}, want: Access{Dispatch: true, FollowUp: true}},
		{name: "unrelated", scopes: []string{"account.balance"}, want: Access{}},
		{name: "wildcard", scopes: []string{"*"}, want: Access{Photos: true, Dispatch: true, FollowUp: true, FullAccess: true}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			review := Build([]Key{{ID: "key-1", Name: "crew", Scopes: tt.scopes}}, time.Date(2026, 9, 21, 14, 0, 0, 0, time.FixedZone("local", 8*60*60)))
			if got := review.Findings[0].Access; !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("access = %#v, want %#v", got, tt.want)
			}
			if review.Date != "2026-09-21" {
				t.Fatalf("date = %q, want UTC date", review.Date)
			}
		})
	}
}

func TestBuildSortsKeysForStableReviews(t *testing.T) {
	review := Build([]Key{
		{ID: "key-2", Name: "west"},
		{ID: "key-1", Name: "east"},
	}, time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC))
	if got := review.Findings[0].Name; got != "east" {
		t.Fatalf("first key = %q, want east", got)
	}
}
