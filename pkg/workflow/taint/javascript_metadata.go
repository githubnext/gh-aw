package taint

import "strings"

type jsValueKind uint8

const (
	jsUnknown jsValueKind = iota
	jsString
	jsNumber
	jsArray
	jsBuffer
	jsRegExp
)

type jsLineSafety uint8

const (
	jsNoCR jsLineSafety = 1 << iota
	jsNoLF
	jsTerminatedLine
	jsSingleLine = jsNoCR | jsNoLF
)

type jsMaskState uint8

const (
	jsNotMask jsMaskState = iota
	jsOpenMask
	jsEndedMask
)

func jsStringLiteral(text string) jsValue {
	v := jsValue{kind: jsString, literal: true, text: text, lines: jsSingleLine}
	if strings.ContainsRune(text, '\r') {
		v.lines &^= jsNoCR
	}
	if strings.ContainsRune(text, '\n') {
		v.lines &^= jsNoLF
	}
	if body := strings.TrimSuffix(strings.TrimSuffix(text, "\n"), "\r"); body != text && !strings.ContainsAny(body, "\r\n") {
		v.lines |= jsTerminatedLine
	}
	if payload, ok := strings.CutPrefix(text, "::add-mask::"); ok {
		end := strings.TrimSuffix(strings.TrimSuffix(payload, "\n"), "\r")
		if !strings.ContainsAny(end, "\r\n") {
			v.mask = jsOpenMask
			if end != payload {
				v.mask = jsEndedMask
			}
		}
	}
	return v
}

func jsStringCoercion(v jsValue) jsValue {
	if v.kind == jsString {
		return v
	}
	v.sources = v.taint()
	v.ref, v.fn, v.fields, v.literal = "", nil, nil, false
	v.mask, v.lines = jsNotMask, 0
	if v.kind == jsNumber {
		v.lines = jsSingleLine
	}
	v.kind = jsString
	return v
}

func jsConcatStrings(left, right jsValue) jsValue {
	left, right = jsStringCoercion(left), jsStringCoercion(right)
	sources := union(left.taint(), right.taint())
	if left.literal && right.literal {
		v := jsStringLiteral(left.text + right.text)
		v.sources = sources
		return v
	}
	v := jsValue{kind: jsString, sources: sources, lines: left.lines & right.lines & jsSingleLine}
	if left.lines&jsSingleLine == jsSingleLine && right.lines&jsTerminatedLine != 0 ||
		left.lines&jsTerminatedLine != 0 && right.literal && right.text == "" {
		v.lines |= jsTerminatedLine
	}
	switch {
	case left.literal && left.text == "":
		v.mask = right.mask
	case right.literal && right.text == "":
		v.mask = left.mask
	case left.mask == jsOpenMask && right.lines&jsSingleLine == jsSingleLine:
		v.mask = jsOpenMask
	case left.mask == jsOpenMask && right.lines&jsTerminatedLine != 0:
		v.mask = jsEndedMask
	}
	return v
}

func (j *javascript) template(n *JavaScriptNode) jsValue {
	v := jsStringLiteral("")
	if n.Kind == "TemplateSpan" {
		v = jsStringCoercion(j.eval(n.Expression))
	}
	for _, child := range n.Other {
		v = jsConcatStrings(v, j.eval(child))
	}
	for _, span := range n.TemplateSpans {
		v = jsConcatStrings(v, j.eval(span))
	}
	return v
}

func jsReplacementLines(base jsValue, args []jsValue, method string) jsLineSafety {
	if len(args) < 2 {
		return base.lines
	}
	pattern, _ := sliceAt(args, 0)
	replacement, _ := sliceAt(args, 1)
	lines := base.lines & replacement.lines & jsSingleLine
	if !replacement.literal || replacement.kind != jsString || strings.ContainsRune(replacement.text, '$') {
		return lines
	}
	var removed jsLineSafety
	if method == "replaceAll" && pattern.kind == jsString && pattern.literal {
		switch pattern.text {
		case "\r":
			removed = jsNoCR
		case "\n":
			removed = jsNoLF
		}
	} else if pattern.kind == jsRegExp {
		// Replacing only the first occurrence cannot prove a single-line
		// workflow command, even when the replacement itself is escaped.
		end := strings.LastIndex(pattern.text, "/")
		if end > 0 && strings.Contains(pattern.text[end+1:], "g") && !strings.ContainsRune(pattern.text[end+1:], 'y') {
			switch pattern.text[1:end] {
			case `\r`:
				removed = jsNoCR
			case `\n`:
				removed = jsNoLF
			case `[\r\n]`, `\r|\n`:
				removed = jsSingleLine
			}
		}
	}
	return lines | removed&replacement.lines
}

func (j *javascript) sequenceMethod(n *JavaScriptNode, base jsValue, args []jsValue, method string) (jsValue, bool) {
	sequence := base.kind == jsString || base.kind == jsArray || base.kind == jsBuffer
	if !sequence && base.kind != jsNumber {
		return jsValue{}, false
	}
	all := jsUnion(append([]jsValue{base}, args...)...)
	switch method {
	case "indexOf", "lastIndexOf":
		if sequence {
			return jsValue{kind: jsNumber}, true
		}
	case "toString":
		v := jsStringCoercion(base)
		v.sources = all.taint()
		return v, true
	case "includes":
		return jsValue{}, sequence
	case "startsWith", "endsWith":
		return jsValue{}, base.kind == jsString
	case "concat":
		if base.kind == jsString {
			for _, arg := range args {
				base = jsConcatStrings(base, arg)
			}
			return base, true
		}
	case "slice":
		all.kind = base.kind
		if base.kind == jsString {
			all.lines = base.lines & jsSingleLine
		}
		return all, sequence
	case "split":
		all.kind = jsArray
		return all, base.kind == jsString
	case "join":
		all.kind = jsString
		return all, base.kind == jsArray
	case "charCodeAt", "codePointAt":
		all.kind = jsNumber
		return all, base.kind == jsString
	default:
		return j.stringMethod(n, base, args, method, all)
	}
	return jsValue{}, false
}

func (j *javascript) stringMethod(n *JavaScriptNode, base jsValue, args []jsValue, method string, all jsValue) (jsValue, bool) {
	switch method {
	case "trim", "trimStart", "trimEnd", "toLowerCase", "toUpperCase", "substring":
		all.kind, all.lines = jsString, base.lines&jsSingleLine
		return all, base.kind == jsString
	case "replace", "replaceAll":
		all.kind, all.lines = jsString, jsReplacementLines(base, args, method)
		for _, arg := range args {
			if arg.fn != nil {
				j.gap(n, "JavaScript replacement callback data flow is not modeled")
			}
		}
		return all, base.kind == jsString
	}
	return jsValue{}, false
}
