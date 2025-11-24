package java

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestIsStdLib(t *testing.T) {
	tests := []struct {
		class string
		want  bool
	}{
		{"java/lang/String", true},
		{"javax/servlet/http/HttpServlet", true},
		{"jdk/internal/misc/Unsafe", true},
		{"sun/misc/Unsafe", true},
		{"org/ietf/jgss/GSSContext", true},
		{"org/omg/CORBA/ORB", true},
		{"org/w3c/dom/Node", true},
		{"org/xml/sax/XMLReader", true},
		{"com/google/common/collect/Lists", false},
		{"org/apache/commons/lang3/StringUtils", false},
	}

	for _, tt := range tests {
		if got := IsStdLib(tt.class); got != tt.want {
			t.Errorf("IsStdLib(%q) = %v, want %v", tt.class, got, tt.want)
		}
	}
}

func TestConstantPoolInfo_Type(t *testing.T) {
	tests := []struct {
		cp   ConstantPoolInfo
		want ConstantKind
	}{
		{ConstantClass{}, ConstantKindClass},
		{ConstantFieldref{}, ConstantKindFieldref},
		{ConstantMethodref{}, ConstantKindMethodref},
		{ConstantInterfaceMethodref{}, ConstantKindInterfaceMethodref},
		{ConstantString{}, ConstantKindString},
		{ConstantInteger{}, ConstantKindInteger},
		{ConstantFloat{}, ConstantKindFloat},
		{ConstantLong{}, ConstantKindLong},
		{ConstantDouble{}, ConstantKindDouble},
		{ConstantNameAndType{}, ConstantKindNameAndType},
		{ConstantUtf8{}, ConstantKindUtf8},
		{ConstantMethodHandle{}, ConstantKindMethodHandle},
		{ConstantMethodType{}, ConstantKindMethodType},
		{ConstantInvokeDynamic{}, ConstantKindInvokeDynamic},
		{ConstantModule{}, ConstantKindModule},
		{ConstantPackage{}, ConstantKindPackage},
		{ConstantDynamic{}, ConstantKindDynamic},
		{ConstantPlaceholder{}, ConstantKindPlaceholder},
	}

	for _, tt := range tests {
		if got := tt.cp.Type(); got != tt.want {
			t.Errorf("%T.Type() = %v, want %v", tt.cp, got, tt.want)
		}
	}
}

func TestParseClass_InvalidMagic(t *testing.T) {
	data := []byte{0x00, 0x00, 0x00, 0x00} // Invalid magic
	r := bytes.NewReader(data)
	_, err := ParseClass(r)
	if err == nil {
		t.Error("ParseClass() expected error for invalid magic")
	}
}

func TestParseClass_ShortRead(t *testing.T) {
	data := []byte{0xCA, 0xFE, 0xBA, 0xBE} // Valid magic, but nothing else
	r := bytes.NewReader(data)
	_, err := ParseClass(r)
	if err == nil {
		t.Error("ParseClass() expected error for short read")
	}
}

// Helper to create a minimal valid class file buffer
func createMinimalClassFile() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, uint32(0xCAFEBABE)) // Magic
	binary.Write(buf, binary.BigEndian, uint16(0))          // Minor
	binary.Write(buf, binary.BigEndian, uint16(52))         // Major (Java 8)
	binary.Write(buf, binary.BigEndian, uint16(2))          // ConstantPoolCount (1 entry + 1 placeholder)

	// Constant Pool Entry 1: UTF8 "Test"
	binary.Write(buf, binary.BigEndian, uint8(ConstantKindUtf8))
	str := "Test"
	binary.Write(buf, binary.BigEndian, uint16(len(str)))
	buf.WriteString(str)

	binary.Write(buf, binary.BigEndian, uint16(0)) // AccessFlags
	binary.Write(buf, binary.BigEndian, uint16(0)) // ThisClass
	// ... rest of the file is not read by ParseClass currently (it stops after ThisClass)
	// Actually ParseClass reads AccessFlags and ThisClass.
	// It does NOT read interfaces, fields, methods, attributes.

	return buf.Bytes()
}

func TestParseClass_Minimal(t *testing.T) {
	data := createMinimalClassFile()
	r := bytes.NewReader(data)
	cf, err := ParseClass(r)
	if err != nil {
		t.Fatalf("ParseClass() error = %v", err)
	}

	if cf.Magic != 0xCAFEBABE {
		t.Errorf("Magic = %x, want CAFEBABE", cf.Magic)
	}
	if cf.ConstantPoolCount != 2 {
		t.Errorf("ConstantPoolCount = %d, want 2", cf.ConstantPoolCount)
	}
	if len(cf.ConstantPool) != 2 { // 0-th placeholder + 1 entry
		t.Errorf("len(ConstantPool) = %d, want 2", len(cf.ConstantPool))
	}
	
	cp1 := cf.ConstantPool[1]
	if cp1.Type() != ConstantKindUtf8 {
		t.Errorf("ConstantPool[1].Type() = %v, want %v", cp1.Type(), ConstantKindUtf8)
	}
	utf8Cp := cp1.(*ConstantUtf8)
	if string(utf8Cp.Bytes) != "Test" {
		t.Errorf("ConstantPool[1] = %s, want Test", string(utf8Cp.Bytes))
	}
}

func TestClassFile_ConstantPoolUtf8(t *testing.T) {
	cf := &ClassFile{
		ConstantPool: []ConstantPoolInfo{
			&ConstantPlaceholder{},
			&ConstantUtf8{Length: 4, Bytes: []byte("Test")},
			&ConstantClass{NameIndex: 1},
		},
	}

	got, err := cf.ConstantPoolUtf8(1)
	if err != nil {
		t.Errorf("ConstantPoolUtf8(1) error = %v", err)
	}
	if got != "Test" {
		t.Errorf("ConstantPoolUtf8(1) = %q, want %q", got, "Test")
	}

	_, err = cf.ConstantPoolUtf8(2) // Not Utf8
	if err == nil {
		t.Error("ConstantPoolUtf8(2) expected error for non-Utf8 constant")
	}

	_, err = cf.ConstantPoolUtf8(3) // Out of bounds
	if err == nil {
		t.Error("ConstantPoolUtf8(3) expected error for out of bounds")
	}
}

func TestClassFile_ConstantPoolClass(t *testing.T) {
	cf := &ClassFile{
		ConstantPool: []ConstantPoolInfo{
			&ConstantPlaceholder{},
			&ConstantUtf8{Length: 4, Bytes: []byte("Test")},
			&ConstantClass{NameIndex: 1},
		},
	}

	got, err := cf.ConstantPoolClass(2)
	if err != nil {
		t.Errorf("ConstantPoolClass(2) error = %v", err)
	}
	if got != "Test" {
		t.Errorf("ConstantPoolClass(2) = %q, want %q", got, "Test")
	}

	_, err = cf.ConstantPoolClass(1) // Not Class
	if err == nil {
		t.Error("ConstantPoolClass(1) expected error for non-Class constant")
	}
}

func TestClassFile_ConstantPoolMethodref(t *testing.T) {
	cf := &ClassFile{
		ConstantPool: []ConstantPoolInfo{
			&ConstantPlaceholder{},
			&ConstantUtf8{Length: 9, Bytes: []byte("TestClass")}, // 1
			&ConstantClass{NameIndex: 1},                         // 2
			&ConstantUtf8{Length: 10, Bytes: []byte("methodName")}, // 3
			&ConstantUtf8{Length: 3, Bytes: []byte("()V")},       // 4
			&ConstantNameAndType{NameIndex: 3, DescriptorIndex: 4}, // 5
			&ConstantMethodref{ClassIndex: 2, NameAndTypeIndex: 5}, // 6
		},
	}

	class, method, desc, err := cf.ConstantPoolMethodref(6)
	if err != nil {
		t.Errorf("ConstantPoolMethodref(6) error = %v", err)
	}
	if class != "TestClass" {
		t.Errorf("class = %q, want %q", class, "TestClass")
	}
	if method != "methodName" {
		t.Errorf("method = %q, want %q", method, "methodName")
	}
	if desc != "()V" {
		t.Errorf("desc = %q, want %q", desc, "()V")
	}

	_, _, _, err = cf.ConstantPoolMethodref(2) // Not Methodref
	if err == nil {
		t.Error("ConstantPoolMethodref(2) expected error for non-Methodref constant")
	}
}
