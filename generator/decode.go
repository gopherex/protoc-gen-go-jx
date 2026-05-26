package generator

import "google.golang.org/protobuf/compiler/protogen"

func genDecode(g *protogen.GeneratedFile, m *protogen.Message) {
	jxDec := jxPkg.Ident("Decoder")
	g.P("func (m *", m.GoIdent, ") Decode(d *", jxDec, ") error {")
	g.P("return d.Obj(func(d *", jxDec, ", key string) error {")
	g.P("switch key {")
	// cases added in later tasks
	g.P("default:")
	g.P(`return `, errorf(g), `("unknown field %q", key)`)
	g.P("}")
	g.P("})")
	g.P("}")
	g.P()
}

// errorf returns the qualified fmt.Errorf ident for use in g.P.
func errorf(g *protogen.GeneratedFile) protogen.GoIdent {
	return protogen.GoIdent{GoName: "Errorf", GoImportPath: "fmt"}
}
