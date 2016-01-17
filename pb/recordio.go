package pb

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/golang/protobuf/proto"
)

// ReadPB reads a proto message from the reader.
func ReadPB(reader io.Reader, pb proto.Message) error {
	sizeBuf := [4]byte{}
	var err error
	if _, err = io.ReadFull(reader, sizeBuf[:]); err != nil {
		return fmt.Errorf("Failed to read 4 bytes for size: %s", err)
	}
	payloadSize := binary.BigEndian.Uint32(sizeBuf[:])
	payloadBytes := make([]byte, payloadSize)
	if _, err = io.ReadFull(reader, payloadBytes); err != nil {
		return fmt.Errorf("Failed to read %d bytes from payload: %s", payloadSize, err)
	}

	if err = proto.Unmarshal(payloadBytes, pb); err != nil {
		return fmt.Errorf("Failed to unmarshal payload bytes: %s", err)
	}
	return nil
}

// WritePB writes a proto message to the writer.
func WritePB(writer io.Writer, pb proto.Message) error {
	payloadBytes, err := proto.Marshal(pb)
	if err != nil {
		return fmt.Errorf("Failed to marshal proto: %s", proto.CompactTextString(pb))
	}

	payloadSize := [4]byte{}
	binary.BigEndian.PutUint32(payloadSize[:], uint32(len(payloadBytes)))
	if _, err = writer.Write(payloadSize[:]); err != nil {
		return fmt.Errorf("Failed to write 4 bytes for size: %s", err)
	}
	if _, err = writer.Write(payloadBytes); err != nil {
		return fmt.Errorf("Failed to write %d bytes to payload: %s", payloadSize, err)
	}
	return nil
}
