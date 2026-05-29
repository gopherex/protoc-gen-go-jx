package generator

import "google.golang.org/protobuf/compiler/protogen"

// wktCodec maps a well-known-type full name to its jxpb codec base name
// (e.g. "Timestamp" => jxpb.EncTimestamp / jxpb.DecTimestamp).
// Struct/Value/ListValue/Any are added in later tasks.
var wktCodec = map[string]string{
	"google.protobuf.Timestamp":   "Timestamp",
	"google.protobuf.Duration":    "Duration",
	"google.protobuf.Any":         "Any",
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
	"google.protobuf.Struct":      "Struct",
	"google.protobuf.Value":       "Value",
	"google.protobuf.ListValue":   "ListValue",
}

// wktName returns the jxpb codec base name for a message field, or "".
func wktName(f *protogen.Field) string {
	if f.Message == nil {
		return ""
	}
	return wktCodec[string(f.Message.Desc.FullName())]
}

// hasGeneratedJx reports whether the field's message type is in the same Go
// package as the message owning the field. Same-package types are generated in
// this run and thus have jx Encode/Decode methods, so we can call them directly.
// Cross-package types may live in a proto without generated jx code (outside our
// jurisdiction); those route through jxpb.{Enc,Dec}Message, which detects the
// generated codec at runtime and falls back to protojson when it is absent.
func hasGeneratedJx(f *protogen.Field) bool {
	if f.Parent == nil || f.Message == nil {
		return false
	}
	return f.Parent.GoIdent.GoImportPath == f.Message.GoIdent.GoImportPath
}

// emitEncMsgValue emits the encode call for a message value expression `expr`.
func emitEncMsgValue(g *protogen.GeneratedFile, f *protogen.Field, expr string) {
	switch {
	case wktName(f) != "":
		g.P(g.QualifiedGoIdent(jxpbPkg.Ident("Enc"+wktName(f))), "(e, ", expr, ")")
	case hasGeneratedJx(f):
		g.P(expr, ".Encode(e)")
	default:
		g.P(g.QualifiedGoIdent(jxpbPkg.Ident("EncMessage")), "(e, ", expr, ")")
	}
}

// emitDecMsgValue emits decode of an already-allocated message pointer `target`.
func emitDecMsgValue(g *protogen.GeneratedFile, f *protogen.Field, target string) {
	switch {
	case wktName(f) != "":
		g.P("if err := ", g.QualifiedGoIdent(jxpbPkg.Ident("Dec"+wktName(f))), "(d, ", target, "); err != nil { return err }")
	case hasGeneratedJx(f):
		g.P("if err := ", target, ".Decode(d); err != nil { return err }")
	default:
		g.P("if err := ", g.QualifiedGoIdent(jxpbPkg.Ident("DecMessage")), "(d, ", target, "); err != nil { return err }")
	}
}
