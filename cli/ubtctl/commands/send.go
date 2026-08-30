package commands

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/sraodev/bluetooth-service-rfcomm-python/cli/ubtctl/client"
	"github.com/sraodev/bluetooth-service-rfcomm-python/sdk/go/pkg/protocol"
)

type sendCmd struct{}

func (sendCmd) Name() string     { return "send" }
func (sendCmd) Synopsis() string { return "send a payload to a peer (file, stdin, or --data)" }

func (sendCmd) Run(args []string, _ RootInfo) error {
	fs := newFlagSet("send")
	var b baseFlags
	bindBase(fs, &b)
	address := fs.String("address", "", "peer address (required)")
	transport := fs.String("transport", "rfcomm", "transport name")
	port := fs.Int("port", 0, "RFCOMM channel / GATT handle (0 = driver default)")
	file := fs.String("file", "", "read payload from file ('-' for stdin)")
	inline := fs.String("data", "", "inline payload string (mutually exclusive with --file)")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	if *address == "" {
		return errors.New("--address is required")
	}

	payload, err := readPayload(*file, *inline)
	if err != nil {
		return err
	}

	c, ctx, cancel, err := dial(b)
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
	var r protocol.SendResult
	if err := client.Decode(res, &r); err != nil {
		return err
	}
	fmt.Printf("sent %d bytes in %d µs\n", r.BytesSent, r.LatencyMicros)
	return nil
}

func readPayload(file, inline string) ([]byte, error) {
	if file != "" && inline != "" {
		return nil, errors.New("--file and --data are mutually exclusive")
	}
	if inline != "" {
		return []byte(inline), nil
	}
	if file == "-" {
		return io.ReadAll(os.Stdin)
	}
	if file != "" {
		return os.ReadFile(file)
	}
	return nil, errors.New("provide --file or --data")
}

func init() { register(sendCmd{}) }
