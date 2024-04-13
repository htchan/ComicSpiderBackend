package service

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func Test_generateFileVersionMap(t *testing.T) {

}

func Test_parseVersion(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		version    string
		expectTime time.Time
		expectErr  bool
	}{
		{
			name:       "parse yr + month + day",
			version:    "2020-02-03.csv",
			expectTime: time.Date(2020, time.February, 3, 0, 0, 0, 0, time.UTC),
			expectErr:  false,
		},
		{
			name:       "parse yr + month",
			version:    "2020-02.csv",
			expectTime: time.Date(2020, time.February, 1, 0, 0, 0, 0, time.UTC),
			expectErr:  false,
		},
		{
			name:       "parse yr",
			version:    "2020.csv",
			expectTime: time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC),
			expectErr:  false,
		},
		{
			name:       "parse unknown",
			version:    "all.csv",
			expectTime: time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC),
			expectErr:  true,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			timeResult, err := parseVersion(test.version)
			if (err != nil) != test.expectErr {
				t.Errorf("got err: %v, expect err: %v", err, test.expectErr)
			}

			if timeResult != test.expectTime {
				t.Errorf("got time: %v, want time: %v", timeResult, test.expectTime)
			}
		})
	}
}

func Test_shouldAggregate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		baseVersion string
		version     string
		expect      bool
	}{
		// same year, same month
		{
			name:        "true when versions has same year but not current year and same month month",
			baseVersion: "2000-02-02.csv",
			version:     "2000-02-03.csv",
			expect:      true,
		},
		{
			name:        "true when versions are current year and same month but not current",
			baseVersion: time.Date(time.Now().Year()-1, time.Now().Month()+1, 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02.csv"),
			version:     time.Date(time.Now().Year()-1, time.Now().Month()+1, 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02.csv"),
			expect:      true,
		},
		{
			name:        "false when versions are current year and current month",
			baseVersion: time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02.csv"),
			version:     time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02.csv"),
			expect:      false,
		},
		// same year, different month
		{
			name:        "true when versions has same year but not current year and different month",
			baseVersion: "2000-01-02.csv",
			version:     "2000-02-03.csv",
			expect:      true,
		},
		{
			name:        "false when versions has current year and different month",
			baseVersion: time.Date(time.Now().Year(), time.January, 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02.csv"),
			version:     time.Date(time.Now().Year(), time.February, 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02.csv"),
			expect:      false,
		},
		// different yr
		{
			name:        "false when versions has different yr and different month",
			baseVersion: "2001-01-02.csv",
			version:     "2002-02-03.csv",
			expect:      false,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			result := shouldAggragate(test.baseVersion, test.version)
			if result != test.expect {
				t.Errorf("got: %v, want: %v", result, test.expect)
			}
		})
	}
}

func Test_aggregatedVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		versions []string
		expect   string
	}{
		{
			name:     "same only one content",
			versions: []string{"2022-01-01.csv"},
			expect:   "2022-01-01.csv",
		},
		{
			name:     "same yr and same month",
			versions: []string{"2022-01-01.csv", "2022-01-02.csv"},
			expect:   "2022-01.csv",
		},
		{
			name:     "same yr and different month",
			versions: []string{"2022-02-01.csv", "2022-03-02.csv"},
			expect:   "2022.csv",
		},
		{
			name:     "different yr and different month",
			versions: []string{"2021-02-01.csv", "2022-03-02.csv"},
			expect:   "",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			result := aggregatedVersion(test.versions)
			if result != test.expect {
				t.Errorf("got: %v, want: %v", result, test.expect)
			}
		})
	}
}

func Test_existContent(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		baseContent []string
		line        string
		expect      bool
	}{
		{
			name:        "true if line exist in base content",
			baseContent: []string{"a", "b", "c"},
			line:        "a",
			expect:      true,
		},
		{
			name:        "false if line not exist in base content",
			baseContent: []string{"a", "b", "c"},
			line:        "d",
			expect:      false,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			result := existContent(test.baseContent, test.line)
			if result != test.expect {
				t.Errorf("got: %v, want: %v", result, test.expect)
			}
		})
	}
}

func Test_aggregateContent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		baseContent []string
		content     []string
		expect      []string
	}{
		{
			name:        "works",
			baseContent: []string{"a", "b", "c"},
			content:     []string{"a", "c", "d", "e"},
			expect:      []string{"a", "b", "c", "d", "e"},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			result := aggregateContent(test.baseContent, test.content)
			if !cmp.Equal(result, test.expect) {
				t.Errorf("got: %v, want: %v", result, test.expect)
			}
		})
	}
}
