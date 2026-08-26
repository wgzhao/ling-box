package dbf

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// readFPT reads a FoxPro .fpt memo block. Blocks are big-endian:
// 4-byte signature (1 = text), 4-byte length, then the data. The
// block number is an index from the file start, not from the header.
func (m *memoFile) readFPT(blockNo int) ([]byte, error) {
	bs := m.blockSize()
	if blockNo <= 0 {
		return nil, fmt.Errorf("invalid block number %d", blockNo)
	}
	off := int64(blockNo) * int64(bs)
	if off+8 > int64(len(m.data)) {
		return nil, fmt.Errorf("block %d beyond end of memo file", blockNo)
	}
	sig := binary.BigEndian.Uint32(m.data[off : off+4])
	length := binary.BigEndian.Uint32(m.data[off+4 : off+8])
	if sig != 1 {
		// Signature 0 = picture, 2 = object: not plain text.
		return nil, fmt.Errorf("block %d is not text (type %d)", blockNo, sig)
	}
	if int64(off+8+int64(length)) > int64(len(m.data)) {
		return nil, fmt.Errorf("block %d truncated", blockNo)
	}
	return m.data[off+8 : off+8+int64(length)], nil
}

// readDBT reads a dBASE III .dbt memo: fixed 512-byte blocks with no
// per-block headers, text terminated by 0x1A.
func (m *memoFile) readDBT(blockNo int) ([]byte, error) {
	const dbtBlock = 512
	if blockNo <= 0 {
		return nil, fmt.Errorf("invalid block number %d", blockNo)
	}
	off := int64(blockNo) * dbtBlock
	if off >= int64(len(m.data)) {
		return nil, fmt.Errorf("block %d beyond end of memo file", blockNo)
	}
	text := make([]byte, 0, dbtBlock)
	for off < int64(len(m.data)) {
		chunk := m.data[off:]
		if len(chunk) > dbtBlock {
			chunk = chunk[:dbtBlock]
		}
		if before, _, ok := bytes.Cut(chunk, []byte{0x1A}); ok {
			text = append(text, before...)
			break
		}
		text = append(text, chunk...)
		off += dbtBlock
	}
	return text, nil
}
