// Package yenc implements a decoder for yEnc-encoded data
// commonly used in Usenet binary posts.
//
// It reads from an underlying io.Reader and decodes the yEnc data on the fly,
// validating the header and trailer metadata as it goes.

// Copied from https://github.com/go-yenc/yenc and enhanced with more robust parsing and error handling.
// The original license is MIT.
package yenc

import (
	"fmt"
	"hash"
	"hash/crc32"
	"io"
	"strconv"
	"strings"

	"gopkg.in/ringbuffer.v0"
)

type Decoder struct {
	h    Header
	r    io.Reader
	b    *ringbuffer.Buffer
	hash hash.Hash32
	s    int // state
	done bool

	// If =ybegin keywork is not at the beginning of the data stream, returns ErrRejectPrefixData
	allowPrefixData bool
	sizeDecoded     uint64
}

func Decode(r io.Reader, options ...DecodeOption) (decoder *Decoder, err error) {
	d := NewOption(options)
	if d.b == nil {
		DecodeWithBufferSize(BufferLimit)(d)
	}
	d.r = r
	d.hash = crc32.NewIEEE()
	if err = d.readHeader(); err != nil {
		return
	}
	decoder = d
	return
}

func (d *Decoder) Read(b []byte) (n int, err error) {
	if d.done {
		return 0, io.EOF
	}

	var (
		i               int
		c               byte
		hasEnd, atDelim bool
	)
	for n < len(b) {
		if d.b.IsEmpty() {
			if err = d.readMore(); err != nil {
				if err != io.EOF {
					// propagate real error (e.g., timeout)
					if n == 0 {
						return 0, err
					}
					break
				}
				if n == 0 {
					break
				}
				// if we already wrote something this iteration, keep going to flush output
				err = nil
			}
		}
		if i = d.b.IndexByteFunc(matchNotCRLF); i > 0 {
			d.b.Consume(i)
			d.s = sBegin
		} else if i < 0 && !d.b.IsEmpty() {
			// Buffer contains only CR/LF bytes — discard them all and refill
			// to avoid spinning one byte at a time without advancing output.
			d.b.Reset()
			d.s = sBegin
			continue
		}
		if d.s == sBegin && d.b.HasPrefix(yend) {
			hasEnd = true
			break
		}
		if d.s == sBegin || d.s == sData {
			// Read from buffer into output until we hit '=' or CR/LF. ReadUntilFunc advances the buffer
			// internally, so we should not consume again here.
			i, atDelim, err = d.b.ReadUntilFunc(b[n:], matchEQCRLF)
			if atDelim {
				i--
				c = b[n+i]
				if c == '=' {
					d.s = sEscape
				} else if matchCRLF(c) {
					d.s = sBegin
				}
			}
			n += i
			if err == io.EOF {
				// buffer drained out before hitting delimieter or filling output slice, should continue the line and read more
				err = nil
				continue
			}
		} else if d.s == sEscape {
			// After consuming '=', the next byte is the escaped byte. Decode it now.
			if !d.b.IsEmpty() {
				b[n] = d.b.CharAt(0) - 64
				d.b.Consume(1)
			} else {
				// Need more data to complete the escape sequence.
				if rerr := d.readMore(); rerr != nil && rerr != io.EOF {
					err = rerr
					break
				}
				if d.b.IsEmpty() {
					// Incomplete escape at EOF
					err = io.EOF
					break
				}
				b[n] = d.b.CharAt(0) - 64
				d.b.Consume(1)
			}
			n++
			d.s = sData
		}
	}
	if n > 0 {
		for i = 0; i < n; i++ {
			b[i] -= 42
		}
		d.sizeDecoded += uint64(n)
		d.hash.Write(b[:n])
	}
	if hasEnd {
		if err = d.consumeEnd(); err != nil {
			return
		}
		d.done = true
	}
	if n == 0 {
		err = io.EOF
	}
	return
}

func (d *Decoder) readMore() (err error) {
	if d.b.IsFull() {
		// Buffer is full and caller needs more data — nothing can be read.
		// Return an explicit error instead of silently returning nil which
		// caused infinite-loop stalls in readArgument / readHeader.
		return fmt.Errorf("[yEnc] ring buffer full (%d bytes) without progress: %w", d.b.Length(), ErrBufferTooSmall)
	}
	if _, err = d.b.ReadFrom(d.r); err == io.EOF && !d.b.IsEmpty() {
		err = nil
	}
	return
}

func (d *Decoder) readHeader() (err error) {
	var (
		i                       int
		key, value              string
		hasSize, hasPart, atEOL bool
	)
	for {
		if err = d.readMore(); err != nil {
			return
		}
		if d.s == sStart {
			if !d.b.HasPrefix(ybegin) {
				// there are data before the =ybegin keyword
				if !d.allowPrefixData {
					err = ErrRejectPrefixData
					return
				}
				i = d.b.IndexByteFunc(matchCRLF)
				if i < 0 {
					d.b.Reset()
				} else {
					d.b.Consume(i + 1)
				}
				continue
			}
			d.b.Consume(len(ybegin))
			for !atEOL {
				if key, value, atEOL, err = d.readArgument(func(key string) bool { return key == "name" }); err != nil {
					return
				}
				switch key {
				case "line":
					if d.h.Line, err = strconv.ParseUint(value, 10, 64); err != nil {
						err = fmt.Errorf("[yEnc] invalid line value %#v: %w", value, ErrInvalidFormat)
						return
					}
					// We should be able to handle arbitrary line size using ring buffer. Disable this check for now.

					// Each escape uses 2 bytes, and each line includes the LF byte, so at max a line can be
					// (d.h.Line * 2) + 1 bytes.

					// lineMax := (d.h.Line * 2) + 1
					// if lineMax > uint64(d.bufferSize) {
					// 	err = fmt.Errorf("[yEnc] average line length is %d, expecting buffer requirement of %d bytes, but buffer has size %d: %w", d.h.Line, lineMax, d.bufferSize, ErrBufferTooSmall)
					// 	return
					// }
				case "size":
					if d.h.Size, err = strconv.ParseUint(value, 10, 64); err != nil {
						err = fmt.Errorf("[yEnc] invalid size value %#v: %w", value, ErrInvalidFormat)
						return
					}
					hasSize = true
				case "part":
					if d.h.Part, err = strconv.ParseUint(value, 10, 64); err != nil {
						err = fmt.Errorf("[yEnc] invalid part value %#v: %w", value, ErrInvalidFormat)
						return
					}
				case "total":
					if d.h.Total, err = strconv.ParseUint(value, 10, 64); err != nil {
						err = fmt.Errorf("[yEnc] invalid total value %#v: %w", value, ErrInvalidFormat)
						return
					}
				case "name":
					// (1.2): Leading and trailing spaces will be cut by decoders!
					d.h.Name = strings.TrimSpace(value)
					if d.h.Name == "" {
						err = fmt.Errorf("[yEnc] empty name value: %w", ErrInvalidFormat)
						return
					}
				}
			}
			if d.h.Line == 0 {
				err = fmt.Errorf("[yEnc] missing line value: %w", ErrInvalidFormat)
				return
			}
			if !hasSize {
				err = fmt.Errorf("[yEnc] missing size value: %w", ErrInvalidFormat)
				return
			}
			d.s = sBegin
			hasSize = false
		} else if d.s == sBegin {
			if i := d.b.IndexByteFunc(matchNotCRLF); i > 0 {
				d.b.Consume(i)
			}
			if d.b.HasPrefix(ypart) {
				// multipart detected
				if err = d.consumePart(); err != nil {
					return
				}
				hasPart = true
			} else if d.b.HasPrefix(yend) {
				// empty file detected, end now
				if err = d.consumeEnd(); err != nil {
					return
				}
			}
			break
		}
	}
	if d.h.Part > 1 || d.h.Total > 1 {
		// multipart checks
		if !hasPart {
			err = fmt.Errorf("[yEnc] missing =ypart line for multipart: %w", ErrInvalidFormat)
			return
		}
	}
	return
}

// =ypart keyword line is seen, now consume it.
func (d *Decoder) consumePart() (err error) {
	var (
		key, value string
		atEOL      bool
	)
	d.b.Consume(len(ypart))
	for !atEOL {
		if key, value, atEOL, err = d.readArgument(nil); err != nil {
			return
		}
		switch key {
		case "begin":
			if d.h.Begin, err = strconv.ParseUint(value, 10, 64); err != nil {
				err = fmt.Errorf("[yEnc] invalid part begin value %#v: %w", value, ErrInvalidFormat)
				return
			}
			if d.h.Begin < 1 {
				err = fmt.Errorf("[yEnc] part begin raw value should start from 1 but got %d: %w", d.h.Begin, ErrInvalidFormat)
				return
			}
		case "end":
			if d.h.End, err = strconv.ParseUint(value, 10, 64); err != nil {
				err = fmt.Errorf("[yEnc] invalid part end value %#v: %w", value, ErrInvalidFormat)
				return
			}
		}
	}
	if d.h.Begin == 0 {
		err = fmt.Errorf("[yEnc] no part begin value: %w", ErrInvalidFormat)
		return
	}
	d.h.Begin-- // our contract is keep Begin a 0-based index
	if d.h.End < d.h.Begin {
		err = fmt.Errorf("[yEnc] part start %d end %d: %w", d.h.Begin, d.h.End, ErrInvalidFormat)
		return
	}
	if d.h.End > d.h.Size {
		err = fmt.Errorf("[yEnc] part end %d exceeds file size %d: %w", d.h.End, d.h.Size, ErrDataCorruption)
		return
	}
	return
}

// =yend keyword line is seen, now consume it.
func (d *Decoder) consumeEnd() (err error) {
	var (
		crc32          uint32
		u64, size      uint64
		key, value     string
		hasSize, atEOL bool
	)
	crc32 = d.hash.Sum32()
	d.b.Consume(len(yend))
	for !atEOL {
		if key, value, atEOL, err = d.readArgument(nil); err != nil {
			return
		}
		switch key {
		case "size":
			if u64, err = strconv.ParseUint(value, 10, 64); err != nil {
				err = fmt.Errorf("[yEnc] invalid trailer size value %#v: %w", value, ErrInvalidFormat)
				return
			}
			if d.h.Part > 0 {
				size = d.h.End - d.h.Begin
			} else {
				size = d.h.Size
			}
			if u64 != size {
				err = fmt.Errorf("[yEnc] header size %d != trailer size %d: %w", d.h.Size, u64, ErrDataCorruption)
				return
			}
			if d.sizeDecoded != u64 {
				err = fmt.Errorf("[yEnc] metadata has size %d but decoded data has size %d: %w", u64, d.sizeDecoded, ErrDataCorruption)
				return
			}
			hasSize = true
		case "part":
			if u64, err = strconv.ParseUint(value, 10, 64); err != nil {
				err = fmt.Errorf("[yEnc] invalid trailer part value %#v: %w", value, ErrInvalidFormat)
				return
			}
			if u64 != d.h.Part {
				err = fmt.Errorf("[yEnc] header part %d != trailer part %d: %w", d.h.Part, u64, ErrDataCorruption)
				return
			}
		case "total":
			if u64, err = strconv.ParseUint(value, 10, 64); err != nil {
				err = fmt.Errorf("[yEnc] invalid trailer total value %#v: %w", value, ErrInvalidFormat)
				return
			}
			if u64 != d.h.Total {
				err = fmt.Errorf("[yEnc] header total %d != trailer total %d: %w", d.h.Total, u64, ErrDataCorruption)
				return
			}
		case "pcrc32":
			if u64, err = strconv.ParseUint(value, 16, 32); err != nil {
				err = fmt.Errorf("[yEnc] invalid trailer pcrc32 value %#v: %w", value, ErrInvalidFormat)
				return
			}
			if uint32(u64) != crc32 {
				err = fmt.Errorf("[yEnc] expected preceding data CRC32 %#08x but got %#08x: %w", uint32(u64), crc32, ErrInvalidFormat)
				return
			}
		case "crc32":
			if u64, err = strconv.ParseUint(value, 16, 32); err != nil {
				err = fmt.Errorf("[yEnc] invalid trailer u64 value %#v: %w", value, ErrInvalidFormat)
				return
			}
			if d.sizeDecoded == d.h.Size {
				// this is the last part, validate the final CRC32 value
				if uint32(u64) != crc32 {
					err = fmt.Errorf("[yEnc] expected final file CRC32 %#08x but got %#08x: %w", uint32(u64), crc32, ErrInvalidFormat)
					return
				}
			}
		}
	}
	if !hasSize {
		err = fmt.Errorf("[yEnc] no trailer size value: %w", ErrInvalidFormat)
		return
	}
	return
}

// Read key=value pair from the line buffer. If readToEOL is nil or returns false, value ends at space or LF. If
// readToEOL returns true, value ends at LF only.
func (d *Decoder) readArgument(readToEOL func(key string) bool) (key, value string, atEOL bool, err error) {
	var token []byte

	// Skip any leading spaces between args
	if i := d.b.IndexByteFunc(func(c byte) bool { return c != ' ' && c != '\r' && c != '\n' }); i > 0 {
		d.b.Consume(i)
	}
	// If line ends, mark EOL
	if !d.b.IsEmpty() {
		c := d.b.CharAt(0)
		if matchCRLF(c) {
			d.b.Consume(1)
			atEOL = true
			return
		}
	}

	// Read key up to '=' (or CR/LF which would indicate malformed trailing token)
	for {
		token, err = d.b.ReadBytesFunc(matchEQCRLF)
		if err == io.EOF {
			// try to read more and retry
			if rerr := d.readMore(); rerr != nil && rerr != io.EOF {
				err = rerr
				return
			}
			if d.b.IsEmpty() {
				// true EOF with no more data; treat as EOL
				atEOL = true
				err = nil
				return
			}
			// got more, retry
			continue
		}
		if err != nil {
			err = fmt.Errorf("[yEnc] invalid keyword argument %#v: %w", token, ErrInvalidFormat)
			return
		}
		break
	}
	if len(token) == 0 {
		// Should not happen; treat as EOL
		atEOL = true
		return
	}
	delim := token[len(token)-1]
	key = string(token[:len(token)-1])
	d.b.Consume(len(token))
	if matchCRLF(delim) {
		// Key without '=', malformed but treat as EOL of this line
		atEOL = true
		return
	}

	// Read value: either to EOL (for name) or until space/CR/LF
	if readToEOL != nil && readToEOL(key) {
		for {
			token, err = d.b.ReadBytesFunc(matchCRLF)
			if err == io.EOF {
				// Try to read more bytes from underlying reader
				prevLen := len(d.b.Bytes())
				if rerr := d.readMore(); rerr != nil && rerr != io.EOF {
					err = rerr
					return
				}
				// If buffer length did not change, we're truly at EOF without a delimiter.
				// Treat the remaining buffer as the final token.
				if len(d.b.Bytes()) == prevLen {
					if prevLen > 0 {
						token = make([]byte, prevLen)
						copy(token, d.b.Bytes())
					}
					err = nil
					break
				}
				continue
			}
			if err != nil {
				err = fmt.Errorf("[yEnc] arg value too long: %w", ErrInvalidFormat)
				return
			}
			break
		}
		if len(token) == 0 {
			atEOL = true
			return
		}
		// If last byte is CR/LF, don't include it
		last := token[len(token)-1]
		if matchCRLF(last) {
			value = string(token[:len(token)-1])
			d.b.Consume(len(token))
			atEOL = true
		} else {
			value = string(token)
			d.b.Consume(len(token))
		}
	} else {
		for {
			token, err = d.b.ReadBytesFunc(matchSPCRLF)
			if err == io.EOF {
				// Try to read more; if nothing changes, accept remaining buffer as token
				prevLen := len(d.b.Bytes())
				if rerr := d.readMore(); rerr != nil && rerr != io.EOF {
					err = rerr
					return
				}
				if len(d.b.Bytes()) == prevLen {
					if prevLen > 0 {
						token = make([]byte, prevLen)
						copy(token, d.b.Bytes())
					}
					err = nil
					break
				}
				continue
			}
			if err != nil {
				err = fmt.Errorf("[yEnc] arg value too long: %w", ErrInvalidFormat)
				return
			}
			break
		}
		if len(token) == 0 {
			atEOL = true
			return
		}
		last := token[len(token)-1]
		if last == ' ' || matchCRLF(last) {
			value = string(token[:len(token)-1])
			d.b.Consume(len(token))
			if matchCRLF(last) {
				atEOL = true
			}
		} else {
			value = string(token)
			d.b.Consume(len(token))
		}
	}
	return
}

// CRC32 checksum of the preceeding data decoded so far.
func (d *Decoder) CRC32() uint32 {
	return d.hash.Sum32()
}

func (d *Decoder) Header() *Header {
	return &d.h
}

// Get the remaining bytes in the buffer consumed but not decoded.
func (d *Decoder) Buffer() []byte {
	return d.b.Bytes()
}

const (
	sStart = iota
	sBegin
	sEscape
	sData
)

var ybegin = []byte("=ybegin ")
var ypart = []byte("=ypart ")
var yend = []byte("=yend ")

func matchCRLF(c byte) bool {
	return c == '\r' || c == '\n'
}

func matchEQCRLF(c byte) bool {
	return c == '=' || c == '\r' || c == '\n'
}

func matchSPCRLF(c byte) bool {
	return c == ' ' || c == '\r' || c == '\n'
}

func matchNotCRLF(c byte) bool {
	return c != '\r' && c != '\n'
}

type DecodeOption func(*Decoder)

func DecodeWithPrefixData() DecodeOption {
	return func(d *Decoder) {
		d.allowPrefixData = true
	}
}

func DecodeWithBuffer(b []byte) DecodeOption {
	return func(d *Decoder) {
		d.b = ringbuffer.New(ringbuffer.WithBuffer(b))
	}
}

func DecodeWithBufferSize(size int) DecodeOption {
	return func(d *Decoder) {
		d.b = ringbuffer.New(ringbuffer.WithSize(size))
	}
}
