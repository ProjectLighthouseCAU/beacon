package types

// Code generated by github.com/tinylib/msgp DO NOT EDIT.

import (
	"github.com/tinylib/msgp/msgp"
)

// DecodeMsg implements msgp.Decodable
func (z *Request) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, err = dc.ReadMapHeader()
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			err = msgp.WrapError(err)
			return
		}
		switch msgp.UnsafeString(field) {
		case "REID":
			err = z.REID.DecodeMsg(dc)
			if err != nil {
				err = msgp.WrapError(err, "REID")
				return
			}
		case "AUTH":
			var zb0002 uint32
			zb0002, err = dc.ReadMapHeader()
			if err != nil {
				err = msgp.WrapError(err, "AUTH")
				return
			}
			if z.AUTH == nil {
				z.AUTH = make(map[string]string, zb0002)
			} else if len(z.AUTH) > 0 {
				for key := range z.AUTH {
					delete(z.AUTH, key)
				}
			}
			for zb0002 > 0 {
				zb0002--
				var za0001 string
				var za0002 string
				za0001, err = dc.ReadString()
				if err != nil {
					err = msgp.WrapError(err, "AUTH")
					return
				}
				za0002, err = dc.ReadString()
				if err != nil {
					err = msgp.WrapError(err, "AUTH", za0001)
					return
				}
				z.AUTH[za0001] = za0002
			}
		case "VERB":
			z.VERB, err = dc.ReadString()
			if err != nil {
				err = msgp.WrapError(err, "VERB")
				return
			}
		case "PATH":
			var zb0003 uint32
			zb0003, err = dc.ReadArrayHeader()
			if err != nil {
				err = msgp.WrapError(err, "PATH")
				return
			}
			if cap(z.PATH) >= int(zb0003) {
				z.PATH = (z.PATH)[:zb0003]
			} else {
				z.PATH = make([]string, zb0003)
			}
			for za0003 := range z.PATH {
				z.PATH[za0003], err = dc.ReadString()
				if err != nil {
					err = msgp.WrapError(err, "PATH", za0003)
					return
				}
			}
		case "PAYL":
			err = z.PAYL.DecodeMsg(dc)
			if err != nil {
				err = msgp.WrapError(err, "PAYL")
				return
			}
		default:
			err = dc.Skip()
			if err != nil {
				err = msgp.WrapError(err)
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *Request) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 5
	// write "REID"
	err = en.Append(0x85, 0xa4, 0x52, 0x45, 0x49, 0x44)
	if err != nil {
		return
	}
	err = z.REID.EncodeMsg(en)
	if err != nil {
		err = msgp.WrapError(err, "REID")
		return
	}
	// write "AUTH"
	err = en.Append(0xa4, 0x41, 0x55, 0x54, 0x48)
	if err != nil {
		return
	}
	err = en.WriteMapHeader(uint32(len(z.AUTH)))
	if err != nil {
		err = msgp.WrapError(err, "AUTH")
		return
	}
	for za0001, za0002 := range z.AUTH {
		err = en.WriteString(za0001)
		if err != nil {
			err = msgp.WrapError(err, "AUTH")
			return
		}
		err = en.WriteString(za0002)
		if err != nil {
			err = msgp.WrapError(err, "AUTH", za0001)
			return
		}
	}
	// write "VERB"
	err = en.Append(0xa4, 0x56, 0x45, 0x52, 0x42)
	if err != nil {
		return
	}
	err = en.WriteString(z.VERB)
	if err != nil {
		err = msgp.WrapError(err, "VERB")
		return
	}
	// write "PATH"
	err = en.Append(0xa4, 0x50, 0x41, 0x54, 0x48)
	if err != nil {
		return
	}
	err = en.WriteArrayHeader(uint32(len(z.PATH)))
	if err != nil {
		err = msgp.WrapError(err, "PATH")
		return
	}
	for za0003 := range z.PATH {
		err = en.WriteString(z.PATH[za0003])
		if err != nil {
			err = msgp.WrapError(err, "PATH", za0003)
			return
		}
	}
	// write "PAYL"
	err = en.Append(0xa4, 0x50, 0x41, 0x59, 0x4c)
	if err != nil {
		return
	}
	err = z.PAYL.EncodeMsg(en)
	if err != nil {
		err = msgp.WrapError(err, "PAYL")
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *Request) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 5
	// string "REID"
	o = append(o, 0x85, 0xa4, 0x52, 0x45, 0x49, 0x44)
	o, err = z.REID.MarshalMsg(o)
	if err != nil {
		err = msgp.WrapError(err, "REID")
		return
	}
	// string "AUTH"
	o = append(o, 0xa4, 0x41, 0x55, 0x54, 0x48)
	o = msgp.AppendMapHeader(o, uint32(len(z.AUTH)))
	for za0001, za0002 := range z.AUTH {
		o = msgp.AppendString(o, za0001)
		o = msgp.AppendString(o, za0002)
	}
	// string "VERB"
	o = append(o, 0xa4, 0x56, 0x45, 0x52, 0x42)
	o = msgp.AppendString(o, z.VERB)
	// string "PATH"
	o = append(o, 0xa4, 0x50, 0x41, 0x54, 0x48)
	o = msgp.AppendArrayHeader(o, uint32(len(z.PATH)))
	for za0003 := range z.PATH {
		o = msgp.AppendString(o, z.PATH[za0003])
	}
	// string "PAYL"
	o = append(o, 0xa4, 0x50, 0x41, 0x59, 0x4c)
	o, err = z.PAYL.MarshalMsg(o)
	if err != nil {
		err = msgp.WrapError(err, "PAYL")
		return
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *Request) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			err = msgp.WrapError(err)
			return
		}
		switch msgp.UnsafeString(field) {
		case "REID":
			bts, err = z.REID.UnmarshalMsg(bts)
			if err != nil {
				err = msgp.WrapError(err, "REID")
				return
			}
		case "AUTH":
			var zb0002 uint32
			zb0002, bts, err = msgp.ReadMapHeaderBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "AUTH")
				return
			}
			if z.AUTH == nil {
				z.AUTH = make(map[string]string, zb0002)
			} else if len(z.AUTH) > 0 {
				for key := range z.AUTH {
					delete(z.AUTH, key)
				}
			}
			for zb0002 > 0 {
				var za0001 string
				var za0002 string
				zb0002--
				za0001, bts, err = msgp.ReadStringBytes(bts)
				if err != nil {
					err = msgp.WrapError(err, "AUTH")
					return
				}
				za0002, bts, err = msgp.ReadStringBytes(bts)
				if err != nil {
					err = msgp.WrapError(err, "AUTH", za0001)
					return
				}
				z.AUTH[za0001] = za0002
			}
		case "VERB":
			z.VERB, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "VERB")
				return
			}
		case "PATH":
			var zb0003 uint32
			zb0003, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "PATH")
				return
			}
			if cap(z.PATH) >= int(zb0003) {
				z.PATH = (z.PATH)[:zb0003]
			} else {
				z.PATH = make([]string, zb0003)
			}
			for za0003 := range z.PATH {
				z.PATH[za0003], bts, err = msgp.ReadStringBytes(bts)
				if err != nil {
					err = msgp.WrapError(err, "PATH", za0003)
					return
				}
			}
		case "PAYL":
			bts, err = z.PAYL.UnmarshalMsg(bts)
			if err != nil {
				err = msgp.WrapError(err, "PAYL")
				return
			}
		default:
			bts, err = msgp.Skip(bts)
			if err != nil {
				err = msgp.WrapError(err)
				return
			}
		}
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *Request) Msgsize() (s int) {
	s = 1 + 5 + z.REID.Msgsize() + 5 + msgp.MapHeaderSize
	if z.AUTH != nil {
		for za0001, za0002 := range z.AUTH {
			_ = za0002
			s += msgp.StringPrefixSize + len(za0001) + msgp.StringPrefixSize + len(za0002)
		}
	}
	s += 5 + msgp.StringPrefixSize + len(z.VERB) + 5 + msgp.ArrayHeaderSize
	for za0003 := range z.PATH {
		s += msgp.StringPrefixSize + len(z.PATH[za0003])
	}
	s += 5 + z.PAYL.Msgsize()
	return
}
