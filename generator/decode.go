package generator

import "google.golang.org/protobuf/compiler/protogen"

func genDecode(g *protogen.GeneratedFile, m *protogen.Message) {
	jxDec := jxPkg.Ident("Decoder")
	g.P("func (m *", m.GoIdent, ") Decode(d *", jxDec, ") error {")
	g.P("return d.Obj(func(d *", jxDec, ", key string) error {")
	g.P("switch key {")
	for _, f := range m.Fields {
		if f.Oneof != nil && !f.Oneof.Desc.IsSynthetic() {
			continue
		}
		if f.Desc.IsList() || f.Desc.IsMap() {
			continue
		}
		decodeSingularCase(g, f)
	}
	g.P("default:")
	g.P("return ", errorf(g), `("unknown field %q", key)`)
	g.P("}")
	g.P("return nil")
	g.P("})")
	g.P("}")
	g.P()
}

// decodeSingularCase emits `case "<json>":` for one scalar field.
func decodeSingularCase(g *protogen.GeneratedFile, f *protogen.Field) {
	k := classify(f.Desc)
	if k == kindEnum || k == kindMessage || k == kindOther {
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
