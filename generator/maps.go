package generator

import (
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// encodeMap emits a JSON object for a map field, omitted when empty. Map keys
// are always rendered as JSON strings.
func encodeMap(g *protogen.GeneratedFile, f *protogen.Field) {
	val := f.Message.Fields[1]
	get := "m." + f.GoName
	g.P("if len(", get, ") > 0 {")
	g.P("e.FieldStart(", strconvQuote(f.Desc.JSONName()), ")")
	g.P("e.ObjStart()")
	g.P("for k, v := range ", get, " {")
	g.P("e.FieldStart(", mapKeyToString(g, f.Desc.MapKey(), "k"), ")")
	emitEncElem(g, val, "v")
	g.P("}")
	g.P("e.ObjEnd()")
	g.P("}")
}

// mapKeyToString returns a Go string expression for a map-key variable k.
func mapKeyToString(g *protogen.GeneratedFile, fd protoreflect.FieldDescriptor, k string) string {
	sc := func(fn string) string {
		return g.QualifiedGoIdent(protogen.GoIdent{GoName: fn, GoImportPath: "strconv"})
	}
	switch fd.Kind() {
	case protoreflect.StringKind:
		return k
	case protoreflect.BoolKind:
		return sc("FormatBool") + "(" + k + ")"
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return sc("FormatInt") + "(" + k + ", 10)"
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return sc("FormatUint") + "(" + k + ", 10)"
	default: // 32-bit integer keys
		return sc("FormatInt") + "(int64(" + k + "), 10)"
	}
}

// decodeMapCase reads a JSON object into a map field.
func decodeMapCase(g *protogen.GeneratedFile, f *protogen.Field) {
	val := f.Message.Fields[1]
	jxDec := g.QualifiedGoIdent(jxPkg.Ident("Decoder"))
	g.P("case ", strconvQuote(f.Desc.JSONName()), ":")
	g.P("if d.Next() == ", g.QualifiedGoIdent(jxPkg.Ident("Null")), " { return d.Null() }")
	g.P("if m.", f.GoName, " == nil {")
	g.P("m.", f.GoName, " = make(", mapGoType(g, f), ")")
	g.P("}")
	g.P("return d.Obj(func(d *", jxDec, ", ks string) error {")
	emitParseMapKey(g, f.Desc.MapKey())
	g.P("var mv ", elemGoType(g, val))
	emitDecElemInto(g, val, "mv")
	g.P("m.", f.GoName, "[mk] = mv")
	g.P("return nil")
	g.P("})")
}

// mapGoType returns the Go map type literal, e.g. "map[string]int64".
func mapGoType(g *protogen.GeneratedFile, f *protogen.Field) string {
	return "map[" + keyGoType(f.Desc.MapKey()) + "]" + elemGoType(g, f.Message.Fields[1])
}

func keyGoType(fd protoreflect.FieldDescriptor) string {
	switch fd.Kind() {
	case protoreflect.StringKind:
		return "string"
	case protoreflect.BoolKind:
		return "bool"
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		return "int32"
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		return "uint32"
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return "int64"
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return "uint64"
	}
	panic("bad map key")
}

// elemGoType returns the Go type of a map value / list element field.
func elemGoType(g *protogen.GeneratedFile, f *protogen.Field) string {
	switch classify(f.Desc) {
	case kindInt32:
		return "int32"
	case kindUint32:
		return "uint32"
	case kindInt64:
		return "int64"
	case kindUint64:
		return "uint64"
	case kindFloat32:
		return "float32"
	case kindFloat64:
		return "float64"
	case kindBool:
		return "bool"
	case kindString:
		return "string"
	case kindBytes:
		return "[]byte"
	case kindEnum:
		return g.QualifiedGoIdent(f.Enum.GoIdent)
	case kindMessage:
		return "*" + g.QualifiedGoIdent(f.Message.GoIdent)
	}
	panic("bad elem")
}

// emitParseMapKey declares `mk` (the typed map key) from the JSON-string key `ks`.
func emitParseMapKey(g *protogen.GeneratedFile, fd protoreflect.FieldDescriptor) {
	sc := func(fn string) string {
		return g.QualifiedGoIdent(protogen.GoIdent{GoName: fn, GoImportPath: "strconv"})
	}
	switch fd.Kind() {
	case protoreflect.StringKind:
		g.P("mk := ks")
	case protoreflect.BoolKind:
		g.P("mk, err := ", sc("ParseBool"), "(ks)")
		g.P("if err != nil { return err }")
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		g.P("mk64, err := ", sc("ParseInt"), "(ks, 10, 32)")
		g.P("if err != nil { return err }")
		g.P("mk := int32(mk64)")
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		g.P("mk64, err := ", sc("ParseUint"), "(ks, 10, 32)")
		g.P("if err != nil { return err }")
		g.P("mk := uint32(mk64)")
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		g.P("mk, err := ", sc("ParseInt"), "(ks, 10, 64)")
		g.P("if err != nil { return err }")
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		g.P("mk, err := ", sc("ParseUint"), "(ks, 10, 64)")
		g.P("if err != nil { return err }")
	}
}

// emitDecElemInto sets the already-declared lvalue `target` from the decoder.
func emitDecElemInto(g *protogen.GeneratedFile, f *protogen.Field, target string) {
	switch classify(f.Desc) {
	case kindMessage:
		g.P(target, " = &", f.Message.GoIdent, "{}")
		emitDecMsgValue(g, f, target)
	case kindEnum:
		emitDecEnumValueInto(g, f, target)
	default:
		dec, _ := decScalarExpr(g, f)
		g.P("tv, err := ", dec)
		g.P("if err != nil { return err }")
		g.P(target, " = tv")
	}
}

// emitDecEnumValueInto sets a non-pointer enum lvalue from a string name or number.
func emitDecEnumValueInto(g *protogen.GeneratedFile, f *protogen.Field, target string) {
	enumGo := f.Enum.GoIdent
	valMap := g.QualifiedGoIdent(protogen.GoIdent{GoName: enumGo.GoName + "_value", GoImportPath: enumGo.GoImportPath})
	g.P("switch d.Next() {")
	g.P("case ", g.QualifiedGoIdent(jxPkg.Ident("String")), ":")
	g.P("s, err := d.Str()")
	g.P("if err != nil { return err }")
	g.P("n, ok := ", valMap, "[s]")
	g.P("if !ok { return ", errorf(g), `("unknown enum value %q", s) }`)
	g.P(target, " = ", enumGo, "(n)")
	g.P("default:")
	g.P("n, err := d.Int32()")
	g.P("if err != nil { return err }")
	g.P(target, " = ", enumGo, "(n)")
	g.P("}")
}
