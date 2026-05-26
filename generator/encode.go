package generator

import "google.golang.org/protobuf/compiler/protogen"

func genEncode(g *protogen.GeneratedFile, m *protogen.Message) {
	jxEnc := jxPkg.Ident("Encoder")
	g.P("func (m *", m.GoIdent, ") Encode(e *", jxEnc, ") {")
	g.P("if m == nil {")
	g.P("e.ObjStart()")
	g.P("e.ObjEnd()")
	g.P("return")
	g.P("}")
	g.P("e.ObjStart()")
	// field emission added in later tasks
	g.P("e.ObjEnd()")
	g.P("}")
	g.P()
}
