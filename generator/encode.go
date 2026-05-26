package generator

import "google.golang.org/protobuf/compiler/protogen"

func genEncode(g *protogen.GeneratedFile, m *protogen.Message, localPath protogen.GoImportPath) {
	g.P("func (m *", m.GoIdent, ") Encode(e *", jxPkg.Ident("Encoder"), ") {")
	g.P("if m == nil {")
	g.P("e.ObjStart()")
	g.P("e.ObjEnd()")
	g.P("return")
	g.P("}")
	g.P("e.ObjStart()")
	for _, f := range m.Fields {
		if f.Oneof != nil && !f.Oneof.Desc.IsSynthetic() {
			continue
		}
		if f.Desc.IsMap() {
			encodeMap(g, f, localPath)
			continue
		}
		if f.Desc.IsList() {
			encodeList(g, f, localPath)
			continue
		}
		encodeSingular(g, f, localPath)
	}
	g.P("e.ObjEnd()")
	g.P("}")
	g.P()
}

// isPointerField reports whether the Go struct field is a pointer.
// This is true for proto3 optional scalars (synthetic oneof) and message fields.
func isPointerField(f *protogen.Field) bool {
	return f.Oneof != nil && f.Oneof.Desc.IsSynthetic()
}

// encodeSingular emits encode logic for one non-list, non-map field.
func encodeSingular(g *protogen.GeneratedFile, f *protogen.Field, localPath protogen.GoImportPath) {
	name := f.Desc.JSONName()
	get := "m." + f.GoName
	ptr := isPointerField(f)

	switch classify(f.Desc) {
	case kindInt32, kindUint32, kindFloat32, kindFloat64, kindBool:
		if ptr {
			g.P("if ", get, " != nil {")
			g.P("e.FieldStart(", strconvQuote(name), ")")
			emitEncScalarCall(g, f, "*"+get)
			g.P("}")
		} else {
			g.P("if ", get, " != ", zeroLit(f), " {")
			g.P("e.FieldStart(", strconvQuote(name), ")")
			emitEncScalarCall(g, f, get)
			g.P("}")
		}
	case kindInt64, kindUint64:
		if ptr {
			g.P("if ", get, " != nil {")
			g.P("e.FieldStart(", strconvQuote(name), ")")
			emitEncScalarCall(g, f, "*"+get)
			g.P("}")
		} else {
			g.P("if ", get, " != 0 {")
			g.P("e.FieldStart(", strconvQuote(name), ")")
			emitEncScalarCall(g, f, get)
			g.P("}")
		}
	case kindString:
		if ptr {
			g.P("if ", get, " != nil {")
			g.P("e.FieldStart(", strconvQuote(name), ")")
			g.P("e.Str(*", get, ")")
			g.P("}")
		} else {
			g.P("if ", get, ` != "" {`)
			g.P("e.FieldStart(", strconvQuote(name), ")")
			g.P("e.Str(", get, ")")
			g.P("}")
		}
	case kindBytes:
		// bytes is always a slice, never a pointer; use len check
		g.P("if len(", get, ") > 0 {")
		g.P("e.FieldStart(", strconvQuote(name), ")")
		g.P(g.QualifiedGoIdent(jxpbPkg.Ident("EncBytes")), "(e, ", get, ")")
		g.P("}")
	case kindEnum:
		if ptr {
			g.P("if ", get, " != nil {")
			g.P("e.FieldStart(", strconvQuote(name), ")")
			emitEncEnum(g, f, "*"+get)
			g.P("}")
		} else {
			g.P("if ", get, " != 0 {")
			g.P("e.FieldStart(", strconvQuote(name), ")")
			emitEncEnum(g, f, get)
			g.P("}")
		}
	case kindMessage:
		// message field is always a pointer; emit when non-nil.
		// Skip external messages (WKTs, etc.) that have no generated Encode.
		if f.Message.GoIdent.GoImportPath != localPath {
			break
		}
		g.P("if ", get, " != nil {")
		g.P("e.FieldStart(", strconvQuote(name), ")")
		g.P(get, ".Encode(e)")
		g.P("}")
	default:
		// kindOther: emit nothing
	}
}

// emitEncEnum writes an enum value as its string name, or its number when the
// number has no registered name. Reuses the <Enum>_name map from the .pb.go.
func emitEncEnum(g *protogen.GeneratedFile, f *protogen.Field, val string) {
	enumGo := f.Enum.GoIdent
	nameMap := g.QualifiedGoIdent(protogen.GoIdent{GoName: enumGo.GoName + "_name", GoImportPath: enumGo.GoImportPath})
	g.P("if s, ok := ", nameMap, "[int32(", val, ")]; ok {")
	g.P("e.Str(s)")
	g.P("} else {")
	g.P("e.Int32(int32(", val, "))")
	g.P("}")
}

// encodeList emits an array for a repeated field, omitted when empty.
func encodeList(g *protogen.GeneratedFile, f *protogen.Field, localPath protogen.GoImportPath) {
	k := classify(f.Desc)
	if k == kindOther {
		return
	}
	// Skip external message types (WKTs, etc.) that have no generated Encode.
	if k == kindMessage && f.Message.GoIdent.GoImportPath != localPath {
		return
	}
	get := "m." + f.GoName
	g.P("if len(", get, ") > 0 {")
	g.P("e.FieldStart(", strconvQuote(f.Desc.JSONName()), ")")
	g.P("e.ArrStart()")
	g.P("for _, v := range ", get, " {")
	emitEncElem(g, f, "v")
	g.P("}")
	g.P("e.ArrEnd()")
	g.P("}")
}

// emitEncElem writes one array/map element value. Scalars only in this task;
// enum/message added in a later task.
func emitEncElem(g *protogen.GeneratedFile, f *protogen.Field, v string) {
	switch classify(f.Desc) {
	case kindInt32:
		g.P("e.Int32(", v, ")")
	case kindUint32:
		g.P("e.UInt32(", v, ")")
	case kindInt64:
		g.P(g.QualifiedGoIdent(jxpbPkg.Ident("EncInt64")), "(e, ", v, ")")
	case kindUint64:
		g.P(g.QualifiedGoIdent(jxpbPkg.Ident("EncUint64")), "(e, ", v, ")")
	case kindFloat32:
		g.P(g.QualifiedGoIdent(jxpbPkg.Ident("EncFloat32")), "(e, ", v, ")")
	case kindFloat64:
		g.P(g.QualifiedGoIdent(jxpbPkg.Ident("EncFloat64")), "(e, ", v, ")")
	case kindBool:
		g.P("e.Bool(", v, ")")
	case kindString:
		g.P("e.Str(", v, ")")
	case kindBytes:
		g.P(g.QualifiedGoIdent(jxpbPkg.Ident("EncBytes")), "(e, ", v, ")")
	case kindEnum:
		emitEncEnum(g, f, v)
	case kindMessage:
		g.P(v, ".Encode(e)")
	}
}

// emitEncScalarCall writes the jx/jxpb call that encodes one numeric/bool value.
func emitEncScalarCall(g *protogen.GeneratedFile, f *protogen.Field, val string) {
	switch classify(f.Desc) {
	case kindInt32:
		g.P("e.Int32(", val, ")")
	case kindUint32:
		g.P("e.UInt32(", val, ")")
	case kindInt64:
		g.P(g.QualifiedGoIdent(jxpbPkg.Ident("EncInt64")), "(e, ", val, ")")
	case kindUint64:
		g.P(g.QualifiedGoIdent(jxpbPkg.Ident("EncUint64")), "(e, ", val, ")")
	case kindFloat32:
		g.P(g.QualifiedGoIdent(jxpbPkg.Ident("EncFloat32")), "(e, ", val, ")")
	case kindFloat64:
		g.P(g.QualifiedGoIdent(jxpbPkg.Ident("EncFloat64")), "(e, ", val, ")")
	case kindBool:
		g.P("e.Bool(", val, ")")
	}
}
