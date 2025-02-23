package capnp

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"testing"
	"testing/quick"

	"capnproto.org/go/capnp/v3/exp/bufferpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		arena       Arena
		extraSegs   int
		err         error
		wantSeg0Len int
	}{{
		name:  "empty single segment",
		arena: SingleSegment(nil),
	}, {
		name:  "empty multi segment",
		arena: MultiSegment(nil),
	}, {
		name:  "short read only arena",
		arena: readOnlyArena{SingleSegment(make([]byte, 0, 7))},
	}, {
		name:  "short read only arena with cap",
		arena: readOnlyArena{SingleSegment(make([]byte, 0, 8))},
	}, {
		name:        "multi segment w/ root word",
		arena:       MultiSegment([][]byte{make([]byte, 8)}),
		wantSeg0Len: 8,
	}, {
		name:        "multi segment w/ data",
		arena:       MultiSegment([][]byte{incrementingData(8)}),
		wantSeg0Len: 8,
	}, {
		name:        "multi segment w/ 2 segments",
		arena:       MultiSegment([][]byte{make([]byte, 0, 16), make([]byte, 0)}),
		extraSegs:   1,
		wantSeg0Len: 0,
	}}
	for i := range tests {
		test := tests[i]
		t.Run(test.name, func(t *testing.T) {
			msg, seg, err := NewMessage(test.arena)
			require.ErrorIs(t, err, test.err)
			if test.err != nil {
				return
			}

			require.NoError(t, err)
			require.Equal(t, int64(1+test.extraSegs), msg.NumSegments())
			require.Equal(t, SegmentID(0), seg.ID())
			require.Len(t, seg.Data(), test.wantSeg0Len)
		})
	}
}

func TestAlloc(t *testing.T) {
	t.Parallel()

	type allocTest struct {
		name string

		seg  *Segment
		size Size

		allocID SegmentID
		addr    address
	}
	var tests []allocTest

	{
		_, seg := NewSingleSegmentMessage(nil)
		tests = append(tests, allocTest{
			name:    "empty alloc in empty segment",
			seg:     seg,
			size:    0,
			allocID: 0,
			addr:    8, // First alloc is after root pointer.
		})
	}
	{
		_, seg := NewMultiSegmentMessage([][]byte{
			incrementingData(24)[:8:8],
			incrementingData(24)[:8],
			incrementingData(24)[:8],
		})
		tests = append(tests, allocTest{
			name:    "prefers given segment",
			seg:     seg,
			size:    16,
			allocID: 1,
			addr:    8,
		})
	}
	{
		_, seg := NewMultiSegmentMessage([][]byte{
			incrementingData(24)[:8],
			incrementingData(24),
		})
		tests = append(tests, allocTest{
			name:    "given segment full with another available",
			seg:     seg,
			size:    16,
			allocID: 0,
			addr:    8,
		})
	}
	{
		msg, seg := NewMultiSegmentMessage([][]byte{
			incrementingData(24),
			incrementingData(24),
		})

		// Make arena not read-only again.
		msg.Arena.(*MultiSegmentArena).bp = &bufferpool.Default
		tests = append(tests, allocTest{
			name:    "given segment full and no others available",
			seg:     seg,
			size:    16,
			allocID: 2,
			addr:    0,
		})
	}

	for i, test := range tests {
		seg, addr, err := alloc(test.seg, test.size)
		if err != nil {
			t.Errorf("tests[%d] - %s: alloc(..., %d) error: %v", i, test.name, test.size, err)
			continue
		}
		if seg.ID() != test.allocID {
			t.Errorf("tests[%d] - %s: alloc(..., %d) returned segment %d; want segment %d", i, test.name, test.size, seg.ID(), test.allocID)
		}
		if addr != test.addr {
			t.Errorf("tests[%d] - %s: alloc(..., %d) returned address %v; want address %v", i, test.name, test.size, addr, test.addr)
		}
		if !seg.regionInBounds(addr, test.size) {
			t.Errorf("tests[%d] - %s: alloc(..., %d) returned address %v, which is not in bounds (len(seg.data) == %d)", i, test.name, test.size, addr, len(seg.Data()))
		} else if data := seg.slice(addr, test.size); !isZeroFilled(data) {
			t.Errorf("tests[%d] - %s: alloc(..., %d) region has data % 02x; want zero-filled", i, test.name, test.size, data)
		}
	}
}

type serializeTest struct {
	name            string
	segs            [][]byte
	out             []byte
	encodeFails     bool
	decodeFails     bool
	decodeError     error
	newMessageFails bool
}

func (st *serializeTest) arena() Arena {
	bb := make([][]byte, len(st.segs))
	for i := range bb {
		bb[i] = make([]byte, len(st.segs[i]))
		copy(bb[i], st.segs[i])
	}
	return MultiSegment(bb)
}

func (st *serializeTest) copyOut() []byte {
	out := make([]byte, len(st.out))
	copy(out, st.out)
	return out
}

var serializeTests = []serializeTest{
	{
		name:            "empty message",
		segs:            [][]byte{},
		encodeFails:     true,
		newMessageFails: true,
	},
	{
		name:        "empty stream",
		out:         []byte{},
		decodeFails: true,
		decodeError: io.EOF,
	},
	{
		name:        "incomplete segment count",
		out:         []byte{0x01},
		decodeFails: true,
	},
	{
		name: "incomplete segment size",
		out: []byte{
			0x00, 0x00, 0x00, 0x00,
			0x00,
		},
		decodeFails: true,
	},
	{
		name: "empty single segment",
		segs: [][]byte{
			{},
		},
		out: []byte{
			0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00,
		},
	},
	{
		name: "missing segment data",
		out: []byte{
			0x00, 0x00, 0x00, 0x00,
			0x01, 0x00, 0x00, 0x00,
		},
		decodeFails: true,
	},
	{
		name: "missing segment size",
		out: []byte{
			0x01, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00,
		},
		decodeFails: true,
	},
	{
		name: "missing segment size padding",
		out: []byte{
			0x01, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00,
		},
		decodeFails: true,
	},
	{
		name: "single segment",
		segs: [][]byte{
			incrementingData(8),
		},
		out: []byte{
			0x00, 0x00, 0x00, 0x00,
			0x01, 0x00, 0x00, 0x00,
			0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
		},
	},
	{
		name: "two segments",
		segs: [][]byte{
			incrementingData(8),
			incrementingData(8),
		},
		out: []byte{
			0x01, 0x00, 0x00, 0x00,
			0x01, 0x00, 0x00, 0x00,
			0x01, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00,
			0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
			0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
		},
	},
	{
		name: "two segments, missing size padding",
		out: []byte{
			0x01, 0x00, 0x00, 0x00,
			0x01, 0x00, 0x00, 0x00,
			0x01, 0x00, 0x00, 0x00,
			0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
			0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
		},
		decodeFails: true,
	},
	{
		name:        "HTTP traffic should not panic on GOARCH=386",
		out:         []byte("GET / HTTP/1.1\r\n\r\n"),
		decodeFails: true,
	},
	{
		name:        "max segment should not panic",
		out:         bytes.Repeat([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}, 16),
		decodeFails: true,
	},
}

func TestMarshal(t *testing.T) {
	t.Parallel()

	for i, test := range serializeTests {
		if test.decodeFails {
			continue
		}
		msg, _, err := NewMessage(test.arena())
		if err != nil != test.newMessageFails {
			t.Errorf("serializeTests[%d] %s: NewMessage unexpected error: %v", i, test.name, err)
			continue
		}
		if err != nil {
			continue
		}
		out, err := msg.Marshal()
		if err != nil {
			if !test.encodeFails {
				t.Errorf("serializeTests[%d] %s: Marshal error: %v", i, test.name, err)
			}
			continue
		}
		if test.encodeFails {
			t.Errorf("serializeTests[%d] - %s: Marshal success; want error", i, test.name)
			continue
		}
		if !bytes.Equal(out, test.out) {
			t.Errorf("serializeTests[%d] - %s: Marshal = % 02x; want % 02x", i, test.name, out, test.out)
		}
	}
}

func TestUnmarshal(t *testing.T) {
	t.Parallel()

	for i, test := range serializeTests {
		if test.encodeFails {
			continue
		}
		msg, err := Unmarshal(test.copyOut())
		if err != nil {
			if !test.decodeFails {
				t.Errorf("serializeTests[%d] - %s: Unmarshal error: %v", i, test.name, err)
			}
			if test.decodeError != nil && err != test.decodeError {
				t.Errorf("serializeTests[%d] - %s: Unmarshal error: %v; want %v", i, test.name, err, test.decodeError)
			}
			continue
		}
		if test.decodeFails {
			t.Errorf("serializeTests[%d] - %s: Unmarshal success; want error", i, test.name)
			continue
		}
		if msg.NumSegments() != int64(len(test.segs)) {
			t.Errorf("serializeTests[%d] - %s: Unmarshal NumSegments() = %d; want %d", i, test.name, msg.NumSegments(), len(test.segs))
			continue
		}
		for j := range test.segs {
			seg, err := msg.Segment(SegmentID(j))
			if err != nil {
				t.Errorf("serializeTests[%d] - %s: Unmarshal Segment(%d) error: %v", i, test.name, j, err)
				continue
			}
			if !bytes.Equal(seg.Data(), test.segs[j]) {
				t.Errorf("serializeTests[%d] - %s: Unmarshal Segment(%d) = % 02x; want % 02x", i, test.name, j, seg.Data(), test.segs[j])
			}
		}
	}
}

func TestWriteTo(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	for _, test := range serializeTests {
		if test.decodeFails || test.newMessageFails {
			continue
		}

		msg, _, err := NewMessage(test.arena())
		require.NoError(t, err)
		n, err := msg.WriteTo(&buf)
		if test.encodeFails {
			require.Error(t, err, test.name)
			continue
		}

		require.NoError(t, err, test.name)
		require.Equal(t, int64(len(test.out)), n, test.name)
		require.Equal(t, test.out, buf.Bytes(), test.name)

		buf.Reset()
	}
}

func TestAddCap(t *testing.T) {
	t.Parallel()

	hook1 := new(dummyHook)
	hook2 := new(dummyHook)
	client1 := NewClient(hook1)
	client2 := NewClient(hook2)
	msg, _ := NewSingleSegmentMessage(nil)

	// Simple case: distinct non-nil clients.
	id1 := msg.CapTable().Add(client1.AddRef())
	assert.Equal(t, CapabilityID(0), id1,
		"first capability ID should be 0")
	assert.Equal(t, 1, msg.CapTable().Len(),
		"should have exactly one capability in the capTable")
	assert.True(t, msg.CapTable().At(0).IsSame(client1),
		"client does not match entry in cap table")

	id2 := msg.CapTable().Add(client2.AddRef())
	assert.Equal(t, CapabilityID(1), id2,
		"second capability ID should be 1")
	assert.Equal(t, 2, msg.CapTable().Len(),
		"should have exactly two capabilities in the capTable")
	assert.True(t, msg.CapTable().At(1).IsSame(client2),
		"client does not match entry in cap table")

	// nil client
	id3 := msg.CapTable().Add(Client{})
	assert.Equal(t, CapabilityID(2), id3,
		"third capability ID should be 2")
	assert.Equal(t, 3, msg.CapTable().Len(),
		"should have exactly three capabilities in the capTable")
	assert.True(t, msg.CapTable().At(2).IsSame(Client{}),
		"client does not match entry in cap table")

	// Add should not attempt to deduplicate.
	id4 := msg.CapTable().Add(client1.AddRef())
	assert.Equal(t, CapabilityID(3), id4,
		"fourth capability ID should be 3")
	assert.Equal(t, 4, msg.CapTable().Len(),
		"should have exactly four capabilities in the capTable")
	assert.True(t, msg.CapTable().At(3).IsSame(client1),
		"client does not match entry in cap table")

	// Verify that Add steals the reference: once client1 and client2
	// and the message's capabilities released, hook1 and hook2 should be
	// shut down.  If they are not, then Add created a new reference.
	client1.Release()
	assert.Zero(t, hook1.shutdowns, "hook1 shut down before releasing msg.capTable")
	client2.Release()
	assert.Zero(t, hook2.shutdowns, "hook2 shut down before releasing msg.capTable")

	msg.CapTable().Reset()

	assert.NotZero(t, hook1.shutdowns, "hook1 not shut down after releasing msg.capTable")
	assert.NotZero(t, hook2.shutdowns, "hook2 not shut down after releasing msg.capTable")
}

func TestFirstSegmentMessage_SingleSegment(t *testing.T) {
	t.Parallel()

	msg, seg := NewSingleSegmentMessage(nil)
	if msg.NumSegments() != 1 {
		t.Errorf("msg.NumSegments() = %d; want 1", msg.NumSegments())
	}
	if seg.Message() != msg {
		t.Errorf("seg.Message() = %p; want %p", seg.Message(), msg)
	}
	if seg.ID() != 0 {
		t.Errorf("seg.ID() = %d; want 0", seg.ID())
	}
	if seg0, err := msg.Segment(0); err != nil {
		t.Errorf("msg.Segment(0): %v", err)
	} else if seg0 != seg {
		t.Errorf("msg.Segment(0) = %p; want %p", seg0, seg)
	}
}

func TestFirstSegmentMessage_MultiSegment(t *testing.T) {
	t.Parallel()

	msg, seg := NewMultiSegmentMessage(nil)
	if msg.NumSegments() != 1 {
		t.Errorf("msg.NumSegments() = %d; want 1", msg.NumSegments())
	}
	if seg.Message() != msg {
		t.Errorf("seg.Message() = %p; want %p", seg.Message(), msg)
	}
	if seg.ID() != 0 {
		t.Errorf("seg.ID() = %d; want 0", seg.ID())
	}
	if seg0, err := msg.Segment(0); err != nil {
		t.Errorf("msg.Segment(0): %v", err)
	} else if seg0 != seg {
		t.Errorf("msg.Segment(0) = %p; want %p", seg0, seg)
	}
}

func TestNextAlloc(t *testing.T) {
	t.Parallel()

	const max32 = 1<<31 - 8
	const max64 = 1<<63 - 8
	const is64bit = int64(maxInt) == math.MaxInt64
	tests := []struct {
		name string
		curr int64
		max  int64
		req  Size
		ok   bool
	}{
		{name: "zero", curr: 0, max: max64, req: 0, ok: true},
		{name: "first word", curr: 0, max: max64, req: 8, ok: true},
		{name: "first word, unaligned curr", curr: 13, max: max64, req: 8, ok: true},
		{name: "second word", curr: 8, max: max64, req: 8, ok: true},
		{name: "one byte pads to word", curr: 8, max: max64, req: 1, ok: true},
		{name: "max size", curr: 0, max: max64, req: 0xfffffff8, ok: is64bit},
		{name: "max size + 1", curr: 0, max: max64, req: 0xfffffff9, ok: false},
		{name: "max req", curr: 0, max: max64, req: 0xffffffff, ok: false},
		{name: "max curr, request 0", curr: max64, max: max64, req: 0, ok: true},
		{name: "max curr, request 1", curr: max64, max: max64, req: 1, ok: false},
		{name: "medium curr, request 2 words", curr: 4 << 20, max: max64, req: 16, ok: true},
		{name: "large curr, request word", curr: 1 << 34, max: max64, req: 8, ok: true},
		{name: "large unaligned curr, request word", curr: 1<<34 + 13, max: max64, req: 8, ok: true},
		{name: "2<<31-8 curr, request 0", curr: 2<<31 - 8, max: max64, req: 0, ok: true},
		{name: "2<<31-8 curr, request 1", curr: 2<<31 - 8, max: max64, req: 1, ok: true},
		{name: "2<<31-8 curr, 32-bit max, request 0", curr: 2<<31 - 8, max: max32, req: 0, ok: true},
		{name: "2<<31-8 curr, 32-bit max, request 1", curr: 2<<31 - 8, max: max32, req: 1, ok: false},
	}
	for _, test := range tests {
		if test.max%8 != 0 {
			t.Errorf("%s: max must be word-aligned. Skipped.", test.name)
			continue
		}
		got, err := nextAlloc(test.curr, test.max, test.req)
		if err != nil {
			if test.ok {
				t.Errorf("%s: nextAlloc(%d, %d, %d) = _, %v; want >=%d, <nil>", test.name, test.curr, test.max, test.req, err, test.req)
			}
			continue
		}
		if !test.ok {
			t.Errorf("%s: nextAlloc(%d, %d, %d) = %d, <nil>; want _, <error>", test.name, test.curr, test.max, test.req, got)
			continue
		}
		max := test.max - test.curr
		if max < 0 {
			max = 0
		}
		if int64(got) < int64(test.req) || int64(got) > max {
			t.Errorf("%s: nextAlloc(%d, %d, %d) = %d, <nil>; want in range [%d, %d]", test.name, test.curr, test.max, test.req, got, test.req, max)
		}
		if got%8 != 0 {
			t.Errorf("%s: nextAlloc(%d, %d, %d) = %d, <nil>; want divisible by 8 (word size)", test.name, test.curr, test.max, test.req, got)
		}
	}
}

// Make sure Message.TotalSize() returns a value consistent with Message.Marshal()
func TestTotalSize(t *testing.T) {
	t.Parallel()

	var emptyWord [8]byte
	err := quick.Check(func(segs [][]byte) bool {
		// Make sure there is at least one segment, and that all segments
		// are multiples of 8 bytes:
		if len(segs) == 0 {
			segs = append(segs, emptyWord[:])
		}
		for i := 0; i < len(segs); i++ {
			length := len(segs[i])
			excess := length % 8
			segs[i] = segs[i][0 : length-excess]
			if len(segs[i]) == 0 {
				segs[i] = emptyWord[:]
			}
		}

		msg, _ := NewMultiSegmentMessage(segs)

		size, err := msg.TotalSize()
		assert.Nil(t, err, "TotalSize() returned an error")

		data, err := msg.Marshal()
		assert.Nil(t, err, "Marshal() returned an error")

		assert.Equal(t, len(data), int(size), "Incorrect value")
		return true

	}, nil)
	assert.Nil(t, err, "quick.Check returned an error")
}

type readOnlyArena struct {
	Arena
}

func (ro readOnlyArena) String() string {
	return fmt.Sprintf("readOnlyArena{%v}", ro.Arena)
}

func (readOnlyArena) Allocate(sz Size, msg *Message, seg *Segment) (*Segment, address, error) {
	return nil, 0, errReadOnlyArena
}

func (ro readOnlyArena) Segment(id SegmentID) *Segment {
	return ro.Arena.Segment(id)
}

var errReadOnlyArena = errors.New("Allocate called on read-only arena")

func BenchmarkMessageGetFirstSegment(b *testing.B) {
	var msg Message

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		arena := SingleSegment(nil)
		_, err := msg.Reset(arena)
		if err != nil {
			b.Fatal(err)
		}
		_, err = msg.Segment(0)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// TestCanResetArenaForRead demonstrates that Reset() can be used when
// intending to read data from an arena.
func TestCanResetArenaForRead(t *testing.T) {
	var msg Message
	var arena Arena = SingleSegment(incrementingData(8))

	_, err := msg.Reset(arena)
	require.NoError(t, err)
}

type test_zero_stream struct{}

func (test_zero_stream) Read(b []byte) (int, error) {
	for i := 0; i < len(b); i++ {
		b[i] = 0
	}
	return len(b), nil
}

// TestZeroStream checks that an all-zero stream will fail to decode with an
// error, and not panic.
// See github.com/capnproto/go-capnp/issues/592
func TestZeroStream(t *testing.T) {
	m, err := NewDecoder(test_zero_stream{}).Decode()
	require.NoError(t, err)

	_, err = m.Root()
	assert.NotNil(t, err, "expected error decoding zero stream")
}
