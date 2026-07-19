package torrent

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/anacrolix/torrent/bencode"
	"github.com/stretchr/testify/require"

	"github.com/anacrolix/torrent/internal/testutil"
	"github.com/anacrolix/torrent/metainfo"
)

func TestReaderReadContext(t *testing.T) {
	cl, err := NewClient(TestingConfig(t))
	require.NoError(t, err)
	defer cl.Close()
	tt, err := cl.AddTorrent(testutil.GreetingMetaInfo())
	require.NoError(t, err)
	defer tt.Drop()
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Millisecond))
	defer cancel()
	r := tt.Files()[0].NewReader()
	defer r.Close()
	_, err = r.ReadContext(ctx, make([]byte, 1))
	require.EqualValues(t, context.DeadlineExceeded, err)
}

func TestReaderSetContextAndRead(t *testing.T) {
	cl, err := NewClient(TestingConfig(t))
	require.NoError(t, err)
	defer cl.Close()
	tt, err := cl.AddTorrent(testutil.GreetingMetaInfo())
	require.NoError(t, err)
	defer tt.Drop()
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Millisecond))
	defer cancel()
	r := tt.Files()[0].NewReader()
	defer r.Close()
	r.SetContext(ctx)
	_, err = r.Read(make([]byte, 1))
	require.EqualValues(t, context.DeadlineExceeded, err)
}

func TestCappedStorageReadFailuresReturn(t *testing.T) {
	cfg := TestingConfig(t)
	cfg.DefaultStorage = badStorage{}
	cl, err := NewClient(cfg)
	require.NoError(t, err)
	defer cl.Close()

	infoBytes, err := bencode.Marshal(metainfo.Info{
		PieceLength: 100,
		Pieces:      make([]byte, 20),
		Files:       []metainfo.FileInfo{{Path: []string{"greeting"}, Length: 100}},
	})
	require.NoError(t, err)
	tt, added := cl.AddTorrentOpt(AddTorrentOpts{
		InfoBytes: infoBytes,
		InfoHash:  metainfo.HashBytes(infoBytes),
	})
	require.True(t, added)
	defer tt.Drop()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r := tt.NewReader()
	_, err = r.Seek(50, io.SeekStart)
	require.NoError(t, err)
	r.SetContext(ctx)
	_, err = r.Read(make([]byte, 1))
	require.ErrorIs(t, err, context.Canceled)
	require.NoError(t, r.Close())

	r = tt.NewReader()
	defer r.Close()
	_, err = r.Seek(50, io.SeekStart)
	require.NoError(t, err)
	ctx, cancel = context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	r.SetContext(ctx)
	_, err = r.Read(make([]byte, 1))
	require.ErrorIs(t, err, context.DeadlineExceeded)
}
