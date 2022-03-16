package cache

import (
	"testing"
)

func TestBadgerCache_Has(t *testing.T) {
	err := testBadgerCache.Forget("foo")
	if err != nil {
		t.Error(err)
	}

	inCache, err := testBadgerCache.Has("foo")
	if err != nil {
		t.Error(err)
	}

	if inCache {
		t.Error("foo found in cache, and it should not be there")
	}

	_ = testBadgerCache.Set("foo", "bar")
	inCache, err = testBadgerCache.Has("foo")
	if err != nil {
		t.Error(err)
	}

	if !inCache {
		t.Error("foo not found in cache, and it should be there")
	}

	testBadgerCache.Forget("foo")
}

func TestBadgerCache_Get(t *testing.T) {
	err := testBadgerCache.Set("foo", "bar")
	if err != nil {
		t.Error(err)
	}

	x, err := testBadgerCache.Get("foo")
	if err != nil {
		t.Error(err)
	}

	if x != "bar" {
		t.Error("did not get correct value from cache")
	}

	testBadgerCache.Forget("foo")
}

func TestBadgerCache_Set(t *testing.T) {
	type args struct {
		str     string
		value   interface{}
		expires []int
	}
	tests := []struct {
		name    string
		c       *BadgerCache
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.c.Set(tt.args.str, tt.args.value, tt.args.expires...); (err != nil) != tt.wantErr {
				t.Errorf("BadgerCache.Set() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBadgerCache_Forget(t *testing.T) {
	err := testBadgerCache.Set("foo", "foo")
	if err != nil {
		t.Error(err)
	}

	err = testBadgerCache.Forget("foo")
	if err != nil {
		t.Error(err)
	}

	inCache, err := testBadgerCache.Has("foo")
	if err != nil {
		t.Error(err)
	}

	if inCache {
		t.Error("foo found in cache, and it should not be there")
	}
}

func TestBadgerCache_EmptyByMatch(t *testing.T) {
	type args struct {
		str string
	}
	tests := []struct {
		name    string
		c       *BadgerCache
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.c.EmptyByMatch(tt.args.str); (err != nil) != tt.wantErr {
				t.Errorf("BadgerCache.EmptyByMatch() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBadgerCache_Empty(t *testing.T) {
	err := testBadgerCache.Set("alpha", "beta")
	if err != nil {
		t.Error(err)
	}

	err = testBadgerCache.Empty()
	if err != nil {
		t.Error(err)
	}

	inCache, err := testBadgerCache.Has("alpha")
	if err != nil {
		t.Error(err)
	}

	if inCache {
		t.Error("alpha found in cache, and it should not be there")
	}
}

func TestBadgerCache_emptyByMatch(t *testing.T) {
	err := testBadgerCache.Set("alpha", "alpha")
	if err != nil {
		t.Error(err)
	}

	err = testBadgerCache.Set("alpha2", "alpha2")
	if err != nil {
		t.Error(err)
	}

	err = testBadgerCache.Set("beta", "beta")
	if err != nil {
		t.Error(err)
	}

	err = testBadgerCache.emptyByMatch("a")
	if err != nil {
		t.Error(err)
	}

	inCache, err := testBadgerCache.Has("alpha")
	if err != nil {
		t.Error(err)
	}

	if inCache {
		t.Error("alpha found in cache, and it should not be there")
	}

	inCache, err = testBadgerCache.Has("alpha2")
	if err != nil {
		t.Error(err)
	}

	if inCache {
		t.Error("alpha2 found in cache, and it should not be there")
	}

	inCache, err = testBadgerCache.Has("beta")
	if err != nil {
		t.Error(err)
	}

	if !inCache {
		t.Error("beta not found in cache, and it should be there")
	}
}
