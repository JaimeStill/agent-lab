package internal_images_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/JaimeStill/agent-lab/internal/images"
)

func TestParsePageRange(t *testing.T) {
	tests := []struct {
		name    string
		expr    string
		maxPage int
		want    []int
		wantErr error
	}{
		{
			"single page",
			"1",
			10,
			[]int{1},
			nil,
		},
		{
			"single page middle",
			"5",
			10,
			[]int{5},
			nil,
		},
		{
			"single page last",
			"10",
			10,
			[]int{10},
			nil,
		},
		{
			"simple range",
			"1-5",
			10,
			[]int{1, 2, 3, 4, 5},
			nil,
		},
		{
			"range middle",
			"3-7",
			10,
			[]int{3, 4, 5, 6, 7},
			nil,
		},
		{
			"range to end",
			"8-10",
			10,
			[]int{8, 9, 10},
			nil,
		},
		{
			"multiple single pages",
			"1,3,5",
			10,
			[]int{1, 3, 5},
			nil,
		},
		{
			"mixed pages and ranges",
			"1,3-5,8",
			10,
			[]int{1, 3, 4, 5, 8},
			nil,
		},
		{
			"complex expression",
			"1-3,5,7-9",
			10,
			[]int{1, 2, 3, 5, 7, 8, 9},
			nil,
		},
		{
			"with spaces",
			" 1 , 3 - 5 , 8 ",
			10,
			[]int{1, 3, 4, 5, 8},
			nil,
		},
		{
			"duplicate pages collapsed",
			"1,1,2,2,3",
			10,
			[]int{1, 2, 3},
			nil,
		},
		{
			"overlapping ranges collapsed",
			"1-5,3-7",
			10,
			[]int{1, 2, 3, 4, 5, 6, 7},
			nil,
		},
		{
			"open start range",
			"-3",
			10,
			[]int{1, 2, 3},
			nil,
		},
		{
			"open end range",
			"8-",
			10,
			[]int{8, 9, 10},
			nil,
		},
		{
			"empty string",
			"",
			10,
			nil,
			images.ErrInvalidPageRange,
		},
		{
			"page zero",
			"0",
			10,
			nil,
			images.ErrPageOutOfRange,
		},
		{
			"page exceeds max",
			"15",
			10,
			nil,
			images.ErrPageOutOfRange,
		},
		{
			"range exceeds max",
			"8-15",
			10,
			nil,
			images.ErrPageOutOfRange,
		},
		{
			"start greater than end",
			"5-3",
			10,
			nil,
			images.ErrInvalidPageRange,
		},
		{
			"invalid page format",
			"abc",
			10,
			nil,
			images.ErrInvalidPageRange,
		},
		{
			"invalid range format",
			"1-abc",
			10,
			nil,
			images.ErrInvalidPageRange,
		},
		{
			"negative page",
			"-5-3",
			10,
			nil,
			images.ErrInvalidPageRange,
		},
		{
			"only commas",
			",,,",
			10,
			nil,
			images.ErrInvalidPageRange,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := images.ParsePageRange(tt.expr, tt.maxPage)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("ParsePageRange() error = nil, want error")
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ParsePageRange() error = %v, want %v", err, tt.wantErr)
				}
			} else {
				if err != nil {
					t.Errorf("ParsePageRange() error = %v, want nil", err)
				}
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("ParsePageRange() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestParsePageRange_Sorting(t *testing.T) {
	got, err := images.ParsePageRange("9,1,5,3,7", 10)
	if err != nil {
		t.Fatalf("ParsePageRange() error = %v", err)
	}

	want := []int{1, 3, 5, 7, 9}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ParsePageRange() = %v, want sorted %v", got, want)
	}
}

func TestParsePageRange_FullDocument(t *testing.T) {
	got, err := images.ParsePageRange("1-9", 9)
	if err != nil {
		t.Fatalf("ParsePageRange() error = %v", err)
	}

	want := []int{1, 2, 3, 4, 5, 6, 7, 8, 9}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ParsePageRange() = %v, want %v", got, want)
	}
}

func TestParsePageRange_SinglePageDocument(t *testing.T) {
	got, err := images.ParsePageRange("1", 1)
	if err != nil {
		t.Fatalf("ParsePageRange() error = %v", err)
	}

	want := []int{1}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ParsePageRange() = %v, want %v", got, want)
	}
}
