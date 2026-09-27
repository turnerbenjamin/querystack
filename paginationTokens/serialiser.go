// Package paginationTokens is responsible for generating pagination tokens to
// enable cursor pagination
//
// This file implements a serialiser and deserialiser for pagination tokens - To
// keep a separation from the valueBuilder package, which abstracts types from
// the rest of the query package - The general pattern followed is to provide
// methods for the serialisation and deserialisation of specific types. The
// serialiser/deserialiser is then passed to entities in the valueBuilder
// package which are responsible for calling the appropriate function for the
// concrete type
package paginationTokens

import (
	"bytes"
	"encoding/binary"
	"math"
	"time"

	qstore "github.com/turnerbenjamin/querystack/queryDataStore"
	qerr "github.com/turnerbenjamin/querystack/queryError"
	mdl "github.com/turnerbenjamin/querystack/queryModel"
)

// serialiser is used to serialise pagination tokens
type serialiser struct {
	buf *bytes.Buffer
}

// deserialiser is used to deserialise pagination tokens
type deserialiser struct {
	valueBuilder mdl.ValueBuilder
	buf          *readerBuf
}

// cursorTimeFormat is the format used for storing time values - precision is
// crucial for cursor pagination
const cursorTimeFormat = "2006-01-02T15:04:05.000000000Z"

// serialiseToken serialises a given pagination token
func serialiseToken(
	str qstore.QueryDataStore,
	version uint32,
	cursorValues []mdl.Value,
) ([]byte, error) {
	s := &serialiser{
		buf: new(bytes.Buffer),
	}

	// Write version
	s.SerialiseUint32(version)

	// Write cursor value count
	cursorValueCount := len(cursorValues)
	if cursorValueCount > math.MaxUint16 {
		return nil, qerr.InternalErr("max cursor value count exceeded: %d", math.MaxUint16)
	}
	s.SerialiseUint16(uint16(cursorValueCount))

	// Write cursor values
	if str.OrderByLen() != len(cursorValues) {
		return nil, qerr.InternalErr("mismatch between order by rules and cursor values")
	}

	for _, v := range cursorValues {
		if err := s.serialiseValue(v); err != nil {
			return nil, err
		}
	}

	// Write resource name
	s.SerialiseString(str.RootResourceMetadata().Name)

	// Write query string
	s.SerialiseString(str.QueryString())

	return s.buf.Bytes(), nil

}

// deserialiseToken deserialises a token string and returns a PagingToken
func deserialiseToken(d []byte, v mdl.ValueBuilder) (PagingToken, error) {
	ds := &deserialiser{
		// sb:           new(strings.Builder),
		buf:          newBuf(d),
		valueBuilder: v,
	}

	o := PagingToken{}

	// read version
	version, err := ds.DeserialiseUint32()
	if err != nil {
		return o, err
	}
	o.Version = version

	// read cursor value count
	cursorValueCount, err := ds.DeserialiseUint16()
	if err != nil {
		return o, err
	}

	// read cursor values
	o.CursorValues = make([]mdl.Value, cursorValueCount)
	for i := range cursorValueCount {
		v, err := ds.deserialiseValue()
		if err != nil {
			return o, err
		}
		o.CursorValues[i] = v
	}

	// read resource name
	resourceName, err := ds.ReadString()
	if err != nil {
		return o, err
	}
	o.ResourceName = resourceName

	// read query string
	queryString, err := ds.ReadString()
	if err != nil {
		return o, err
	}
	o.QueryString = queryString

	return o, nil
}

// serialiseValue serialises a cursor value
func (s *serialiser) serialiseValue(v mdl.Value) error {
	// Write type
	_ = s.buf.WriteByte(uint8(v.Type()))

	// Reserve 2 bytes for content length
	contentLenPos := s.buf.Len()
	_ = s.buf.WriteByte(0)
	_ = s.buf.WriteByte(0)
	contentStart := s.buf.Len()

	// Serialise the content
	v.Serialise(s)

	// Compute content length
	contentLenRaw := s.buf.Len() - contentStart
	if contentLenRaw > math.MaxUint16 {
		return qerr.InternalErr("token content exceeds the maximum length %d", math.MaxUint16)
	}

	// Write content length in the reserved bytes
	contentLen := uint16(contentLenRaw)
	binary.BigEndian.PutUint16(s.buf.Bytes()[contentLenPos:], contentLen)

	return nil
}

// deserialiseValue deserialises a value
func (ds *deserialiser) deserialiseValue() (mdl.Value, error) {
	// Read type
	typeByte, err := ds.buf.readByte()
	if err != nil {
		return nil, err
	}
	ltype := mdl.ValueType(typeByte)

	// Read content length
	b1, err := ds.buf.readByte()
	if err != nil {
		return nil, err
	}

	b2, err := ds.buf.readByte()
	if err != nil {
		return nil, err
	}
	contentLength := uint16(b1)<<8 | uint16(b2)

	// Read content bytes
	content, err := ds.buf.readBytes(uint64(contentLength))
	if err != nil {
		return nil, qerr.InternalErr(
			"unable to read value expression content: %w",
			err,
		)
	}

	// parse content
	return ds.valueBuilder.ExecuteDeserialisation(ds, ltype, content)
}

// SerialiseString writes a string to the serialiser buffer
func (s *serialiser) SerialiseString(str string) {
	// write string to buffer byte by byte
	for i := 0; i < len(str); i++ {
		b := str[i]

		// break on null bytes, they are used as sentinels
		if b == 0 {
			break
		}

		s.buf.WriteByte(b)
	}
	// write final null byte
	s.buf.WriteByte(0)
}

// DeserialiseString deserialises bytes into a string
func (ds *deserialiser) DeserialiseString(d []byte) (string, error) {
	i := 0
	for i < len(d) {
		if d[i] == 0 {
			return string(d[0:i]), nil
		}
		i++
	}
	return "", qerr.InternalErr(
		"unable to deserialise string: end of content reached before null terminator",
	)
}

// SerialiseStringList writes a list of strings to the serialiser buffer
func (s *serialiser) SerialiseStringList(els []string) {
	for _, el := range els {
		s.SerialiseString(el)
	}
}

// DeserialiseListString deserialises bytes into a list of strings
func (ds *deserialiser) DeserialiseListString(d []byte) ([]string, error) {
	dLen := len(d)

	o := []string{}
	i := 0
	for i < dLen {
		// read string
		s, err := ds.DeserialiseString(d[i:])
		if err != nil {
			return nil, err
		}
		o = append(o, s)

		// move i past string and separator
		i += len(s) + 1
	}
	return o, nil
}

// ReadString reads the next string from the buffer
func (ds *deserialiser) ReadString() (string, error) {
	start := ds.buf.offset
	i := 0
	for ds.buf.hasNextByte() {
		b, err := ds.buf.readByte()
		if err != nil {
			return "", err
		}

		if b == 0 {
			return string(ds.buf.bytes[start : start+uint64(i)]), nil
		}
		i++
	}

	return "", qerr.InternalErr(
		"unable to deserialise string: end of content reached before null terminator",
	)
}

// SerialiseInt writes an integer to the serialiser buffer
func (s *serialiser) SerialiseInt(n int64) {
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], uint64(n))
	_, _ = s.buf.Write(b[:])
}

// DeserialiseInt deserialises an integer value
func (ds *deserialiser) DeserialiseInt(d []byte) (int64, error) {
	if len(d) != 8 {
		return 0, qerr.InternalErr(
			"invalid int literal content length: got %d, expected 8",
			len(d),
		)
	}

	return int64(binary.BigEndian.Uint64(d)), nil
}

// SerialiseIntList writes a list or integers to the serialiser buffer
func (s *serialiser) SerialiseIntList(els []int64) {
	for _, el := range els {
		s.SerialiseInt(el)
	}
}

// DeserialiseListInt deserialises bytes into a list of integers
func (ds *deserialiser) DeserialiseListInt(d []byte) ([]int64, error) {
	// Validate data length
	dLen := len(d)
	if dLen%8 != 0 {
		return nil, qerr.InternalErr(
			"invalid int list content length: %d is not divisible by 8",
			dLen,
		)
	}

	// Initialise output slice
	listLen := dLen / 8
	o := make([]int64, listLen)

	// Loop through the data and parse the ints
	li := 0
	bi := 0
	for li < listLen {
		v, err := ds.DeserialiseInt(d[bi:])
		if err != nil {
			return nil, err
		}

		o[li] = v
		li++
		bi += 8
	}
	return o, nil
}

// SerialiseFloat writes a float to the serialiser buffer
func (s *serialiser) SerialiseFloat(f float64) {
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], math.Float64bits(f))
	_, _ = s.buf.Write(b[:])
}

// DeserialiseFloat deserialises bytes to a float64 value
func (ds *deserialiser) DeserialiseFloat(d []byte) (float64, error) {
	if len(d) != 8 {
		return 0, qerr.InternalErr(
			"invalid float literal content length: got %d, expected 8",
			len(d),
		)
	}
	return math.Float64frombits(binary.BigEndian.Uint64(d)), nil
}

// SerialiseFloatList writes a list of floats to the serialiser buffer
func (s *serialiser) SerialiseFloatList(els []float64) {
	for _, el := range els {
		s.SerialiseFloat(el)
	}
}

// DeserialiseFloat deserialises bytes to a list of float64 values
func (ds *deserialiser) DeserialiseListFloat(d []byte) ([]float64, error) {
	// Validate data length
	dLen := len(d)
	if dLen%8 != 0 {
		return nil, qerr.InternalErr(
			"invalid float list content length: %d is not divisible by 8",
			dLen,
		)
	}

	// initialise output array
	listLen := dLen / 8
	o := make([]float64, listLen)

	// loop though the data and deserialiser the floats
	li := 0
	bi := 0
	for li < listLen {
		v, err := ds.DeserialiseFloat(d[bi:])
		if err != nil {
			return nil, err
		}

		o[li] = v
		li++
		bi += 8
	}
	return o, nil
}

// SerialiseFloatList writes a time instance to the serialiser buffer
func (s *serialiser) SerialiseTime(t time.Time) {
	timeString := t.Format(cursorTimeFormat)
	s.SerialiseString(timeString)
}

// SerialiseTime deserialises a date/time string in the cursor time format
func (ds *deserialiser) DeserialiseTime(d []byte) (time.Time, error) {
	timeString, err := ds.DeserialiseString(d)
	if err != nil {
		return time.Time{}, err
	}
	return time.Parse(cursorTimeFormat, timeString)
}

// SerialisePoint writes a point value to the serialiser buffer
func (s *serialiser) SerialisePoint(p mdl.Point) {
	s.SerialiseFloat(p.Coordinates[0])
	s.SerialiseFloat(p.Coordinates[1])
}

// DeserialisePoint deserialises a point value
func (ds *deserialiser) DeserialisePoint(d []byte) (mdl.Point, error) {
	if len(d) != 16 {
		return mdl.NewPoint(0, 0), qerr.InternalErr(
			"invalid point content length: got %d, expected 16",
			len(d),
		)
	}

	long, err := ds.DeserialiseFloat(d[0:8])
	if err != nil {
		return mdl.NewPoint(0, 0), err
	}

	lat, err := ds.DeserialiseFloat(d[8:])
	if err != nil {
		return mdl.NewPoint(0, 0), err
	}

	return mdl.NewPoint(long, lat), nil
}

// SerialiseUnit32 writes a uint32 value to the serialiser buffer
func (s *serialiser) SerialiseUint32(n uint32) {
	var vb [4]byte
	binary.BigEndian.PutUint32(vb[:], n)
	_, _ = s.buf.Write(vb[:])
}

// DeserialiseUnit32 deserialises a uint32 value
func (ds *deserialiser) DeserialiseUint32() (uint32, error) {
	ibuf, err := ds.buf.readBytes(4)
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint32(ibuf), nil
}

// SerialiseUnit16 writes a uint16 value to the serialiser buffer
func (s *serialiser) SerialiseUint16(n uint16) {
	var vb [2]byte
	binary.BigEndian.PutUint16(vb[:], n)
	_, _ = s.buf.Write(vb[:])
}

// DeserialiseUnit16 deserialises a uint16 value
func (ds *deserialiser) DeserialiseUint16() (uint16, error) {
	ibuf, err := ds.buf.readBytes(2)
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint16(ibuf), nil
}

// SerialiseNull serialises a null value
func (s *serialiser) SerialiseNull() {}
