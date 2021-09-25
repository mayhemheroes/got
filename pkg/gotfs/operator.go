package gotfs

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/brendoncarroll/go-state/cadata"
	"github.com/chmduquesne/rollinghash/rabinkarp64"
	"github.com/gotvc/got/pkg/chunking"
	"github.com/gotvc/got/pkg/gdat"
	"github.com/gotvc/got/pkg/gotkv"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type (
	Ref   = gotkv.Ref
	Store = gotkv.Store
	Root  = gotkv.Root
)

const (
	DefaultMaxBlobSize             = 1 << 21
	DefaultMinBlobSizeData         = 1 << 12
	DefaultAverageBlobSizeData     = 1 << 20
	DefaultAverageBlobSizeMetadata = 1 << 13
)

type Option func(o *Operator)

func WithDataOperator(dop gdat.Operator) Option {
	return func(o *Operator) {
		o.dop = dop
	}
}

func WithSeed(seed []byte) Option {
	return func(o *Operator) {
		o.seed = append([]byte{}, seed...)
	}
}

type Operator struct {
	dop   gdat.Operator
	gotkv gotkv.Operator

	maxBlobSize                                       int
	minSizeData, averageSizeData, averageSizeMetadata int
	seed                                              []byte

	poly rabinkarp64.Pol
}

func NewOperator(opts ...Option) Operator {
	o := Operator{
		dop:                 gdat.NewOperator(),
		maxBlobSize:         DefaultMaxBlobSize,
		minSizeData:         DefaultMinBlobSizeData,
		averageSizeData:     DefaultAverageBlobSizeData,
		averageSizeMetadata: DefaultAverageBlobSizeMetadata,
	}
	for _, opt := range opts {
		opt(&o)
	}
	o.gotkv = gotkv.NewOperator(
		gotkv.WithDataOperator(o.dop),
		gotkv.WithAverageSize(o.averageSizeMetadata),
		gotkv.WithMaxSize(o.maxBlobSize),
		gotkv.WithSeed(o.seed),
	)
	o.poly = chunking.DerivePolynomial(o.seed)
	return o
}

// Select returns a new root containing everything under p, shifted to the root.
func (o *Operator) Select(ctx context.Context, s cadata.Store, root Root, p string) (*Root, error) {
	p = cleanPath(p)
	_, err := o.GetMetadata(ctx, s, root, p)
	if err != nil {
		return nil, err
	}
	x := &root
	k := makeMetadataKey(p)
	if x, err = o.deleteOutside(ctx, s, *x, gotkv.PrefixSpan(k)); err != nil {
		return nil, err
	}
	var prefix []byte
	if len(k) > 1 {
		prefix = k[:len(k)-1]
	}
	if x, err = o.gotkv.RemovePrefix(ctx, s, *x, prefix); err != nil {
		return nil, err
	}
	return x, err
}

func (o *Operator) deleteOutside(ctx context.Context, s cadata.Store, root Root, span gotkv.Span) (*Root, error) {
	x := &root
	var err error
	if x, err = o.gotkv.DeleteSpan(ctx, s, *x, gotkv.Span{Start: nil, End: span.Start}); err != nil {
		return nil, err
	}
	if x, err = o.gotkv.DeleteSpan(ctx, s, *x, gotkv.Span{Start: span.End, End: nil}); err != nil {
		return nil, err
	}
	return x, err
}

func (o *Operator) ForEach(ctx context.Context, s cadata.Store, root Root, p string, fn func(p string, md *Metadata) error) error {
	p = cleanPath(p)
	fn2 := func(ent gotkv.Entry) error {
		if !isExtentKey(ent.Key) {
			md, err := parseMetadata(ent.Value)
			if err != nil {
				return err
			}
			p, err := parseMetadataKey(ent.Key)
			if err != nil {
				return err
			}
			return fn(p, md)
		}
		return nil
	}
	k := makeMetadataKey(p)
	return o.gotkv.ForEach(ctx, s, root, gotkv.PrefixSpan(k), fn2)
}

func (o *Operator) ForEachFile(ctx context.Context, s cadata.Store, root Root, p string, fn func(p string, md *Metadata) error) error {
	return o.ForEach(ctx, s, root, p, func(p string, md *Metadata) error {
		if os.FileMode(md.Mode).IsDir() {
			return nil
		}
		return fn(p, md)
	})
}

// Graft places branch at p in root.
// If p == "" then branch is returned unaltered.
func (o *Operator) Graft(ctx context.Context, s cadata.Store, root Root, p string, branch Root) (*Root, error) {
	p = cleanPath(p)
	if p == "" {
		return &branch, nil
	}
	root2, err := o.MkdirAll(ctx, s, root, parentPath(p))
	if err != nil {
		return nil, err
	}
	b := o.gotkv.NewBuilder(s)
	k := makeMetadataKey(p)
	beforeIt := o.gotkv.NewIterator(s, *root2, gotkv.Span{Start: nil, End: k})
	branch2, err := o.gotkv.AddPrefix(ctx, s, branch, k[:len(k)-1])
	if err != nil {
		return nil, err
	}
	branchIt := o.gotkv.NewIterator(s, *branch2, gotkv.Span{})
	afterIt := o.gotkv.NewIterator(s, root, gotkv.Span{Start: gotkv.PrefixEnd(k), End: nil})
	for _, it := range []gotkv.Iterator{beforeIt, branchIt, afterIt} {
		if err := gotkv.CopyAll(ctx, b, it); err != nil {
			return nil, err
		}
	}
	return b.Finish(ctx)
}

func (o *Operator) AddPrefix(ctx context.Context, s Store, p string, x Root) (*Root, error) {
	p = cleanPath(p)
	k := makeMetadataKey(p)
	return o.gotkv.AddPrefix(ctx, s, x, k[:len(k)-1])
}

func (o *Operator) Check(ctx context.Context, s Store, root Root, checkData func(ref gdat.Ref) error) error {
	var lastPath *string
	var lastOffset *uint64
	return o.gotkv.ForEach(ctx, s, root, gotkv.Span{}, func(ent gotkv.Entry) error {
		switch {
		case lastPath == nil:
			logrus.Printf("checking root")
			if !bytes.Equal(ent.Key, []byte{Sep}) {
				logrus.Printf("first key: %q", ent.Key)
				return errors.Errorf("filesystem is missing root")
			}
			p := ""
			lastPath = &p
		case !isExtentKey(ent.Key):
			p, err := parseMetadataKey(ent.Key)
			if err != nil {
				return err
			}
			_, err = parseMetadata(ent.Value)
			if err != nil {
				return err
			}
			logrus.Printf("checking %q", p)
			if !strings.HasPrefix(*lastPath, parentPath(p)) {
				return errors.Errorf("path %s did not have parent", p)
			}
			lastPath = &p
			lastOffset = nil
		default:
			p, off, err := splitExtentKey(ent.Key)
			if err != nil {
				return err
			}
			part, err := parseExtent(ent.Value)
			if err != nil {
				return err
			}
			if *lastPath != p {
				return errors.Errorf("part not proceeded by metadata")
			}
			if lastOffset != nil && off <= *lastOffset {
				return errors.Errorf("part offsets not monotonic")
			}
			ref, err := gdat.ParseRef(part.Ref)
			if err != nil {
				return err
			}
			if err := checkData(*ref); err != nil {
				return err
			}
			lastPath = &p
			lastOffset = &off
		}
		return nil
	})
}

// Segment is a span of a GotFS instance.
type Segment struct {
	Span gotkv.Span
	Root Root
}

func (s Segment) String() string {
	return fmt.Sprintf("{ %v : %v}", s.Span, s.Root.Ref.CID)
}

func (o *Operator) Splice(ctx context.Context, ms, ds Store, segs []Segment) (*Root, error) {
	b := o.newBuilder(ctx, ms, ds)
	for _, seg := range segs {
		if err := b.CopyFrom(ctx, seg.Root, seg.Span); err != nil {
			return nil, err
		}
	}
	return b.Finish()
}

func IsEmpty(root Root) bool {
	return len(root.First) == 0
}

func Dump(ctx context.Context, s Store, root Root, w io.Writer) error {
	bw := bufio.NewWriter(w)
	op := NewOperator()
	it := op.gotkv.NewIterator(s, root, gotkv.TotalSpan())
	var ent gotkv.Entry
	for err := it.Next(ctx, &ent); err != gotkv.EOS; err = it.Next(ctx, &ent) {
		if err != nil {
			return err
		}
		switch {
		case isExtentKey(ent.Key):
			ext, err := parseExtent(ent.Value)
			if err != nil {
				fmt.Fprintf(bw, "EXTENT (INVALID):\t%q\t%q\n", ent.Key, ent.Value)
				continue
			}
			ref, err := gdat.ParseRef(ext.Ref)
			var refString string
			if err == nil {
				refString = ref.String()
			}
			fmt.Fprintf(bw, "EXTENT\t%q\toffset=%d,length=%d,ref=%s\n", ent.Key, ext.Offset, ext.Length, refString)
		default:
			md, err := parseMetadata(ent.Value)
			if err != nil {
				fmt.Fprintf(bw, "METADATA (INVALID):\t%q\t%q\n", ent.Key, ent.Value)
				continue
			}
			fmt.Fprintf(bw, "METADATA\t%q\tmode=%o,labels=%v\n", ent.Key, md.Mode, md.Labels)
		}
	}
	return bw.Flush()
}
