package nexus

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveLatestMainFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		files   []ModFile
		wantID  int
		wantErr bool
	}{
		{
			name: "picks newest main",
			files: []ModFile{
				{FileID: 1, CategoryName: "MAIN", UploadedTimestamp: 100},
				{FileID: 2, CategoryName: "MAIN", UploadedTimestamp: 200},
			},
			wantID: 2,
		},
		{
			name: "ignores removed and archived",
			files: []ModFile{
				{FileID: 1, CategoryName: "REMOVED", UploadedTimestamp: 999},
				{FileID: 2, CategoryName: "ARCHIVED", UploadedTimestamp: 888},
				{FileID: 3, CategoryName: "OPTIONAL", UploadedTimestamp: 50},
			},
			wantID: 3,
		},
		{
			name: "empty category treated as main",
			files: []ModFile{
				{FileID: 10, CategoryName: "", UploadedTimestamp: 10},
				{FileID: 11, CategoryName: "MAIN", UploadedTimestamp: 11},
			},
			wantID: 11,
		},
		{
			name:    "no files",
			files:   nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			must := require.New(t)

			got, err := ResolveLatestMainFile(tt.files)
			if tt.wantErr {
				must.Error(err)
				return
			}
			must.NoError(err)
			must.Equal(tt.wantID, got.FileID)
		})
	}
}
