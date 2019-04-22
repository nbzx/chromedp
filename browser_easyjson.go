// Code generated by easyjson for marshaling/unmarshaling. DO NOT EDIT.

package chromedp

import (
	json "encoding/json"

	target "github.com/nbzx/cdproto/target"
	easyjson "github.com/mailru/easyjson"
	jlexer "github.com/mailru/easyjson/jlexer"
	jwriter "github.com/mailru/easyjson/jwriter"
)

// suppress unused package warning
var (
	_ *json.RawMessage
	_ *jlexer.Lexer
	_ *jwriter.Writer
	_ easyjson.Marshaler
)

func easyjsonC5ff9ce6DecodeGithubComChromedpChromedp(in *jlexer.Lexer, out *eventReceivedMessageFromTarget) {
	isTopLevel := in.IsStart()
	if in.IsNull() {
		if isTopLevel {
			in.Consumed()
		}
		in.Skip()
		return
	}
	in.Delim('{')
	for !in.IsDelim('}') {
		key := in.UnsafeString()
		in.WantColon()
		if in.IsNull() {
			in.Skip()
			in.WantComma()
			continue
		}
		switch key {
		case "sessionId":
			out.SessionID = target.SessionID(in.String())
		case "message":
			(out.Message).UnmarshalEasyJSON(in)
		default:
			in.SkipRecursive()
		}
		in.WantComma()
	}
	in.Delim('}')
	if isTopLevel {
		in.Consumed()
	}
}

func easyjsonC5ff9ce6EncodeGithubComChromedpChromedp(out *jwriter.Writer, in eventReceivedMessageFromTarget) {
	out.RawByte('{')
	first := true
	_ = first
	{
		const prefix string = ",\"sessionId\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.SessionID))
	}
	{
		const prefix string = ",\"message\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		easyjsonC5ff9ce6EncodeGithubComChromedpChromedp1(out, in.Message)
	}
	out.RawByte('}')
}

// MarshalJSON supports json.Marshaler interface
func (v eventReceivedMessageFromTarget) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{}
	easyjsonC5ff9ce6EncodeGithubComChromedpChromedp(&w, v)
	return w.Buffer.BuildBytes(), w.Error
}

// MarshalEasyJSON supports easyjson.Marshaler interface
func (v eventReceivedMessageFromTarget) MarshalEasyJSON(w *jwriter.Writer) {
	easyjsonC5ff9ce6EncodeGithubComChromedpChromedp(w, v)
}

// UnmarshalJSON supports json.Unmarshaler interface
func (v *eventReceivedMessageFromTarget) UnmarshalJSON(data []byte) error {
	r := jlexer.Lexer{Data: data}
	easyjsonC5ff9ce6DecodeGithubComChromedpChromedp(&r, v)
	return r.Error()
}

// UnmarshalEasyJSON supports easyjson.Unmarshaler interface
func (v *eventReceivedMessageFromTarget) UnmarshalEasyJSON(l *jlexer.Lexer) {
	easyjsonC5ff9ce6DecodeGithubComChromedpChromedp(l, v)
}

func easyjsonC5ff9ce6DecodeGithubComChromedpChromedp1(in *jlexer.Lexer, out *messageString) {
	isTopLevel := in.IsStart()
	if in.IsNull() {
		if isTopLevel {
			in.Consumed()
		}
		in.Skip()
		return
	}
	in.Delim('{')
	for !in.IsDelim('}') {
		key := in.UnsafeString()
		in.WantColon()
		if in.IsNull() {
			in.Skip()
			in.WantComma()
			continue
		}
		switch key {
		case "M":
			(out.M).UnmarshalEasyJSON(in)
		default:
			in.SkipRecursive()
		}
		in.WantComma()
	}
	in.Delim('}')
	if isTopLevel {
		in.Consumed()
	}
}

func easyjsonC5ff9ce6EncodeGithubComChromedpChromedp1(out *jwriter.Writer, in messageString) {
	out.RawByte('{')
	first := true
	_ = first
	{
		const prefix string = ",\"M\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		(in.M).MarshalEasyJSON(out)
	}
	out.RawByte('}')
}
