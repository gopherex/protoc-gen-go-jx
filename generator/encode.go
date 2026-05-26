package generator

import "google.golang.org/protobuf/compiler/protogen"

func genEncode(g *protogen.GeneratedFile, m *protogen.Message) {
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
		if f.Desc.IsList() || f.Desc.IsMap() {
			continue
		}
		encodeSingular(g, f)
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
func encodeSingular(g *protogen.GeneratedFile, f *protogen.Field) {
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
	default:
		// kindEnum, kindMessage, kindOther: not yet implemented; emit nothing
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
