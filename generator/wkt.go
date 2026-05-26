package generator

import "google.golang.org/protobuf/compiler/protogen"

// wktCodec maps a well-known-type full name to its jxpb codec base name
// (e.g. "Timestamp" => jxpb.EncTimestamp / jxpb.DecTimestamp).
// Struct/Value/ListValue/Any are added in later tasks.
var wktCodec = map[string]string{
	"google.protobuf.Timestamp":   "Timestamp",
	"google.protobuf.Duration":    "Duration",
	"google.protobuf.Empty":       "Empty",
	"google.protobuf.FieldMask":   "FieldMask",
	"google.protobuf.DoubleValue": "DoubleValue",
	"google.protobuf.FloatValue":  "FloatValue",
	"google.protobuf.Int32Value":  "Int32Value",
	"google.protobuf.UInt32Value": "Uint32Value",
	"google.protobuf.Int64Value":  "Int64Value",
	"google.protobuf.UInt64Value": "Uint64Value",
	"google.protobuf.BoolValue":   "BoolValue",
	"google.protobuf.StringValue": "StringValue",
	"google.protobuf.BytesValue":  "BytesValue",
}

// wktName returns the jxpb codec base name for a message field, or "".
func wktName(f *protogen.Field) string {
	if f.Message == nil {
		return ""
	}
	return wktCodec[string(f.Message.Desc.FullName())]
}

// msgSupported reports whether a message-typed field can be (de)serialized:
// a known WKT, or a message generated in this same package.
func msgSupported(f *protogen.Field, localPath protogen.GoImportPath) bool {
	return wktName(f) != "" || f.Message.GoIdent.GoImportPath == localPath
}

// emitEncMsgValue emits the encode call for a message value expression `expr`.
func emitEncMsgValue(g *protogen.GeneratedFile, f *protogen.Field, expr string) {
	if w := wktName(f); w != "" {
		g.P(g.QualifiedGoIdent(jxpbPkg.Ident("Enc"+w)), "(e, ", expr, ")")
	} else {
		g.P(expr, ".Encode(e)")
	}
}

// emitDecMsgValue emits decode of an already-allocated message pointer `target`.
func emitDecMsgValue(g *protogen.GeneratedFile, f *protogen.Field, target string) {
	if w := wktName(f); w != "" {
		g.P("if err := ", g.QualifiedGoIdent(jxpbPkg.Ident("Dec"+w)), "(d, ", target, "); err != nil { return err }")
	} else {
		g.P("if err := ", target, ".Decode(d); err != nil { return err }")
	}
}
