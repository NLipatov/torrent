package torrent

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/anacrolix/torrent/bencode"
	qt "github.com/go-quicktest/qt"

	"github.com/anacrolix/torrent/internal/testutil"
	"github.com/anacrolix/torrent/metainfo"
)

func TestReaderReadContext(t *testing.T) {
	cl, err := NewClient(TestingConfig(t))
	qt.Assert(t, qt.IsNil(err))
	defer cl.Close()
	tt, err := cl.AddTorrent(testutil.GreetingMetaInfo())
	qt.Assert(t, qt.IsNil(err))
	defer tt.Drop()
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Millisecond))
	defer cancel()
	r := tt.Files()[0].NewReader()
	defer r.Close()
	_, err = r.ReadContext(ctx, make([]byte, 1))
	qt.Assert(t, qt.Equals(err, context.DeadlineExceeded))
}

func TestReaderSetContextAndRead(t *testing.T) {
	cl, err := NewClient(TestingConfig(t))
	qt.Assert(t, qt.IsNil(err))
	defer cl.Close()
	tt, err := cl.AddTorrent(testutil.GreetingMetaInfo())
	qt.Assert(t, qt.IsNil(err))
	defer tt.Drop()
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Millisecond))
	defer cancel()
	r := tt.Files()[0].NewReader()
	defer r.Close()
	r.SetContext(ctx)
	_, err = r.Read(make([]byte, 1))
	qt.Assert(t, qt.Equals(err, context.DeadlineExceeded))
}

func TestCappedStorageReadFailuresReturn(t *testing.T) {
	cfg := TestingConfig(t)
	cfg.DefaultStorage = badStorage{}
	cl, err := NewClient(cfg)
	qt.Assert(t, qt.IsNil(err))
	defer cl.Close()

	infoBytes, err := bencode.Marshal(metainfo.Info{
		PieceLength: 100,
		Pieces:      make([]byte, 20),
		Files:       []metainfo.FileInfo{{Path: []string{"greeting"}, Length: 100}},
	})
	qt.Assert(t, qt.IsNil(err))
	tt, added := cl.AddTorrentOpt(AddTorrentOpts{
		InfoBytes: infoBytes,
		InfoHash:  metainfo.HashBytes(infoBytes),
	})
	qt.Assert(t, qt.IsTrue(added))
	defer tt.Drop()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r := tt.NewReader()
	_, err = r.Seek(50, io.SeekStart)
	qt.Assert(t, qt.IsNil(err))
	r.SetContext(ctx)
	_, err = r.Read(make([]byte, 1))
	qt.Assert(t, qt.ErrorIs(err, context.Canceled))
	qt.Assert(t, qt.IsNil(r.Close()))

	r = tt.NewReader()
	defer r.Close()
	_, err = r.Seek(50, io.SeekStart)
	qt.Assert(t, qt.IsNil(err))
	_, err = r.Read(make([]byte, 1))
	qt.Assert(t, qt.ErrorIs(err, io.ErrUnexpectedEOF))
}
