package requestbuilder

import (
	"bytes"
	"io"
	"math/rand"
	"strconv"
	"time"

	"github.com/valyala/fasttemplate"
)

// Builder is a class used to Generate bid requests.
type Builder struct {
	template3   *fasttemplate.Template
	template2   *fasttemplate.Template
	encryptor   *Encryptor
	uuidBuilder *uuidBuilder
	rng         *rand.Rand

	requestBuffer *bytes.Buffer
	formatBuffer  []byte
}

// New creates a new Builder.
func New() (*Builder, error) {
	encryptor, err := NewDefaultEncryptor()
	if err != nil {
		return nil, err
	}

	return &Builder{
		template3:     fasttemplate.New(bidRequestTemplateText3, "<", ">"),
		template2:     fasttemplate.New(bidRequestTemplateText2, "<", ">"),
		encryptor:     encryptor,
		uuidBuilder:   newUUIDBuilder(),
		rng:           rand.New(rand.NewSource(time.Now().Unix())),
		requestBuffer: &bytes.Buffer{},
	}, nil
}

// Generate generates bid request in JSON format
func (b *Builder) Generate(devicesUsed int, nobidFraction, openRTB3Fraction float64) (body []byte, version string, err error) {
	b.requestBuffer.Reset()

	template, version := b.chooseTemplate(openRTB3Fraction)

	_, err = template.ExecuteFunc(b.requestBuffer,
		func(w io.Writer, tag string) (int, error) {
			switch tag {
			case "RequestID":
				return w.Write(b.uuidBuilder.uuid())
			case "TimeStamp":
				b.formatBuffer = b.formatBuffer[:0]
				b.formatBuffer = strconv.AppendInt(b.formatBuffer, time.Now().Unix(), 10)
				return w.Write(b.formatBuffer)
			case "ItemID":
				return w.Write(b.uuidBuilder.uuid())
			case "TagID":
				return w.Write(b.uuidBuilder.uuid())
			case "UserID":
				return w.Write(b.uuidBuilder.uuid())
			case "BuyerID":
				return w.Write(b.uuidBuilder.uuid())
			case "DeviceIFA":
				return w.Write(b.generateDeviceIFA(devicesUsed, nobidFraction))
			}
			return 0, nil
		},
	)

	if err != nil {
		return nil, version, err
	}

	return b.requestBuffer.Bytes(), version, nil
}

func (b *Builder) chooseTemplate(openRTB3Fraction float64) (template *fasttemplate.Template, version string) {
	if rand.Float64() < openRTB3Fraction {
		return b.template3, "3.0"
	}
	return b.template2, "2.5"
}

func resizeBuffer(size int, buffer *[]byte) {
	if cap(*buffer) < size {
		*buffer = make([]byte, size)
	} else {
		*buffer = (*buffer)[:size]
	}
}
