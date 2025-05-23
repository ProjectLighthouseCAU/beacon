package types

// Code generated by github.com/tinylib/msgp DO NOT EDIT.

import (
	"github.com/tinylib/msgp/msgp"
)

// DecodeMsg implements msgp.Decodable
func (z *Snapshot) DecodeMsg(dc *msgp.Reader) (err error) {
	var zb0003 uint32
	zb0003, err = dc.ReadMapHeader()
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	if (*z) == nil {
		(*z) = make(Snapshot, zb0003)
	} else if len((*z)) > 0 {
		for key := range *z {
			delete((*z), key)
		}
	}
	var field []byte
	_ = field
	for zb0003 > 0 {
		zb0003--
		var zb0001 string
		var zb0002 msgp.Raw
		zb0001, err = dc.ReadString()
		if err != nil {
			err = msgp.WrapError(err)
			return
		}
		err = zb0002.DecodeMsg(dc)
		if err != nil {
			err = msgp.WrapError(err, zb0001)
			return
		}
		(*z)[zb0001] = zb0002
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z Snapshot) EncodeMsg(en *msgp.Writer) (err error) {
	err = en.WriteMapHeader(uint32(len(z)))
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	for zb0004, zb0005 := range z {
		err = en.WriteString(zb0004)
		if err != nil {
			err = msgp.WrapError(err)
			return
		}
		err = zb0005.EncodeMsg(en)
		if err != nil {
			err = msgp.WrapError(err, zb0004)
			return
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z Snapshot) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	o = msgp.AppendMapHeader(o, uint32(len(z)))
	for zb0004, zb0005 := range z {
		o = msgp.AppendString(o, zb0004)
		o, err = zb0005.MarshalMsg(o)
		if err != nil {
			err = msgp.WrapError(err, zb0004)
			return
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *Snapshot) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var zb0003 uint32
	zb0003, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	if (*z) == nil {
		(*z) = make(Snapshot, zb0003)
	} else if len((*z)) > 0 {
		for key := range *z {
			delete((*z), key)
		}
	}
	var field []byte
	_ = field
	for zb0003 > 0 {
		var zb0001 string
		var zb0002 msgp.Raw
		zb0003--
		zb0001, bts, err = msgp.ReadStringBytes(bts)
		if err != nil {
			err = msgp.WrapError(err)
			return
		}
		bts, err = zb0002.UnmarshalMsg(bts)
		if err != nil {
			err = msgp.WrapError(err, zb0001)
			return
		}
		(*z)[zb0001] = zb0002
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z Snapshot) Msgsize() (s int) {
	s = msgp.MapHeaderSize
	if z != nil {
		for zb0004, zb0005 := range z {
			_ = zb0005
			s += msgp.StringPrefixSize + len(zb0004) + zb0005.Msgsize()
		}
	}
	return
}
