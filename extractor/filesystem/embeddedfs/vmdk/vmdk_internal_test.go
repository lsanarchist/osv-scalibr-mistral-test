package vmdk

import (
	"bytes"
	"encoding/binary"
	"io"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestReadHeaderAt(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		offset  int64
		want    sparseExtentHeader
		wantErr bool
	}{
		{
			name: "valid_header",
			data: func() []byte {
				hdr := sparseExtentHeader{
					MagicNumber: SparseMagic,
					Version:     1,
					Capacity:    100,
					GrainSize:   128,
				}
				buf := new(bytes.Buffer)
				binary.Write(buf, binary.LittleEndian, hdr)
				padding := make([]byte, SectorSize-buf.Len())
				buf.Write(padding)
				return buf.Bytes()
			}(),
			offset: 0,
			want: sparseExtentHeader{
				MagicNumber: SparseMagic,
				Version:     1,
				Capacity:    100,
				GrainSize:   128,
			},
			wantErr: false,
		},
		{
			name: "invalid_magic",
			data: func() []byte {
				hdr := sparseExtentHeader{
					MagicNumber: 0xDEADBEEF,
				}
				buf := new(bytes.Buffer)
				binary.Write(buf, binary.LittleEndian, hdr)
				padding := make([]byte, SectorSize-buf.Len())
				buf.Write(padding)
				return buf.Bytes()
			}(),
			offset:  0,
			wantErr: true,
		},
		{
			name:    "short_read",
			data:    make([]byte, 10),
			offset:  0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bytes.NewReader(tt.data)
			got, err := readHeaderAt(r, tt.offset)
			if (err != nil) != tt.wantErr {
				t.Errorf("readHeaderAt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if diff := cmp.Diff(tt.want, got); diff != "" {
					t.Errorf("readHeaderAt() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestGetGDGT(t *testing.T) {
	tests := []struct {
		name    string
		hdr     sparseExtentHeader
		want    *gdgtInfo
		wantErr bool
	}{
		{
			name: "valid",
			hdr: sparseExtentHeader{
				Capacity:     1024,
				GrainSize:    128,
				NumGTEsPerGT: 512,
			},
			want: &gdgtInfo{
				// lastGrainNr = 1024 / 128 = 8.
				// 1024 & 127 = 0. lastGrainSize = 0.
				// GTEs = 8.
				// GTs = (8 + 512 - 1) / 512 = 1.
				// GDsectors = (1*4 + 511) / 512 = 1.
				// GTsectors = (512*4 + 511) / 512 = 5.
				GTEs:      8,
				GTs:       1,
				GDsectors: 1,
				GTsectors: 4,
			},
			wantErr: false,
		},
		{
			name: "invalid_grain_size",
			hdr: sparseExtentHeader{
				GrainSize: 129,
			},
			wantErr: true,
		},
		{
			name: "invalid_num_gtes",
			hdr: sparseExtentHeader{
				GrainSize:    128,
				NumGTEsPerGT: 127,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getGDGT(tt.hdr)
			if (err != nil) != tt.wantErr {
				t.Errorf("getGDGT() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				// Ignore the gd slice for comparison as it is allocated but empty
				if got.GTEs != tt.want.GTEs || got.GTs != tt.want.GTs || got.GDsectors != tt.want.GDsectors || got.GTsectors != tt.want.GTsectors {
					t.Errorf("getGDGT() = %+v, want %+v", got, tt.want)
				}
			}
		})
	}
}

func TestReadStreamMarker(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		wantVal  uint64
		wantSize uint32
		wantTyp  uint32
		wantData []byte
		wantErr  bool
	}{
		{
			name: "marker_no_data",
			data: func() []byte {
				buf := new(bytes.Buffer)
				binary.Write(buf, binary.LittleEndian, uint64(123)) // val
				binary.Write(buf, binary.LittleEndian, uint32(0))   // size
				binary.Write(buf, binary.LittleEndian, uint32(1))   // typ
				// Pad to 512 (SectorSize) alignment. Consumed 16 bytes. Pad 496.
				padding := make([]byte, 496)
				buf.Write(padding)
				return buf.Bytes()
			}(),
			wantVal:  123,
			wantSize: 0,
			wantTyp:  1,
			wantData: nil,
			wantErr:  false,
		},
		{
			name: "marker_with_data",
			data: func() []byte {
				buf := new(bytes.Buffer)
				binary.Write(buf, binary.LittleEndian, uint64(456)) // val
				binary.Write(buf, binary.LittleEndian, uint32(4))   // size
				buf.Write([]byte("test"))                           // data
				// Consumed 12 + 4 = 16 bytes. Pad 496.
				padding := make([]byte, 496)
				buf.Write(padding)
				return buf.Bytes()
			}(),
			wantVal:  456,
			wantSize: 4,
			wantTyp:  0,
			wantData: []byte("test"),
			wantErr:  false,
		},
		{
			name:    "short_read_head",
			data:    make([]byte, 10),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temp file because readStreamMarker takes *os.File
			tmpFile, err := os.CreateTemp(t.TempDir(), "marker-test")
			if err != nil {
				t.Fatalf("failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())
			defer tmpFile.Close()

			if _, err := tmpFile.Write(tt.data); err != nil {
				t.Fatalf("failed to write data: %v", err)
			}
			if _, err := tmpFile.Seek(0, io.SeekStart); err != nil {
				t.Fatalf("failed to seek: %v", err)
			}

			val, size, typ, data, err := readStreamMarker(tmpFile)
			if (err != nil) != tt.wantErr {
				t.Errorf("readStreamMarker() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if val != tt.wantVal {
					t.Errorf("val = %v, want %v", val, tt.wantVal)
				}
				if size != tt.wantSize {
					t.Errorf("size = %v, want %v", size, tt.wantSize)
				}
				if typ != tt.wantTyp {
					t.Errorf("typ = %v, want %v", typ, tt.wantTyp)
				}
				if !bytes.Equal(data, tt.wantData) {
					t.Errorf("data = %v, want %v", data, tt.wantData)
				}
			}
		})
	}
}

func TestReadFooterIfGDAtEnd(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T) (*os.File, sparseExtentHeader)
		wantErr bool
	}{
		{
			name: "no_gd_at_end",
			setup: func(t *testing.T) (*os.File, sparseExtentHeader) {
				f, err := os.CreateTemp(t.TempDir(), "footer-test")
				if err != nil {
					t.Fatal(err)
				}
				return f, sparseExtentHeader{GDOffset: 100}
			},
			wantErr: false,
		},
		{
			name: "valid_footer",
			setup: func(t *testing.T) (*os.File, sparseExtentHeader) {
				f, err := os.CreateTemp(t.TempDir(), "footer-test")
				if err != nil {
					t.Fatal(err)
				}
				// File size must be >= 1536
				// Footer is at size - 1536 + 512 = size - 1024
				size := int64(2048)
				if err := f.Truncate(size); err != nil {
					t.Fatal(err)
				}
				
				hdr := sparseExtentHeader{
					MagicNumber: SparseMagic,
					Version: 2,
				}
				buf := new(bytes.Buffer)
				binary.Write(buf, binary.LittleEndian, hdr)
				
				// Write footer at correct location
				offset := size - 1024 // base + 512. base = size - 1536.
				if _, err := f.WriteAt(buf.Bytes(), offset); err != nil {
					t.Fatal(err)
				}
				
				return f, sparseExtentHeader{GDOffset: GDAtEnd}
			},
			wantErr: false,
		},
		{
			name: "file_too_small",
			setup: func(t *testing.T) (*os.File, sparseExtentHeader) {
				f, err := os.CreateTemp(t.TempDir(), "footer-test")
				if err != nil {
					t.Fatal(err)
				}
				if err := f.Truncate(1000); err != nil {
					t.Fatal(err)
				}
				return f, sparseExtentHeader{GDOffset: GDAtEnd}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, hdr := tt.setup(t)
			defer os.Remove(f.Name())
			defer f.Close()

			err := readFooterIfGDAtEnd(f, &hdr)
			if (err != nil) != tt.wantErr {
				t.Errorf("readFooterIfGDAtEnd() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestReadGD(t *testing.T) {
	tests := []struct {
		name    string
		hdr     sparseExtentHeader
		info    *gdgtInfo
		setup   func(t *testing.T) *os.File
		wantErr bool
	}{
		{
			name: "no_gd_offset",
			hdr:  sparseExtentHeader{GDOffset: 0},
			info: &gdgtInfo{},
			setup: func(t *testing.T) *os.File {
				f, err := os.CreateTemp(t.TempDir(), "gd-test")
				if err != nil {
					t.Fatal(err)
				}
				return f
			},
			wantErr: true,
		},
		{
			name: "valid_gd",
			hdr:  sparseExtentHeader{GDOffset: 1},
			info: &gdgtInfo{GDsectors: 1, gd: make([]uint32, 128)}, // 1 sector = 512 bytes = 128 uint32s
			setup: func(t *testing.T) *os.File {
				f, err := os.CreateTemp(t.TempDir(), "gd-test")
				if err != nil {
					t.Fatal(err)
				}
				// Write 512 bytes at offset 512 (SectorSize * GDOffset)
				if err := f.Truncate(1024); err != nil {
					t.Fatal(err)
				}
				data := make([]byte, 512)
				binary.LittleEndian.PutUint32(data[0:4], 0xCAFEBABE)
				if _, err := f.WriteAt(data, 512); err != nil {
					t.Fatal(err)
				}
				return f
			},
			wantErr: false,
		},
		{
			name: "short_read",
			hdr:  sparseExtentHeader{GDOffset: 1},
			info: &gdgtInfo{GDsectors: 1, gd: make([]uint32, 128)},
			setup: func(t *testing.T) *os.File {
				f, err := os.CreateTemp(t.TempDir(), "gd-test")
				if err != nil {
					t.Fatal(err)
				}
				// File too small
				if err := f.Truncate(512); err != nil {
					t.Fatal(err)
				}
				return f
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := tt.setup(t)
			defer os.Remove(f.Name())
			defer f.Close()

			err := readGD(f, tt.hdr, tt.info)
			if (err != nil) != tt.wantErr {
				t.Errorf("readGD() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && tt.info.gd[0] != 0xCAFEBABE {
				t.Errorf("gd[0] = %x, want %x", tt.info.gd[0], 0xCAFEBABE)
			}
		})
	}
}
