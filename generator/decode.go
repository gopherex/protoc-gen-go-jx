package generator

import "google.golang.org/protobuf/compiler/protogen"

func genDecode(g *protogen.GeneratedFile, m *protogen.Message, localPath protogen.GoImportPath) {
	jxDec := jxPkg.Ident("Decoder")
	g.P("func (m *", m.GoIdent, ") Decode(d *", jxDec, ") error {")
	g.P("return d.Obj(func(d *", jxDec, ", key string) error {")
	g.P("switch key {")
	for _, f := range m.Fields {
		if f.Oneof != nil && !f.Oneof.Desc.IsSynthetic() {
			continue
		}
		if f.Desc.IsMap() {
			decodeMapCase(g, f, localPath)
			continue
		}
		if f.Desc.IsList() {
			decodeListCase(g, f, localPath)
			continue
		}
		decodeSingularCase(g, f, localPath)
	}
	for _, oo := range m.Oneofs {
		if oo.Desc.IsSynthetic() {
			continue
		}
		for _, f := range oo.Fields {
			decodeOneofCase(g, oo, f)
		}
	}
	g.P("default:")
	g.P("return ", errorf(g), `("unknown field %q", key)`)
	g.P("}")
	g.P("})")
	g.P("}")
	g.P()
}

// decodeSingularCase emits `case "<json>":` for one scalar field.
func decodeSingularCase(g *protogen.GeneratedFile, f *protogen.Field, localPath protogen.GoImportPath) {
	k := classify(f.Desc)
	if k == kindOther {
		return
	}
	if k == kindEnum {
		g.P("case ", strconvQuote(f.Desc.JSONName()), ":")
		emitDecEnum(g, f, "m."+f.GoName, isPointerField(f))
		return
	}
	if k == kindMessage {
		// Skip external message types (WKTs, etc.) that have no generated Decode.
		if f.Message.GoIdent.GoImportPath != localPath {
			return
		}
		g.P("case ", strconvQuote(f.Desc.JSONName()), ":")
		g.P("if d.Next() == ", g.QualifiedGoIdent(jxPkg.Ident("Null")), " { return d.Null() }")
		g.P("m.", f.GoName, " = &", f.Message.GoIdent, "{}")
		g.P("return m.", f.GoName, ".Decode(d)")
		return
	}
	g.P("case ", strconvQuote(f.Desc.JSONName()), ":")
	g.P("if d.Next() == ", g.QualifiedGoIdent(jxPkg.Ident("Null")), " {")
	g.P("return d.Null()")
	g.P("}")
	// isPointerField: proto3 optional scalars backed by a synthetic oneof
	ptr := isPointerField(f)
	dec, _ := decScalarExpr(g, f)
	if k == kindBytes {
		g.P("v, err := ", dec)
		g.P("if err != nil { return err }")
		g.P("m.", f.GoName, " = v")
		g.P("return nil")
		return
	}
	g.P("v, err := ", dec)
	g.P("if err != nil { return err }")
	if ptr {
		g.P("m.", f.GoName, " = &v")
	} else {
		g.P("m.", f.GoName, " = v")
	}
	g.P("return nil")
}

// decodeListCase reads a JSON array into a repeated scalar field.
func decodeListCase(g *protogen.GeneratedFile, f *protogen.Field, localPath protogen.GoImportPath) {
	k := classify(f.Desc)
	if k == kindOther {
		return
	}
	if k == kindEnum {
		g.P("case ", strconvQuote(f.Desc.JSONName()), ":")
		g.P("if d.Next() == ", g.QualifiedGoIdent(jxPkg.Ident("Null")), " { return d.Null() }")
		g.P("return d.Arr(func(d *", g.QualifiedGoIdent(jxPkg.Ident("Decoder")), ") error {")
		emitDecEnumElem(g, f)
		g.P("})")
		return
	}
	if k == kindMessage {
		// Skip external message types (WKTs, etc.) that have no generated Decode.
		if f.Message.GoIdent.GoImportPath != localPath {
			return
		}
		g.P("case ", strconvQuote(f.Desc.JSONName()), ":")
		g.P("if d.Next() == ", g.QualifiedGoIdent(jxPkg.Ident("Null")), " { return d.Null() }")
		g.P("return d.Arr(func(d *", g.QualifiedGoIdent(jxPkg.Ident("Decoder")), ") error {")
		g.P("el := &", f.Message.GoIdent, "{}")
		g.P("if err := el.Decode(d); err != nil { return err }")
		g.P("m.", f.GoName, " = append(m.", f.GoName, ", el)")
		g.P("return nil")
		g.P("})")
		return
	}
	g.P("case ", strconvQuote(f.Desc.JSONName()), ":")
	g.P("if d.Next() == ", g.QualifiedGoIdent(jxPkg.Ident("Null")), " { return d.Null() }")
	g.P("return d.Arr(func(d *", g.QualifiedGoIdent(jxPkg.Ident("Decoder")), ") error {")
	dec, _ := decScalarExpr(g, f)
	g.P("v, err := ", dec)
	g.P("if err != nil { return err }")
	g.P("m.", f.GoName, " = append(m.", f.GoName, ", v)")
	g.P("return nil")
	g.P("})")
}

// decScalarExpr returns the decode call expression and the resulting Go type.
func decScalarExpr(g *protogen.GeneratedFile, f *protogen.Field) (expr, typ string) {
	q := func(name string) string { return g.QualifiedGoIdent(jxpbPkg.Ident(name)) }
	switch classify(f.Desc) {
	case kindInt32:
		return q("DecInt32") + "(d)", "int32"
	case kindUint32:
		return q("DecUint32") + "(d)", "uint32"
	case kindInt64:
		return q("DecInt64") + "(d)", "int64"
	case kindUint64:
		return q("DecUint64") + "(d)", "uint64"
	case kindFloat32:
		return q("DecFloat32") + "(d)", "float32"
	case kindFloat64:
		return q("DecFloat64") + "(d)", "float64"
	case kindBool:
		return "d.Bool()", "bool"
	case kindString:
		return "d.Str()", "string"
	case kindBytes:
		return q("DecBytes") + "(d)", "[]byte"
	}
	return "", ""
}

// errorf returns the qualified fmt.Errorf ident for use in g.P.
func errorf(g *protogen.GeneratedFile) protogen.GoIdent {
	return protogen.GoIdent{GoName: "Errorf", GoImportPath: "fmt"}
}

// emitDecEnum reads an enum from a string name or number into target (a Go
// lvalue). ptr means the target is a *Enum (proto3 optional).
func emitDecEnum(g *protogen.GeneratedFile, f *protogen.Field, target string, ptr bool) {
	enumGo := f.Enum.GoIdent
	valMap := g.QualifiedGoIdent(protogen.GoIdent{GoName: enumGo.GoName + "_value", GoImportPath: enumGo.GoImportPath})
	g.P("switch d.Next() {")
	g.P("case ", g.QualifiedGoIdent(jxPkg.Ident("String")), ":")
	g.P("s, err := d.Str()")
	g.P("if err != nil { return err }")
	g.P("n, ok := ", valMap, "[s]")
	g.P("if !ok { return ", errorf(g), `("unknown enum value %q", s) }`)
	assignEnum(g, enumGo, target, "n", ptr)
	g.P("return nil")
	g.P("case ", g.QualifiedGoIdent(jxPkg.Ident("Number")), ":")
	g.P("n, err := d.Int32()")
	g.P("if err != nil { return err }")
	assignEnum(g, enumGo, target, "n", ptr)
	g.P("return nil")
	g.P("case ", g.QualifiedGoIdent(jxPkg.Ident("Null")), ":")
	g.P("return d.Null()")
	g.P("default:")
	g.P("return ", errorf(g), `("invalid enum token %s", d.Next())`)
	g.P("}")
}

func assignEnum(g *protogen.GeneratedFile, enumGo protogen.GoIdent, target, nExpr string, ptr bool) {
	if ptr {
		g.P("v := ", enumGo, "(", nExpr, ")")
		g.P(target, " = &v")
	} else {
		g.P(target, " = ", enumGo, "(", nExpr, ")")
	}
}

// decodeOneofCase reads one oneof member and assigns the wrapper struct.
func decodeOneofCase(g *protogen.GeneratedFile, oo *protogen.Oneof, f *protogen.Field) {
	g.P("case ", strconvQuote(f.Desc.JSONName()), ":")
	g.P("if d.Next() == ", g.QualifiedGoIdent(jxPkg.Ident("Null")), " { return d.Null() }")
	switch classify(f.Desc) {
	case kindMessage:
		g.P("w := &", f.GoIdent, "{}")
		g.P("w.", f.GoName, " = &", f.Message.GoIdent, "{}")
		g.P("if err := w.", f.GoName, ".Decode(d); err != nil { return err }")
		g.P("m.", oo.GoName, " = w")
		g.P("return nil")
	case kindEnum:
		g.P("w := &", f.GoIdent, "{}")
		emitDecEnumValueInto(g, f, "w."+f.GoName)
		g.P("m.", oo.GoName, " = w")
		g.P("return nil")
	default:
		dec, _ := decScalarExpr(g, f)
		g.P("val, err := ", dec)
		g.P("if err != nil { return err }")
		g.P("m.", oo.GoName, " = &", f.GoIdent, "{", f.GoName, ": val}")
		g.P("return nil")
	}
}

// emitDecEnumElem reads one enum element (string name or number) and appends it
// to the repeated field m.<GoName>.
func emitDecEnumElem(g *protogen.GeneratedFile, f *protogen.Field) {
	enumGo := f.Enum.GoIdent
	valMap := g.QualifiedGoIdent(protogen.GoIdent{GoName: enumGo.GoName + "_value", GoImportPath: enumGo.GoImportPath})
	g.P("var n int32")
	g.P("switch d.Next() {")
	g.P("case ", g.QualifiedGoIdent(jxPkg.Ident("String")), ":")
	g.P("s, err := d.Str()")
	g.P("if err != nil { return err }")
	g.P("v, ok := ", valMap, "[s]")
	g.P("if !ok { return ", errorf(g), `("unknown enum value %q", s) }`)
	g.P("n = v")
	g.P("default:")
	g.P("v, err := d.Int32()")
	g.P("if err != nil { return err }")
	g.P("n = v")
	g.P("}")
	g.P("m.", f.GoName, " = append(m.", f.GoName, ", ", enumGo, "(n))")
	g.P("return nil")
}
