package codec

import "github.com/vmihailenco/msgpack/v5"

// Codec 编解码缓存载荷。
type Codec interface {
	Marshal(v any) ([]byte, error)
	Unmarshal(b []byte, dst any) error
}

type msgpackCodec struct{}

// Msgpack 返回 msgpack Codec（与旧业务缓存同一编码，避免 gob 绑包路径）。
func Msgpack() Codec {
	return msgpackCodec{}
}

func (msgpackCodec) Marshal(v any) ([]byte, error) {
	return msgpack.Marshal(v)
}

func (msgpackCodec) Unmarshal(b []byte, dst any) error {
	return msgpack.Unmarshal(b, dst)
}
