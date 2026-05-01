package doltprobe

import "testing"

func TestSelectAuthoritativeDatabase(t *testing.T) {
	tests := []struct {
		name              string
		expectedProjectID string
		candidates        []Candidate
		want              string
	}{
		{
			name:              "returns unique project match",
			expectedProjectID: "project-a",
			candidates: []Candidate{
				{Name: "compat", HasIssuesTable: true, ProjectID: "project-b"},
				{Name: "authoritative", HasIssuesTable: true, ProjectID: "project-a"},
			},
			want: "authoritative",
		},
		{
			name:              "rejects ambiguous matches",
			expectedProjectID: "project-a",
			candidates: []Candidate{
				{Name: "compat", HasIssuesTable: true, ProjectID: "project-a"},
				{Name: "orphan", HasIssuesTable: true, ProjectID: "project-a"},
			},
			want: "",
		},
		{
			name:              "rejects candidates without issues table",
			expectedProjectID: "project-a",
			candidates:        []Candidate{{Name: "empty", HasIssuesTable: false, ProjectID: "project-a"}},
			want:              "",
		},
		{
			name:              "rejects missing expected identity",
			expectedProjectID: "",
			candidates:        []Candidate{{Name: "legacy", HasIssuesTable: true, ProjectID: "project-a"}},
			want:              "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SelectAuthoritativeDatabase(tt.expectedProjectID, tt.candidates); got != tt.want {
				t.Fatalf("SelectAuthoritativeDatabase() = %q, want %q", got, tt.want)
			}
		})
	}
}
