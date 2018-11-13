// Code generated by go-bindata. DO NOT EDIT.
// sources:
// internal/template/base.tmpl (470B)
// internal/template/client.tmpl (1.055kB)
// internal/template/client_impl.tmpl (2.094kB)
// internal/template/client_stream.tmpl (2.656kB)
// internal/template/fx.tmpl (2.764kB)
// internal/template/server.tmpl (2.402kB)
// internal/template/server_impl.tmpl (2.143kB)
// internal/template/server_stream.tmpl (1.743kB)

package templatedata

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func bindataRead(data []byte, name string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("read %q: %v", name, err)
	}

	var buf bytes.Buffer
	_, err = io.Copy(&buf, gz)
	clErr := gz.Close()

	if err != nil {
		return nil, fmt.Errorf("read %q: %v", name, err)
	}
	if clErr != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

type asset struct {
	bytes  []byte
	info   os.FileInfo
	digest [sha256.Size]byte
}

type bindataFileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
}

func (fi bindataFileInfo) Name() string {
	return fi.name
}
func (fi bindataFileInfo) Size() int64 {
	return fi.size
}
func (fi bindataFileInfo) Mode() os.FileMode {
	return fi.mode
}
func (fi bindataFileInfo) ModTime() time.Time {
	return fi.modTime
}
func (fi bindataFileInfo) IsDir() bool {
	return false
}
func (fi bindataFileInfo) Sys() interface{} {
	return nil
}

var _internalTemplateBaseTmpl = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x7c\x8c\x41\x4b\xc3\x40\x10\x85\xef\xfb\x2b\x9e\xa1\x07\x05\x93\xdc\x0b\x9e\xac\x4a\x2f\x6d\xc1\xde\x65\xdc\x4e\xe3\x62\x92\x5d\x36\x6b\xb1\x0c\xf3\xdf\xc5\x6e\xaa\x82\xc6\xdb\xcc\xfb\xde\xf7\xea\x1a\xb7\x7e\xc7\x68\xb8\xe7\x48\x89\x77\x78\x3e\x22\x44\x9f\xbc\x2d\x1b\xee\xcb\x23\xc5\x60\xcb\xc6\x9b\xba\xc6\xe0\xdf\xa2\xe5\x39\x44\xaa\x7b\xd7\x72\xb5\xa2\x8e\x55\x3f\xc9\x62\x8d\xd5\x7a\x8b\xbb\xc5\x72\x7b\x61\x4c\x20\xfb\x4a\x0d\x7f\xf5\x36\xf9\xaf\x1e\xfc\x78\xa9\x1a\x23\xe2\xf6\xc8\xfc\x91\xe3\xc1\x59\x1e\x50\xaa\x1a\xc0\x75\xc1\xc7\x84\x4b\x03\x00\x22\x91\xfa\x86\x31\xcb\xe9\x86\xd2\xcb\x35\x66\xd4\x3a\x1a\x30\xbf\x41\xb5\x3c\xc5\x67\x35\x1b\x19\xab\xa2\x10\xf9\xe1\xa9\x16\xe3\x24\xf7\xbb\x51\xb8\x32\xdf\x9f\x11\x49\xdc\x85\x96\x12\xa3\xb0\xad\xe3\x3e\x15\xa8\x4e\xe8\x37\x79\x72\x5d\x68\xff\xc1\x43\x8a\x4c\xdd\x5f\x85\x81\xe3\x81\xe3\x34\x99\x5c\x1e\xf1\xf4\xf2\xfe\xfd\x9c\x7e\x04\x00\x00\xff\xff\xdc\x7d\xb8\x98\xd6\x01\x00\x00")

func internalTemplateBaseTmplBytes() ([]byte, error) {
	return bindataRead(
		_internalTemplateBaseTmpl,
		"internal/template/base.tmpl",
	)
}

func internalTemplateBaseTmpl() (*asset, error) {
	bytes, err := internalTemplateBaseTmplBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "internal/template/base.tmpl", size: 470, mode: os.FileMode(420), modTime: time.Unix(1540583859, 0)}
	a := &asset{bytes: bytes, info: info, digest: [32]uint8{0xc0, 0x4, 0xb8, 0x46, 0xd, 0x49, 0xa6, 0x68, 0x60, 0xd, 0xcf, 0x2, 0xfb, 0x70, 0x8c, 0x9, 0x3d, 0x42, 0xe2, 0x5b, 0xa6, 0x56, 0xea, 0xa3, 0x44, 0x3a, 0x93, 0x36, 0xea, 0x2e, 0xbb, 0xe5}}
	return a, nil
}

var _internalTemplateClientTmpl = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xa4\x52\xcd\x6e\xdb\x3c\x10\xbc\xeb\x29\x06\xc6\xf7\xb5\xb6\xe1\xd2\xf7\x00\x3d\x14\x06\x52\xf4\x50\x37\x4d\x7a\xe9\x91\xa1\x57\x32\x11\x99\x64\x48\xaa\x6e\xb0\xe0\xbb\x17\x12\x15\xfd\xd8\x05\x5a\xa0\xba\x88\xdc\xc5\xce\xcc\x0e\x87\xf9\x40\xa5\x36\x84\x85\xaa\x35\x99\xb8\xc0\xbb\x94\x0a\xe6\xb3\x8e\x47\x88\x5b\x5d\x53\x77\xfd\xaf\xb2\xee\xa9\xc2\xcd\x7b\x88\x3b\xa9\x9e\x64\x45\xe2\xa3\xed\x4f\x29\x15\x05\xb3\x97\xa6\x22\x88\x07\xf2\x3f\xb4\xa2\xd0\xc1\x14\x00\xf3\x76\x8d\x5d\x07\x0d\x6d\x22\xf9\x52\x2a\xc2\x7a\x9b\xbb\xdb\x2d\x98\x45\x6e\xa7\x04\x1d\x10\x8f\xd4\x96\xf6\xf2\x44\x29\x21\x64\xb4\xb7\x01\xea\x02\x42\x14\x40\x7c\x71\x34\x9f\x1f\x08\xb8\x00\x5a\xf2\x5e\xd5\x67\x8a\x47\x7b\xc8\xa2\x80\xdc\xd2\x25\xac\x47\x3f\xfc\x10\x3d\xc9\x93\x36\x55\xde\x80\xfc\x58\x18\x67\x30\x2a\x5b\x0e\x25\x40\x59\x13\xe9\x67\x14\xbb\xfc\xdf\x4c\x5a\x1d\x8b\xb1\xf1\x9a\x66\x8a\xda\x7e\x6b\xe6\xca\x7e\x6b\x17\x12\xf7\xf4\xdc\x50\x88\xc8\x9e\xa7\x34\x07\x24\x73\xb8\x18\x16\x42\xbc\x48\xef\x94\xd8\xc9\xba\xfe\xe2\xa2\xb6\x66\x1c\x59\x61\xc9\x2c\x32\xef\xab\x4f\x1b\x90\xf7\xd6\xaf\x06\x2b\xa8\x0e\xf4\x4f\x7b\xfe\xa5\xfa\x3f\x29\x9d\xc1\x04\x67\x4d\xa0\x11\xe7\x4a\xf4\xc4\x88\xe9\xed\x2a\x76\xca\x9a\x10\x7d\xa3\x5a\xba\x69\xf2\xf6\x74\x9e\x86\xe7\xb1\xd1\xf5\x21\x40\xc2\xd0\x19\xdf\x3f\xdc\xdf\xed\x5e\x43\x57\x5a\xff\xfb\x5c\xb6\x21\x2c\x1b\xa3\x2e\xa0\x96\x0a\xfd\x9a\x5d\x61\x03\xeb\x62\x18\x96\x77\xde\x46\xfb\xd8\x94\x7d\x37\xdb\xb0\x9a\xe5\x38\xa7\xd7\x53\x6c\xbc\xc1\x9b\xa1\xf3\xe9\xe4\xea\x94\x38\x74\x8f\x79\x83\x39\xda\x9e\xce\xd3\x57\x5e\xaa\x0d\x98\x9d\xd7\x26\x96\x58\xfc\xff\xbc\x80\xb8\xfd\xba\x6f\x6d\x6c\xc5\x08\x21\x56\xbd\x57\xa3\x75\xe3\x71\xac\xfd\x0a\x00\x00\xff\xff\xd6\x5e\x65\x4c\x1f\x04\x00\x00")

func internalTemplateClientTmplBytes() ([]byte, error) {
	return bindataRead(
		_internalTemplateClientTmpl,
		"internal/template/client.tmpl",
	)
}

func internalTemplateClientTmpl() (*asset, error) {
	bytes, err := internalTemplateClientTmplBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "internal/template/client.tmpl", size: 1055, mode: os.FileMode(420), modTime: time.Unix(1540583859, 0)}
	a := &asset{bytes: bytes, info: info, digest: [32]uint8{0x5c, 0x33, 0xaf, 0x80, 0x4b, 0xdc, 0x36, 0x4b, 0x61, 0x90, 0x50, 0xc0, 0x17, 0xd9, 0xb0, 0x9b, 0x57, 0x3e, 0xe3, 0x64, 0x1f, 0x78, 0x3b, 0x22, 0x31, 0x91, 0xf1, 0xfa, 0xd7, 0x1b, 0x64, 0x5a}}
	return a, nil
}

var _internalTemplateClient_implTmpl = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xec\x55\xc9\x6e\xdb\x30\x10\xbd\xeb\x2b\x5e\x8c\xb4\x90\x0c\x85\xb9\xbb\xf0\xc9\x68\x8b\x1e\xba\xa0\xee\x3d\x60\xe5\x91\x42\x58\x22\x65\x92\xce\x02\x82\xff\x5e\x90\x94\xf7\xc4\xbd\xa4\x2d\x0a\xe4\x64\x72\xcc\xb7\xcc\x1b\x51\x72\x6e\x41\xb5\x90\x84\x51\xd5\x0a\x92\xf6\x46\x74\x7d\x3b\xc2\x95\xf7\x99\x73\xf7\xc2\xde\x82\x7d\x10\x2d\xc5\xed\x65\xa3\xfa\x65\x83\xc9\x14\xec\x1b\xaf\x96\xbc\x21\xf6\x51\x0d\x2b\xef\xb3\xcc\x39\xcd\x65\x43\x60\x73\xd2\x77\xa2\x22\x13\x69\x00\xe7\x2e\x13\xf9\xa7\xae\x6f\x23\x7c\xb6\xdd\x06\x5c\x38\x71\x3d\x46\x2a\x22\x18\xa0\x8e\xa4\xe5\x56\x28\x89\xf1\x75\x3a\x62\x1f\x7b\x3a\x64\xf2\x1e\xc6\xea\x75\x65\xe1\x32\x00\x61\x43\xbc\xc3\x23\xd7\x7d\xd5\x6b\x65\xd5\xcf\x75\xcd\xe6\xb1\x98\xa8\x33\x20\x52\xdd\x71\x8d\x1b\x38\x37\xd8\xf0\x1e\x53\xe4\xe3\x23\xee\x22\x97\xa2\x2d\x92\xb9\xa1\xad\xcf\x64\x6f\xd5\xc2\xc4\x9e\x42\x59\xd4\xe0\x72\xb1\xe9\x26\x29\x09\xd9\xa4\xfe\x49\xef\x0a\x57\x03\x04\xa8\xd7\xb2\x42\x5e\xe1\x44\x2d\xd8\xf9\xc2\x3b\xf2\x3e\xaf\xec\x03\x2a\x25\x2d\x3d\x58\x36\x4b\xbf\x25\x54\x6f\x0d\x18\x63\xb1\x3b\x36\xe3\x6d\xfb\xb5\x0f\xf9\x14\xc8\x9d\x3b\xe8\xd2\xfb\x12\xa4\xb5\xd2\xc5\x90\x4b\xcc\x26\xd6\x42\xf6\x15\x4b\x39\x45\x8e\x84\x0b\x8a\x25\x9c\xeb\xb5\x90\xb6\xc6\xe8\xcd\x6a\x84\xc1\x4c\x12\x66\x8c\x15\x5b\x2a\x51\x47\xaa\x8b\x29\xa4\x68\xf7\x24\x00\x4d\x76\xad\x65\x28\x47\xb5\xed\x3f\x3e\x3b\x3a\xf1\xf6\xc8\x72\xca\xc0\x25\x63\x13\x18\x5f\x06\x92\x6c\x1f\xed\x1c\xb5\x86\x82\xf8\x49\xdc\xaf\xe9\xbe\x64\xba\x7f\xe0\xd9\xd5\xb4\x0a\xa0\x46\xfd\x08\x77\x98\x7d\xa7\xd5\x9a\x8c\x45\x7a\x9d\x6c\x52\xf8\xff\xe3\x1f\xb0\x93\x29\x0c\x9b\x93\x5c\xe4\x9a\x56\xc5\xbb\xbf\x3b\xce\x93\x91\xfe\xe3\xf1\x1d\xe0\x4c\xaf\xa4\xa1\x3d\xe0\xc9\x24\x3b\xd3\x3c\x3d\xcb\xb3\x53\xd4\xb4\x2a\x21\xe9\x3e\x3f\x23\x56\xbc\xf8\x5d\x33\x25\xd4\x32\x18\xed\x4c\xc3\xce\x36\x7a\xa0\x78\xa1\x96\xcf\x4a\x1d\x7e\xba\x66\xdc\xd8\xf7\x21\xa1\xfc\xf7\xbd\x69\x32\xc5\xf3\x8f\x50\x34\xfb\xc4\x9d\x97\x8b\xed\x17\x7a\xb3\xde\xad\x76\xcb\x5d\xed\x57\x00\x00\x00\xff\xff\x86\x35\xf1\x9f\x2e\x08\x00\x00")

func internalTemplateClient_implTmplBytes() ([]byte, error) {
	return bindataRead(
		_internalTemplateClient_implTmpl,
		"internal/template/client_impl.tmpl",
	)
}

func internalTemplateClient_implTmpl() (*asset, error) {
	bytes, err := internalTemplateClient_implTmplBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "internal/template/client_impl.tmpl", size: 2094, mode: os.FileMode(420), modTime: time.Unix(1540583859, 0)}
	a := &asset{bytes: bytes, info: info, digest: [32]uint8{0xbe, 0x90, 0xab, 0xb3, 0x5c, 0x11, 0x3f, 0xdf, 0xbd, 0x70, 0x1d, 0x8b, 0x6a, 0x7c, 0x5f, 0x67, 0x92, 0x53, 0xc, 0x4d, 0x37, 0x26, 0x3, 0xbb, 0x96, 0xe6, 0xb8, 0xd4, 0x7, 0xb1, 0x84, 0x8f}}
	return a, nil
}

var _internalTemplateClient_streamTmpl = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xec\x55\xbd\x6e\xdb\x30\x10\x9e\x9b\xa7\xb8\x04\x19\x24\x23\xa5\xf7\x16\x19\x8a\xa0\x2d\x3a\x14\x2d\xe2\xee\x81\x2a\x9d\x64\xc2\xd2\x51\x21\x29\xa7\x01\xa1\x77\x2f\x48\x51\xb4\x64\x4b\x8e\x63\xa4\x5b\xa7\x30\xa7\xbb\xef\xe7\xf2\x91\x31\x26\xc3\x9c\x13\xc2\x55\x5a\x72\x24\xfd\xa0\xb4\xc4\xa4\xba\x82\xf7\x6d\x7b\x61\xcc\x13\xd7\x6b\x60\x5f\x78\x89\xee\xd7\xeb\x42\xd4\x9b\x02\x3e\xdc\x02\xfb\x99\xa4\x9b\xa4\x40\xf6\x55\xf8\x53\xdb\x5e\x5c\x18\x23\x13\x2a\x10\xd8\x0a\xe5\x96\xa7\xa8\x1c\x0c\x80\x31\xd7\x1d\xbc\x1b\xbd\x73\x47\xdb\x6f\xbf\x2c\x17\xd0\x15\xa0\xa3\x06\x4e\x1a\x65\x9e\xd8\xe9\xc5\xb2\xef\xf2\xb8\xdf\x51\xaf\x45\xd6\xc3\xda\x0f\x3c\x07\x21\x7b\xcc\x95\x43\xe0\x54\x74\x0a\x50\x86\x82\xef\x5f\x2e\xc1\x18\xd6\x55\x7b\x19\xc0\x15\x24\x9e\xdc\x8e\x06\x7e\x68\x14\x66\xc0\x09\xf4\x1a\x77\x16\xec\x40\xdf\xc1\x1c\xa8\x7e\xae\x71\x0a\x36\xe0\x18\xd7\x06\x70\x27\x48\xe3\x1f\x1d\xc5\x90\x76\x27\xe6\x2b\x3b\x2f\x07\x46\x7a\xa7\x00\x2b\xa4\x2c\x5a\x18\x53\x88\x5f\x96\x90\xdd\xe3\x63\x83\x4a\x43\xf7\x47\x69\xdb\x1b\x60\x8c\x3d\x27\xb2\x4e\xbd\x92\x1f\xb5\xe6\x82\x62\x40\x29\x85\xf4\x14\x48\xd9\x78\x79\xfb\x8b\x1a\x10\xde\x63\xba\x8d\x66\x30\xc7\x42\x54\x2d\x48\xe1\x40\x89\xa3\x8c\x7b\xdb\xa5\x50\xe8\xd4\x9f\x23\x30\xa1\xec\x70\x2b\x11\x09\x7d\x20\x3d\x1e\x68\x77\x9c\x9f\x28\x7b\x1b\x0f\x63\x59\xed\x41\x6d\x77\x9e\x89\x74\x55\x97\x58\x21\xe9\xc4\x32\xff\x83\x5c\x4f\x45\xf0\x5b\x55\x97\x6d\x6b\x25\x34\xa9\x0e\x19\xf4\x8a\x16\x6e\x21\xb5\x14\x5a\xfc\x6e\xf2\x11\x8d\xf7\xe8\x7e\x6c\x13\x09\x0f\x13\xd9\xbe\x75\xcb\x9b\xa0\x8b\x23\xe2\x65\xdc\x0d\xe7\x0d\xa5\x10\xa5\x30\xd3\x39\x7f\x1d\x82\x58\x89\xba\x91\x04\x29\xeb\x54\xb3\x30\x31\xd4\x38\x79\x6f\xfc\x5e\x5e\x92\xe0\x52\x29\xf1\x11\x8e\xde\x2b\x51\x6b\x75\xf4\x72\xcd\x0a\xee\xf1\x3b\x0c\xc6\x58\x3c\x99\xa0\xf9\xeb\x78\xa2\x0f\x17\xf3\x63\x32\x4f\xca\x7a\xb0\x51\xa9\xc2\xd5\xec\x73\x1d\xbc\xdc\x63\x8a\x7c\x8b\x11\xe1\x53\x74\x04\x2c\xde\xf3\x0a\xc0\x73\x87\x75\x79\x0b\xc4\xcb\xc0\x11\x96\x45\xbc\x74\x64\xbe\xde\x86\x55\xaa\x1b\x10\x1b\x2b\xa1\x52\x05\x3b\x6a\x60\xc0\x74\x29\x36\x33\x14\x7b\x89\x4f\x94\xfe\x6c\x5d\x9f\xe0\xa7\x52\x45\x7c\xa0\xce\x01\x3b\x91\xc4\xcb\x3e\x8f\xef\x5e\xcc\x7c\x78\x0b\xcf\x0f\x95\xc3\x88\x4e\x0c\xd4\x2b\x9e\xcf\x13\xb3\x36\x7a\x5a\xdf\x2e\x73\x3e\x25\xc3\xc4\x8d\x8d\x7e\x3c\x27\x45\xff\x93\xfc\x9a\x24\xf7\xda\x67\xe2\x34\xf9\x0f\x6f\xea\xb8\xab\xfd\x0d\x00\x00\xff\xff\x9e\x66\x58\x33\x60\x0a\x00\x00")

func internalTemplateClient_streamTmplBytes() ([]byte, error) {
	return bindataRead(
		_internalTemplateClient_streamTmpl,
		"internal/template/client_stream.tmpl",
	)
}

func internalTemplateClient_streamTmpl() (*asset, error) {
	bytes, err := internalTemplateClient_streamTmplBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "internal/template/client_stream.tmpl", size: 2656, mode: os.FileMode(420), modTime: time.Unix(1540583859, 0)}
	a := &asset{bytes: bytes, info: info, digest: [32]uint8{0x2e, 0xfe, 0x2c, 0xa6, 0x89, 0xd1, 0x46, 0x7e, 0x2, 0x55, 0xb6, 0xeb, 0x20, 0xca, 0x5a, 0x15, 0x16, 0x21, 0xe2, 0x37, 0xc7, 0x4d, 0xe7, 0xd7, 0x3a, 0x96, 0xab, 0x85, 0x4b, 0xf9, 0x7b, 0x94}}
	return a, nil
}

var _internalTemplateFxTmpl = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x9c\x56\x4f\x6f\xdb\xb8\x13\xbd\xeb\x53\xcc\xcf\xf8\x15\xb0\x0a\x97\xb9\x1b\xe8\x65\xb7\xcd\xa2\x87\xa6\xd9\x66\xb7\x97\x20\x40\x19\x69\xa4\x10\x71\x48\x66\x48\xb9\x0e\x08\x7e\xf7\x05\xff\x48\x96\x14\xb5\x71\x9a\x93\xf2\x34\x7c\xef\x91\x33\x7a\xa6\x73\x35\x36\x42\x22\xac\x9a\xc3\x0a\xde\x79\x5f\x38\xf7\x43\xd8\x3b\x60\xe7\x62\x87\xf1\xdf\xff\xb7\x4a\xdf\xb7\xb0\x7d\x0f\xec\x92\x57\xf7\xbc\x45\xf6\x97\xca\x4f\xa9\x80\xb0\xd9\x61\x65\x85\x92\xdf\x38\xc5\xc2\xaf\x03\xc2\xbe\x71\xf2\xbe\x28\x9c\x23\x2e\x5b\x04\x76\x85\xb4\x17\x15\x9a\x28\x56\x00\x38\x77\xf6\x16\xce\x0f\x50\xed\x04\x4a\x6b\xe0\xed\x59\xc2\xcf\xce\xc0\x39\x76\x7e\xf8\x33\xe2\xde\x5f\x72\xe2\x0f\x06\x92\x5d\x03\xf6\x0e\x41\x07\x08\x2d\x92\x49\xf5\x84\x8f\x9d\x20\xac\xc1\x2a\xd0\xa4\xf6\xa2\x46\xe0\x81\xa5\xe7\x00\x21\xad\x02\x2e\x53\xf9\xf9\x01\xb8\xd6\x3b\x51\xf1\x68\xb4\x00\xb0\x4f\x1a\x17\x55\x8d\xa5\xae\xb2\xe0\x0a\x00\x80\xe6\xc0\x3e\xc9\x22\x3e\xa6\xb2\xcb\xa4\x45\xf0\xc4\x49\x57\x6c\x0a\x16\x00\x8b\xfb\xf9\x8a\xa6\xdb\xd9\xde\xa7\x99\x1a\x4d\xf5\xd9\xed\x49\x46\x33\xdd\xdc\xe8\x97\xce\x8e\x9d\xce\x34\x7a\x63\x17\xf8\x63\x42\xf6\x1b\xae\x36\xd0\x19\x21\xdb\xd8\x97\x56\xec\x31\x9f\xb1\xe4\x0f\x08\x8d\x22\x20\xd5\x59\x21\x5b\x16\xe1\xf4\x2e\xf8\xcb\xa7\xb4\xce\x48\x18\x87\x34\x6f\xde\xb3\xb9\xab\xf5\xca\xa4\xd9\x79\x17\x58\x57\xe5\x66\x58\xc4\x18\xcb\xcf\x65\x01\xd0\x74\xb2\x7a\xb6\xa5\x75\x74\x62\x2c\x09\xd9\x6e\x40\x69\x6b\xc2\xaa\xd8\x30\x4d\xca\xaa\xdb\xae\xc9\xfb\xfc\xa2\xc3\x7e\xca\xb0\x4f\xa4\x86\x57\xe8\x7c\x3e\x4f\x42\xdb\x91\x8c\xfc\x6b\xbd\x34\x27\x25\xac\x17\x9a\xb2\x01\x24\x52\x54\x66\x16\xc8\xa3\xbe\x01\x75\x1f\x3e\x16\x3d\x1b\x98\xfc\x6f\x34\x5c\xe6\x15\xa2\x81\xff\xa9\xfb\x81\x60\xb0\xb2\xa0\xe6\xfc\x66\xa8\x02\x68\x1e\x2c\xfb\x18\xd4\x9b\xf5\xaa\x45\x89\xc4\x2d\xd6\x50\xa9\x1a\xa1\x52\xdd\xae\x06\xa9\x6c\x20\x23\x81\x7b\xcc\xc6\x62\xbf\xde\x3c\xae\x36\x30\xb6\xe0\x8b\x17\x95\x07\xdd\x84\x6f\x73\x13\x86\x16\x0c\xfb\xd6\xd6\x30\xc6\xca\xde\xa8\xdf\x80\x14\xbb\xa2\x57\x99\xc4\x42\x68\x39\xd2\xf3\x58\xb8\x8a\xf8\xcf\x63\x81\xff\x3a\x16\x42\x9d\x73\xac\x67\x09\x78\x85\x75\x47\x68\x4e\xff\xf4\x66\x16\x7e\x96\x11\xa9\x6c\xac\x36\x8f\x84\x1e\x9f\x47\xc2\xcc\x64\x5a\x71\x74\xfa\x2a\x93\x2f\xe5\x83\x73\xa2\x81\x75\x27\x39\x3d\x7d\x46\x7b\xa7\x6a\x03\xac\xf4\x1e\xfe\x0d\xc8\xe5\x51\x33\xfe\x5d\xdf\xa4\xa4\xfb\x28\x2b\x55\x0b\xd9\x0e\xef\xd3\xeb\xef\x2d\xa9\x4e\x6f\x57\xb1\xa8\x39\xac\xbe\x3b\x87\xb2\xf6\x7e\x24\x64\x2c\x21\x7f\x18\x2b\x5d\x45\x64\xa6\xd4\x0b\xfd\x43\x5c\x1a\xad\xc8\x4e\x95\x7e\x29\x74\xfc\x0d\xfa\x8c\x96\xc3\xf1\x47\x2a\x1f\x68\x44\x9f\x31\x2c\xa4\xe2\x78\x48\x4e\x6b\xcc\x62\x5b\xe0\x93\x05\x3c\x68\xac\x6c\x9e\x31\x3e\x99\x40\xab\xe0\x16\x41\x13\x9a\xf0\x09\x0a\x19\x45\x2a\x25\x2d\x17\x12\xe9\x77\x72\xb3\xa7\x5e\x9f\x96\x94\xc7\xf2\xd3\xa2\x6f\x3a\xfe\xe5\xe2\xb8\xb9\x85\xd0\x98\x96\x1c\x43\xe3\xc4\x09\xdc\xc2\x1f\x9d\xd8\xd5\x11\x75\x8e\x1d\x5f\x78\xbf\xd6\xf9\x38\xcb\x0d\x8c\x27\xe1\x35\x63\x97\xe9\x13\xfc\x0a\xfe\xe9\xb4\x6d\x97\xc7\xcd\x8d\x92\x39\xdf\x83\x2e\xf8\x03\x6e\xc1\x39\x4d\x42\xda\x06\x56\x6f\x1e\x57\xc0\xce\xff\xbe\xf0\x93\x18\x0f\x97\xb1\x0f\x68\x2a\x12\xda\x2a\x32\x61\xc1\xf4\xd6\x35\x2e\x1f\x1e\xfd\x38\x52\xa3\xe1\x74\xe9\x8a\xd9\x7a\x34\x9c\x33\x36\x45\xec\x9e\xd3\x02\x39\xbc\x87\xeb\x9b\xeb\x9b\xdb\x27\x8b\x6e\x08\xae\x60\x3d\x9e\x80\x73\xe3\x0b\x5f\x1f\x0a\xc9\xd2\x70\xef\xfb\x80\x1a\x65\x8d\xb2\x12\xf9\xee\x17\xbc\xcd\x88\x5e\xa2\xea\x77\x30\xde\xce\x11\xfd\x2f\x00\x00\xff\xff\x83\x4a\xa7\x00\xcc\x0a\x00\x00")

func internalTemplateFxTmplBytes() ([]byte, error) {
	return bindataRead(
		_internalTemplateFxTmpl,
		"internal/template/fx.tmpl",
	)
}

func internalTemplateFxTmpl() (*asset, error) {
	bytes, err := internalTemplateFxTmplBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "internal/template/fx.tmpl", size: 2764, mode: os.FileMode(420), modTime: time.Unix(1542075822, 0)}
	a := &asset{bytes: bytes, info: info, digest: [32]uint8{0x3, 0x19, 0x41, 0x32, 0x5b, 0xae, 0x3c, 0x8e, 0xb2, 0x77, 0x1e, 0x89, 0xab, 0x36, 0xd, 0x5, 0xd6, 0x55, 0xcf, 0xfb, 0x54, 0x72, 0xe3, 0x5c, 0x18, 0x74, 0x21, 0xdd, 0x29, 0x45, 0x68, 0xbb}}
	return a, nil
}

var _internalTemplateServerTmpl = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xb4\x54\x4b\x6f\xdb\x3c\x10\xbc\xfb\x57\x2c\x8c\x7c\x5f\xa5\x20\xa1\xef\x06\x7a\x68\x83\xa6\xed\x21\x81\x9b\xa4\x87\xa2\xe8\x81\x95\x56\x32\x11\x9b\x54\x48\x2a\x69\x40\xf0\xbf\x17\x7c\xe8\x41\xc5\x6e\xdc\x47\x7c\xb1\x48\xee\xce\xce\x2e\x67\x68\x4c\x89\x15\xe3\x08\x73\x85\xf2\x1e\xe5\x1c\x4e\xad\x9d\x19\xf3\xc0\xf4\x1a\xc8\x39\xdb\xa0\x5f\x1e\xd5\xa2\xb9\xad\x61\xf9\x1a\xc8\x8a\x16\xb7\xb4\x46\xf2\x5e\xc4\x2f\x6b\x67\x33\x63\x24\xe5\x35\x02\xb9\x46\x79\xcf\x0a\x54\x1e\x06\xc0\x98\x23\x75\x5f\xf8\xbc\x4b\xba\xf5\xa1\x6e\x73\x71\x0c\x31\x10\x18\xd7\x28\x2b\x5a\x20\x1c\x2f\xc2\xf1\x62\x01\xc6\x78\x20\x94\xd6\x02\x53\xa0\xd7\x18\x91\xac\x05\x15\x12\x5f\x29\x08\x8c\x07\x04\x32\x03\xd0\x8f\x0d\xa6\xe9\x3d\xbe\x99\x01\xb8\xe2\x91\xe9\x05\xea\xb5\x28\x3b\xa2\xe1\x88\x55\x20\x24\x90\xb3\x0d\x43\xae\xaf\xb5\x44\xba\x65\xbc\x86\x88\x36\x6c\x0c\x39\x2e\x2b\xb6\x96\xf5\x5b\x11\x8a\x0b\xfd\x14\x6b\x9c\xea\x7e\xc7\xc6\xd4\xe2\xc6\xb1\x26\x57\x78\xd7\xa2\xd2\x10\x86\x6d\xed\x49\x02\x88\xbc\x9c\x24\xbb\x36\x3d\x6e\xd7\xec\x90\x90\x8f\x18\x4c\xc8\x5b\x0b\x59\x52\x54\x35\x82\x2b\x1c\xaa\x02\x4a\x29\xa4\x43\xc0\x8d\x42\x6b\xc3\x3a\x30\x18\xcd\xca\x1d\x3e\x3f\x88\x42\x70\x8d\x3f\x34\x39\x0b\xff\xe3\x96\x0e\xea\x3c\x3f\x8c\xec\x6c\xd7\x94\xc6\xab\x41\x78\x2b\x29\x0a\x2c\x5b\x89\x8e\x9b\xd2\xb2\x2d\x34\x13\xbc\x57\x9f\x9f\x5b\xd6\x72\x2a\x1f\x3b\x85\x90\x3c\x82\x2c\x16\xf0\xb6\x65\x9b\xf2\xb3\x3b\x35\x86\xf4\x50\xca\xda\x01\x2d\xe8\xf5\xcb\x9b\xab\xd5\x19\x78\x1c\x68\xfa\x38\xa8\x84\xdc\x29\x67\xa7\xdd\xaa\xe5\xc5\xfe\x02\x99\x1a\xeb\x3a\x87\xaf\xdf\x1e\xa9\x6c\x0a\xf2\x8e\x17\xa2\x64\xbc\x1e\xfa\x0a\x42\x5f\x3b\xd7\xfd\xdf\xa7\x7c\xdc\x36\x1b\x6b\x4d\xf0\xcc\x12\x54\x18\x91\x44\xdd\x4a\x0e\x1e\xa9\x91\x42\x8b\xef\x6d\x45\x7c\xf5\xa1\x74\x77\x9d\xbf\x0c\x5a\x51\x49\xb7\xca\xf4\xf7\x16\xed\xbd\x04\x63\x1a\xc9\xb8\xae\x60\xfe\xdf\xdd\x1c\xc8\xf9\xa7\xcb\xf1\xf5\x7a\x94\x65\xd7\xcb\x1e\x70\x93\x68\x3e\xd8\x37\xbd\xa0\xa9\x31\x12\x87\x85\xa8\x27\x4c\x82\x56\x4f\x92\xd0\x0f\x94\x97\x1b\x37\x9e\x94\xce\x25\x3e\x78\x46\xf1\x38\x4b\x72\x76\x0e\x26\x46\x4e\xa7\x92\xd6\x59\xc2\x9a\xf4\xa6\x39\x99\x04\x4d\x36\xf2\x74\x19\x0d\xe3\x5c\xb1\xf4\xba\xc9\x72\xf0\x0c\xc8\x05\x2a\x45\x6b\x04\xd3\xdd\x2d\xc7\x87\x6c\xbf\xd3\xf2\xb4\xd0\x33\x6f\x4e\x7f\x1c\x3f\x72\xef\x2c\x17\x79\x0a\xe1\x75\xe8\x1d\xa4\xfc\x7b\xb3\xd7\x42\xe1\x39\x3a\xc8\x43\x01\xe9\xf7\x4d\xb4\xb3\xc4\x3e\x17\xdd\x48\xca\x55\x23\xa4\xfe\x77\x36\x0a\xf5\x9f\xf3\xd1\x34\xea\x0f\x8d\x14\x60\x9e\x3a\x69\x02\xbf\xcb\x4a\x93\x9b\x7a\x79\x2f\x05\x4e\x07\x99\x29\x09\x7d\x21\x37\xfd\xb5\xe6\x87\x9c\xe1\x73\xd8\xfb\x19\x00\x00\xff\xff\x79\xb5\xef\x24\x62\x09\x00\x00")

func internalTemplateServerTmplBytes() ([]byte, error) {
	return bindataRead(
		_internalTemplateServerTmpl,
		"internal/template/server.tmpl",
	)
}

func internalTemplateServerTmpl() (*asset, error) {
	bytes, err := internalTemplateServerTmplBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "internal/template/server.tmpl", size: 2402, mode: os.FileMode(420), modTime: time.Unix(1542075590, 0)}
	a := &asset{bytes: bytes, info: info, digest: [32]uint8{0x56, 0xb5, 0x71, 0xa0, 0x9f, 0x70, 0xc9, 0xb0, 0x0, 0xe9, 0xb4, 0x82, 0x12, 0xf9, 0x24, 0xb8, 0xb4, 0x2c, 0x87, 0xd2, 0xac, 0x37, 0x85, 0xd6, 0x98, 0x1a, 0x1c, 0x76, 0x80, 0x1e, 0xd6, 0x79}}
	return a, nil
}

var _internalTemplateServer_implTmpl = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xbc\x55\x4d\x6f\xeb\x36\x10\xbc\xfb\x57\x4c\x5f\x1f\x0a\xc9\x50\xa8\x7b\x80\x9c\x82\xb4\xe8\x21\x45\x11\x07\xe8\x31\x60\xa8\x95\x44\x98\x22\x65\x92\xb2\x63\x08\xfa\xef\x05\x29\x7f\xc8\x4e\x9c\xb4\x69\xfa\x0c\x03\x22\x97\xdc\xd9\x99\x59\x8a\xea\xfb\x82\x4a\xa9\x09\xdf\x1c\xd9\x35\xd9\x27\xd9\xb4\xea\x1b\xae\x86\x61\xd6\xf7\x1b\xe9\x6b\xb0\x5f\xa5\xa2\x38\xfd\x5e\x99\x76\x59\xe1\xfa\x06\xec\x4f\x2e\x96\xbc\x22\xf6\x9b\xd9\x8d\x86\x61\x36\xeb\x7b\xcb\x75\x45\x60\x0b\xb2\x6b\x29\xc8\x45\x18\xa0\xef\xbf\x8f\xe0\xbf\x37\xad\x8a\xe9\x8b\xc3\x34\xe4\x85\x1d\xf9\x1c\x63\x10\x81\x00\x35\xa4\x3d\xf7\xd2\x68\xcc\xf3\x71\x8b\xdf\xb6\x74\x8a\x34\x0c\x70\xde\x76\xc2\xa3\x9f\x01\xc0\xb8\x82\xbe\xdf\xc1\xc7\xda\x3b\xf8\x1d\xb1\x7b\xf2\xb5\x29\x5c\x5c\x09\x61\x59\x82\xeb\x02\xec\x56\x49\xd2\x7e\xe1\x2d\xf1\x46\xea\x6a\x4f\xf0\x18\xb8\xda\xa5\x00\x65\xa7\x05\x92\x1a\xf3\x33\x2e\x69\x28\xfc\x07\x6f\x68\x18\x12\x87\xf9\x96\xdb\x56\xb4\xd6\x78\xf3\xdc\x95\x27\x70\x29\xc8\x5a\x63\x77\xa4\xc3\xcf\x92\xef\xac\x46\xcd\x46\x3c\x76\x04\xfa\x25\x88\x89\x59\x53\xc7\x7a\x17\x43\xd7\x70\x43\xba\x03\xd9\x0b\x22\xe5\x08\xb2\x7c\xad\xe8\x7f\x16\xe0\xb2\x10\x0c\xbd\xfd\x2f\x2a\x10\xa8\x07\x9c\x9f\x6e\xa0\xa5\x9a\x54\x38\x98\x44\xd6\x1e\x82\xc3\xb9\x83\x8e\x2d\x48\x17\x89\x25\x77\xd1\x97\x1f\xdc\x58\xb1\x3e\x18\xe3\xd8\x03\x09\x92\x6b\x4a\x34\x6d\x92\xbe\xaf\xcc\x63\x38\xd2\xec\x81\x56\x1d\x39\x8f\xf1\xed\x1a\x86\xf4\x0b\xdc\x58\x65\x78\x0a\x35\x03\x01\x96\xcc\xdf\x29\x36\xad\x65\x69\x85\x9b\x8b\xb5\x4e\x85\xdf\x72\xe7\xef\x82\xdc\x8f\xd4\x64\x91\x44\x7a\xb9\x69\x6f\x1c\x98\x28\xe0\x73\x67\xff\xdf\xf5\x53\xf8\x17\x08\xa3\x3d\xbd\x78\x76\x3b\x3e\x33\x34\x88\x2a\xd9\x3d\x39\xc7\x2b\x4a\x91\x9c\xcc\xb3\xb1\xcd\xe9\x49\x9f\xf7\x86\x37\x5f\xe5\xb6\x96\x2a\xfb\xbc\xe5\xcd\xd4\xef\xc3\x30\xcf\xf1\x58\x4b\x87\xf0\xd7\x9e\x74\xb8\x5f\xb9\x62\xf8\xab\x96\x8a\x20\x3d\x1c\x51\xe3\xa0\xe4\x92\x26\x39\xd3\xf4\xf7\x9a\x26\xfc\x4b\xe8\xf5\x2a\xbd\x90\xbb\x91\x4a\xe1\x99\x6a\xbe\x26\xf8\x9a\xe0\x78\x43\xd8\xf0\x6d\x16\x67\xc5\x56\xf3\x46\x0a\xac\xb9\xea\x08\xa6\x8c\xc1\xe0\xcd\x58\x90\x0a\x48\x37\x45\x73\xde\x1a\x5d\xa9\x6d\xfc\x2e\x14\x41\x03\x69\xb4\xdc\x39\x2a\xa0\xb8\x27\x0b\xee\xc0\x75\x14\x6a\x4b\x2e\xa8\x1f\x20\xb5\x80\x29\x48\xb0\x3b\x2d\x4c\x41\x09\x63\x2c\xcd\xa6\xa0\x3e\xb8\x13\x8a\x8e\x2c\x9e\x49\x98\x86\x1c\x38\xb4\xd1\x57\x87\x78\x16\x16\x78\xe7\xa2\x65\xd2\x9d\x53\x79\xc3\x6e\xd7\x51\xd8\xc8\x97\x52\xc3\x1b\xd4\xde\xb7\xee\x3a\xcf\x2b\xa3\xb8\xae\x98\xb1\x55\x5e\x18\x91\x97\x7c\xf5\xb3\x96\xea\x29\x1e\x2f\xf6\x4f\x2f\xd7\xd7\xb6\x7f\x78\x6b\xc4\xb3\xf5\xee\x45\x1a\x6b\x6a\xa9\xce\xdf\x30\x5d\x1c\x3e\xe9\xfb\xf1\x71\x74\x1c\x1e\x63\x7f\x07\x00\x00\xff\xff\x30\x74\x79\x0e\x5f\x08\x00\x00")

func internalTemplateServer_implTmplBytes() ([]byte, error) {
	return bindataRead(
		_internalTemplateServer_implTmpl,
		"internal/template/server_impl.tmpl",
	)
}

func internalTemplateServer_implTmpl() (*asset, error) {
	bytes, err := internalTemplateServer_implTmplBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "internal/template/server_impl.tmpl", size: 2143, mode: os.FileMode(420), modTime: time.Unix(1542149039, 0)}
	a := &asset{bytes: bytes, info: info, digest: [32]uint8{0x74, 0x11, 0xe0, 0xd1, 0xf3, 0x68, 0x5c, 0x28, 0xbf, 0x1b, 0x11, 0xb5, 0x27, 0xb4, 0x24, 0x29, 0x6a, 0x17, 0x48, 0xde, 0x68, 0x4d, 0x84, 0x11, 0x7, 0x68, 0xfc, 0x54, 0x49, 0xe3, 0xcc, 0x3}}
	return a, nil
}

var _internalTemplateServer_streamTmpl = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xb4\x54\xc1\x6e\xdb\x30\x0c\xbd\xf7\x2b\xd8\xa2\x07\x3b\xc8\x94\xfb\x80\x9c\x82\x6d\xd8\x61\xd8\x90\xec\x5e\x78\x0e\xed\x08\xb1\x25\x57\x92\xdd\x15\x82\xfe\x7d\x10\x2d\x2b\x49\x63\xa5\xbd\xec\x14\x85\x26\xdf\x7b\x24\x9f\x64\xed\x1e\x2b\x2e\x10\x1e\x34\xaa\x01\xd5\x93\x36\x0a\x8b\xf6\x01\x3e\x39\x77\x67\xed\x0b\x37\x07\x60\x5f\x79\x83\xf4\xf7\xb1\x96\xdd\xb1\x86\xcf\x6b\x60\xbf\x8a\xf2\x58\xd4\xc8\xbe\xc9\x70\x72\xee\xee\xce\x5a\x55\x88\x1a\x81\xed\x50\x0d\xbc\x44\x4d\x30\x00\xd6\x3e\x8e\xf0\x54\xba\xa3\xa3\xcf\xf7\x5f\x56\x0b\x18\x03\x30\x52\x03\x17\x06\x55\x55\xf8\xea\xc5\x6a\xca\x0a\xb8\x3f\xd0\x1c\xe4\x7e\x82\xf5\x1f\x78\x05\x52\x01\xdb\x34\x1c\x85\xd9\x11\x02\x17\xf5\x44\x12\x03\x21\x7f\xb5\x02\x6b\xd9\x18\x9d\x64\x00\xd7\x50\x04\x72\x5f\x1a\xf9\xa1\xd7\xb8\x07\x2e\xc0\x1c\xf0\xd4\x82\x2f\x98\x32\x18\x81\x9a\xd7\x0e\xe7\x60\x23\x8e\xa5\x34\x80\x8d\x14\x06\xff\x9a\x2c\x87\x72\x3c\xb1\x10\x39\xf5\x72\xd5\xc8\xd4\x29\xc0\x16\xcb\x21\x63\x8c\xbd\x16\xaa\x2b\x03\xd9\xcf\xce\x70\x29\x72\xc8\x16\xd6\xd6\xf2\xb7\x17\xc2\xb6\xf8\xdc\xa3\x36\x30\x2e\xcb\xb9\x25\xa0\x52\x52\xe5\x81\x04\xc5\xfe\x72\x7c\x6f\x47\x75\x46\xb9\x43\xb1\xbf\x84\xd6\x9d\x14\x1a\xcf\xb0\x13\x82\x88\x72\x86\xd1\x5d\xc5\x4e\xe7\x84\x1d\xda\xae\xc1\x16\x85\x29\x3c\xf2\x7f\xf0\xc4\xdc\xfa\xbe\xb7\x5d\xe3\x9c\x97\xd0\x97\x26\xee\x2f\x28\x5a\x50\xc3\x9d\x92\x46\xfe\xe9\xab\x0b\xd4\xd0\x23\xfd\x0c\x85\x82\xa7\x19\x5f\xac\x69\x5b\x33\x74\x79\x26\x78\x93\x8f\xc5\x55\x2f\x4a\xc8\x34\x24\x32\xd3\x56\x8a\x62\x15\x9a\x5e\x09\xd0\x6c\x54\xcd\x62\xc5\xb9\xc6\x59\xcf\x85\xb9\xbc\x27\x81\xfc\x28\x3b\xa3\x53\x1e\xf8\x88\x29\xa3\xdc\x56\xd7\x14\xf3\x2f\x44\xd4\xbc\xc5\x12\xf9\x80\x99\xc0\x97\x2c\x8d\x95\x2f\xc1\xcb\x60\x8c\xe5\x01\x8c\x57\x04\x75\xbf\x06\xc1\x9b\x48\x11\x67\x22\x78\x43\x5c\x21\xee\xe2\xc4\x9e\x97\x20\x8f\x5e\x41\xab\x6b\x76\x4b\xfe\x19\xd1\xbd\x3c\x26\x18\x2e\x6d\xb2\x29\xb4\xf9\xe2\x7b\x7e\xbf\x9b\x56\xd7\xf9\x95\x36\xc2\x25\x89\x82\x37\xb3\x77\x29\x7d\xa5\x3f\xb8\x51\xba\xee\x0a\x29\xe1\xc6\x95\xbf\xb5\x73\xda\x6a\xd2\x83\x13\xc1\x9b\x85\x25\x1a\x99\x7d\x23\xe6\x8e\xa7\xd8\xbf\x00\x00\x00\xff\xff\x41\x80\x9a\x3a\xcf\x06\x00\x00")

func internalTemplateServer_streamTmplBytes() ([]byte, error) {
	return bindataRead(
		_internalTemplateServer_streamTmpl,
		"internal/template/server_stream.tmpl",
	)
}

func internalTemplateServer_streamTmpl() (*asset, error) {
	bytes, err := internalTemplateServer_streamTmplBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "internal/template/server_stream.tmpl", size: 1743, mode: os.FileMode(420), modTime: time.Unix(1540583859, 0)}
	a := &asset{bytes: bytes, info: info, digest: [32]uint8{0xf2, 0xf8, 0x11, 0xaa, 0x40, 0x44, 0x82, 0x90, 0xdf, 0x8b, 0x86, 0xcd, 0x10, 0xd, 0xd5, 0xaf, 0xd3, 0x83, 0x6, 0xc6, 0x20, 0xc0, 0xda, 0x92, 0xe0, 0x32, 0xf7, 0x5c, 0x83, 0x1c, 0xf, 0x4c}}
	return a, nil
}

// Asset loads and returns the asset for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func Asset(name string) ([]byte, error) {
	canonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[canonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("Asset %s can't read by error: %v", name, err)
		}
		return a.bytes, nil
	}
	return nil, fmt.Errorf("Asset %s not found", name)
}

// AssetString returns the asset contents as a string (instead of a []byte).
func AssetString(name string) (string, error) {
	data, err := Asset(name)
	return string(data), err
}

// MustAsset is like Asset but panics when Asset would return an error.
// It simplifies safe initialization of global variables.
func MustAsset(name string) []byte {
	a, err := Asset(name)
	if err != nil {
		panic("asset: Asset(" + name + "): " + err.Error())
	}

	return a
}

// MustAssetString is like AssetString but panics when Asset would return an
// error. It simplifies safe initialization of global variables.
func MustAssetString(name string) string {
	return string(MustAsset(name))
}

// AssetInfo loads and returns the asset info for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func AssetInfo(name string) (os.FileInfo, error) {
	canonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[canonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("AssetInfo %s can't read by error: %v", name, err)
		}
		return a.info, nil
	}
	return nil, fmt.Errorf("AssetInfo %s not found", name)
}

// AssetDigest returns the digest of the file with the given name. It returns an
// error if the asset could not be found or the digest could not be loaded.
func AssetDigest(name string) ([sha256.Size]byte, error) {
	canonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[canonicalName]; ok {
		a, err := f()
		if err != nil {
			return [sha256.Size]byte{}, fmt.Errorf("AssetDigest %s can't read by error: %v", name, err)
		}
		return a.digest, nil
	}
	return [sha256.Size]byte{}, fmt.Errorf("AssetDigest %s not found", name)
}

// Digests returns a map of all known files and their checksums.
func Digests() (map[string][sha256.Size]byte, error) {
	mp := make(map[string][sha256.Size]byte, len(_bindata))
	for name := range _bindata {
		a, err := _bindata[name]()
		if err != nil {
			return nil, err
		}
		mp[name] = a.digest
	}
	return mp, nil
}

// AssetNames returns the names of the assets.
func AssetNames() []string {
	names := make([]string, 0, len(_bindata))
	for name := range _bindata {
		names = append(names, name)
	}
	return names
}

// _bindata is a table, holding each asset generator, mapped to its name.
var _bindata = map[string]func() (*asset, error){
	"internal/template/base.tmpl": internalTemplateBaseTmpl,

	"internal/template/client.tmpl": internalTemplateClientTmpl,

	"internal/template/client_impl.tmpl": internalTemplateClient_implTmpl,

	"internal/template/client_stream.tmpl": internalTemplateClient_streamTmpl,

	"internal/template/fx.tmpl": internalTemplateFxTmpl,

	"internal/template/server.tmpl": internalTemplateServerTmpl,

	"internal/template/server_impl.tmpl": internalTemplateServer_implTmpl,

	"internal/template/server_stream.tmpl": internalTemplateServer_streamTmpl,
}

// AssetDir returns the file names below a certain
// directory embedded in the file by go-bindata.
// For example if you run go-bindata on data/... and data contains the
// following hierarchy:
//     data/
//       foo.txt
//       img/
//         a.png
//         b.png
// then AssetDir("data") would return []string{"foo.txt", "img"},
// AssetDir("data/img") would return []string{"a.png", "b.png"},
// AssetDir("foo.txt") and AssetDir("notexist") would return an error, and
// AssetDir("") will return []string{"data"}.
func AssetDir(name string) ([]string, error) {
	node := _bintree
	if len(name) != 0 {
		canonicalName := strings.Replace(name, "\\", "/", -1)
		pathList := strings.Split(canonicalName, "/")
		for _, p := range pathList {
			node = node.Children[p]
			if node == nil {
				return nil, fmt.Errorf("Asset %s not found", name)
			}
		}
	}
	if node.Func != nil {
		return nil, fmt.Errorf("Asset %s not found", name)
	}
	rv := make([]string, 0, len(node.Children))
	for childName := range node.Children {
		rv = append(rv, childName)
	}
	return rv, nil
}

type bintree struct {
	Func     func() (*asset, error)
	Children map[string]*bintree
}

var _bintree = &bintree{nil, map[string]*bintree{
	"internal": &bintree{nil, map[string]*bintree{
		"template": &bintree{nil, map[string]*bintree{
			"base.tmpl":          &bintree{internalTemplateBaseTmpl, map[string]*bintree{}},
			"client.tmpl":        &bintree{internalTemplateClientTmpl, map[string]*bintree{}},
			"client_impl.tmpl":   &bintree{internalTemplateClient_implTmpl, map[string]*bintree{}},
			"client_stream.tmpl": &bintree{internalTemplateClient_streamTmpl, map[string]*bintree{}},
			"fx.tmpl":            &bintree{internalTemplateFxTmpl, map[string]*bintree{}},
			"server.tmpl":        &bintree{internalTemplateServerTmpl, map[string]*bintree{}},
			"server_impl.tmpl":   &bintree{internalTemplateServer_implTmpl, map[string]*bintree{}},
			"server_stream.tmpl": &bintree{internalTemplateServer_streamTmpl, map[string]*bintree{}},
		}},
	}},
}}

// RestoreAsset restores an asset under the given directory.
func RestoreAsset(dir, name string) error {
	data, err := Asset(name)
	if err != nil {
		return err
	}
	info, err := AssetInfo(name)
	if err != nil {
		return err
	}
	err = os.MkdirAll(_filePath(dir, filepath.Dir(name)), os.FileMode(0755))
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(_filePath(dir, name), data, info.Mode())
	if err != nil {
		return err
	}
	return os.Chtimes(_filePath(dir, name), info.ModTime(), info.ModTime())
}

// RestoreAssets restores an asset under the given directory recursively.
func RestoreAssets(dir, name string) error {
	children, err := AssetDir(name)
	// File
	if err != nil {
		return RestoreAsset(dir, name)
	}
	// Dir
	for _, child := range children {
		err = RestoreAssets(dir, filepath.Join(name, child))
		if err != nil {
			return err
		}
	}
	return nil
}

func _filePath(dir, name string) string {
	canonicalName := strings.Replace(name, "\\", "/", -1)
	return filepath.Join(append([]string{dir}, strings.Split(canonicalName, "/")...)...)
}
