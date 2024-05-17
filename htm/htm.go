package htm

import (
	"fmt"
	"log"
	"html"
	"net/url"
	"strings"
)

// contains well-formed HTML fragment
type Result struct {
	pieces []string
}

type Attr string

var voidEl = map[string]bool{"area": true, "base": true, "br": true, "col": true, "command": true, "embed": true, "hr": true, "img": true, "input": true, "keygen": true, "link": true, "meta": true, "source": true, "track": true, "wbr": true}

func NewHTML(cap int) Result {
	return Result{make([]string, 0, cap)}
}

func NewElem(tag string, attr Attr, bodyEl ...Result) Result {
	var r, body Result
	if len(attr) > 0 && string(attr)[0] != ' ' {
		log.Panic("Invalid attribute")
	}
	switch len(bodyEl) {
	case 0:
		if voidEl[tag] || voidEl[strings.ToLower(tag)] {
			return Result{[]string{"<" + tag + string(attr) + "\n>"}}
		}
	case 1:
		body = bodyEl[0]
	default:
		body = Append(body, bodyEl...)
	}

	switch len(body.pieces) {
	case 0:
		r = Result{make([]string, 1, 1)}
	case 1:
		if len(body.pieces[0]) < 256 {
			return Result{[]string{"<" + tag + string(attr) + "\n>" + body.pieces[0] + "</" + tag + ">"}}
		}
		r = Result{[]string{"", body.pieces[0], ""}}
	default:
		r = body
	}
	r.pieces[0] = "<" + tag + string(attr) + "\n>" + r.pieces[0]
	r.pieces[len(r.pieces)-1] += "</" + tag + ">"
	return r
}

var attrEscaper = strings.NewReplacer(`"`, EncodeURIComponent(`"`))

func Prepend(doctype string, html Result) Result {
	if len(html.pieces) > 0 {
		html.pieces[0] = doctype + html.pieces[0]
		return html
	}
	return Result{[]string{doctype}}
}

//NewAttr constructs an attribute portion of an element(tag)
//attr: can be a complete attribute(s) specified as-is, or an attribute name, in which case it must end with =,
//and be followed by attribute value in the next arg, e.g.
//attr("disabled", `rel="icon" id=""`, "href=", "/", "class=", "") => disabled rel="icon" id="" href="/" class=""
func NewAttr(attr ...string) Attr {
	if len(attr) == 0 {
		return ""
	}
	sar := make([]string, 0, len(attr)*5/2)
	expectAttrVal := false
	for _, v := range attr {
		if expectAttrVal {
			expectAttrVal = false
			if strings.Index(v, `"`) >= 0 {
				v = attrEscaper.Replace(v)
			}
			sar = append(sar, v, `"`)
		} else {
			if len(v) == 0 {
				continue
			}
			if v[0] != ' ' {
				sar = append(sar, " ")
			}
			if v[len(v)-1] == '=' {
				expectAttrVal = true
				sar = append(sar, v, `"`)
			} else {
				sar = append(sar, v)
			}
		}
	}
	if expectAttrVal {
		log.Panic("Expecting attribute value")
	}
	return Attr(strings.Join(sar, ""))
}

func See(what ...any) string {
	return Map(what, func(i int) Result {
		return Result{[]string{fmt.Sprintf("%+v\n", what[i])}}
	}).String()
}

func Map[T any](a []T, f func(int) Result) Result {
	r := NewHTML(len(a))
	for i := range a {
		r = Append(r, f(i))
	}
	return r
}

func If[T ~string | Result](cond bool, result T) T {
	if cond {
		return result
	}
	var r T
	return r
}

func IfCall[T ~string | Result](cond bool, call func() T) T {
	if cond {
		return call()
	}
	var r T
	return r
}

func IfElse[T ~string | Result](cond bool, ifR T, elseR T) T {
	if cond {
		return ifR
	}
	return elseR
}

func IfElseCall[T ~string | Result](cond bool, ifCall func() T, elseCall func() T) T {
	if cond {
		return ifCall()
	}
	return elseCall()
}

func Append(collect Result, frags ...Result) Result {
	var n int
	for _, frag := range frags {
		n += len(frag.pieces)
	}
	if cap(collect.pieces) < len(collect.pieces)+n {
		var newPieces []string
		if len(collect.pieces) > n {
			newPieces = make([]string, 0, len(collect.pieces)*2)
		} else {
			newPieces = make([]string, 0, len(collect.pieces)+n)
		}
		collect.pieces = append(newPieces, collect.pieces...)
	}

	for _, frag := range frags {
		collect.pieces = append(collect.pieces, frag.pieces...)
	}
	return collect
}

func (c Result) IsEmpty() bool {
	return len(c.pieces) == 0
}

func (c Result) String() string {
	return strings.Join(c.pieces, "")
}

func AsIs(a ...string) Result {
	return Result{a}
}

// Used to output HTML text, encoding HTML reserved characters <>&"
func Text(a string) Result {
	return Result{[]string{html.EscapeString(a)}}
}

var EncodeURIComponent = url.QueryEscape

var jsStringEscaper = strings.NewReplacer(
	`"`, `\"`,
	`'`, `\'`,
	"`", "\\`",
	`\`, `\\`,
)

func JSStringEscape(a string) Result {
	return Result{[]string{jsStringEscaper.Replace(a)}}
}
