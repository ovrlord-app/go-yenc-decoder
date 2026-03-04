# Go yEncode (yenc) decoder module

A Go port of the decoder functionality of [yencode 0.46](https://sourceforge.net/projects/yencode/files/yencode/0.46/) that decodes binary data for Usenet (NNTP) transfer.

Forked from [https://github.com/go-yenc/yenc](https://github.com/go-yenc/yenc) and enhanced with more robust parsing and error handling.  Encoding features ommitted from this fork.

## 📚 Guides & Documentation

- 🤝 [Contributing Guide](docs/CONTRIBUTING.md)
- 🔒 [Security Policy](docs/SECURITY.md)


### Example use

```bash
go get github.com/ovrlord-app/go-yenc-decoder
```

```go
package main

import (
	"fmt"
	"io"
	"os"

	yenc "github.com/ovrlord-app/go-yenc-decoder"
)

func decodePart(articleBody []byte, outFile *os.File) (int64, error) {
	decoder, err := yenc.Decode(articleBody, yenc.DecodeWithPrefixData())
	if err != nil {
		return 0, fmt.Errorf("yenc: %w", err)
	}

	written, err := io.Copy(outFile, decoder)
	if err != nil {
		return written, err
	}

	return written, nil
}
```

## License

This project is licensed under the MIT - see the [LICENSE](LICENSE) file for details.

## Support

- 🐛 [Issues](https://github.com/ovrlord-app/go-yenc-decoder/issues)