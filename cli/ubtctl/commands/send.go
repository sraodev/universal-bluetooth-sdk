package commands

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/sraodev/bluetooth-service-rfcomm-python/cli/ubtctl/client"
	"github.com/sraodev/bluetooth-service-rfcomm-python/sdk/go/pkg/protocol"
)

type sendCmd struct{}

func (sendCmd) Name() string     { return "send" }
func (sendCmd) Synopsis() string { return "send a payload to a peer (file, stdin, or --data)" }

func (sendCmd) Run(ctx context.Context, args []string, invocation Invocation) error {
	fs := newFlagSet(invocation, "send")
	var b baseFlags
	bindBase(fs, &b)
	address := fs.String("address", "", "peer address (required)")
	transport := fs.String("transport", "rfcomm", "transport name")
	port := fs.Int("port", 0, "RFCOMM channel / GATT handle (0 = driver default)")
	file := fs.String("file", "", "read payload from file ('-' for stdin)")
	inline := fs.String("data", "", "inline payload string (mutually exclusive with --file)")
	if err := parseFlags(fs, args, invocation.Out); err != nil {
		return err
	}
	if *address == "" {
		return usageError(errors.New("--address is required"))
	}

	fileSet, dataSet := false, false
	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "file":
			fileSet = true
		case "data":
			dataSet = true
		}
	})
	payload, err := readPayload(invocation.In, *file, *inline, fileSet, dataSet)
	if err != nil {
		return usageError(err)
	}

	c, ctx, cancel, err := dial(ctx, b)
	if err != nil {
		return err
	}
	defer cancel()
	defer c.Close()

	res, err := c.Call(ctx, protocol.MethodSend, protocol.SendParams{
		Address:   *address,
		Transport: *transport,
		Payload:   payload,
		UUIDPort:  *port,
	})
	if err != nil {
		return err
	}
	var result protocol.SendResult
	if err := client.Decode(res, &result); err != nil {
		return err
	}
	fmt.Fprintf(invocation.Out, "sent %d bytes in %d µs\n", result.BytesSent, result.LatencyMicros)
	return nil
}

func readPayload(in io.Reader, file, inline string, fileSet, dataSet bool) ([]byte, error) {
	if fileSet && dataSet {
		return nil, errors.New("--file and --data are mutually exclusive")
	}
	if dataSet {
		return []byte(inline), nil
	}
	if fileSet {
		// Set-but-empty --file "" is a deliberate zero-length payload,
		// matching --data "". A non-empty path (including "-") is read as usual.
		if file == "" {
			return []byte{}, nil
		}
		if file == "-" {
			return io.ReadAll(in)
		}
		return os.ReadFile(file)
	}
	return nil, errors.New("provide --file or --data")
}
