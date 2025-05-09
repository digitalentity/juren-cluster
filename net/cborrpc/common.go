package cborrpc

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/fxamacker/cbor/v2"
)

func readLength(reader io.Reader) (uint32, error) {
	bufHeader := make([]byte, 4)

	_, err := io.ReadFull(reader, bufHeader)
	if err == io.EOF {
		return 0, io.EOF
	} else if err != nil {
		return 0, err
	}

	return binary.LittleEndian.Uint32(bufHeader), nil
}

func writeLength(writer io.Writer, length uint32) error {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, length)

	_, err := writer.Write(buf)
	if err != nil {
		return err
	}

	return nil
}

func writeAny(writer *bufio.Writer, ror any, payload any) error {
	bufRequest, err := cbor.Marshal(&ror)
	if err != nil {
		return fmt.Errorf("cbor req/resp marshal error: %w", err)
	}

	bufPayload, err := cbor.Marshal(&payload)
	if err != nil {
		return fmt.Errorf("cbor payload marshal error: %w", err)
	}

	err = writeLength(writer, uint32(len(bufRequest)))
	if err != nil {
		return fmt.Errorf("header write error: %w", err)
	}
	_, err = writer.Write(bufRequest)
	if err != nil {
		return fmt.Errorf("req/resp write error: %w", err)
	}
	err = writeLength(writer, uint32(len(bufPayload)))
	if err != nil {
		return fmt.Errorf("header write error: %w", err)
	}
	_, err = writer.Write(bufPayload)
	if err != nil {
		return fmt.Errorf("payload write error: %w", err)
	}

	return writer.Flush()
}

func readAny(reader io.Reader, payload any) error {
	length, err := readLength(reader)
	if err != nil {
		return fmt.Errorf("header read error: %w", err)
	}

	buffer := make([]byte, length)
	_, err = io.ReadFull(reader, buffer)
	if err == io.EOF {
		return io.EOF
	} else if err != nil {
		return fmt.Errorf("payload read error: %w", err)
	}

	err = cbor.Unmarshal(buffer, payload)
	if err != nil {
		return fmt.Errorf("payload unmarshal error: %w", err)
	}

	fmt.Printf("readAny (%d bytes): %T: %v\n", length, payload, payload)

	return nil
}
