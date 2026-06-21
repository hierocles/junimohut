package modoverwrites

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRecordMergeAndContains(t *testing.T) {
	must := require.New(t)

	path := t.TempDir() + "/overwrite-merges.json"
	svc, err := NewService(path)
	must.NoError(err)

	must.False(svc.ContainsOverwrites("Fashion Sense::PeacefulEnd.FashionSense"))
	must.NoError(svc.RecordMerge("Fashion Sense::PeacefulEnd.FashionSense"))
	must.True(svc.ContainsOverwrites("Fashion Sense::PeacefulEnd.FashionSense"))

	svc2, err := NewService(path)
	must.NoError(err)
	must.True(svc2.ContainsOverwrites("Fashion Sense::PeacefulEnd.FashionSense"))

	must.NoError(svc2.Delete("Fashion Sense::PeacefulEnd.FashionSense"))
	must.False(svc2.ContainsOverwrites("Fashion Sense::PeacefulEnd.FashionSense"))
}
