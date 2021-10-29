package sdlang

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"time"
)

type sdlValueTag int

const (
	tNull sdlValueTag = iota
	tString
	tInt
	tFloat
	tDateTime
	tTimeSpan
	tBool
	tBinary
)

// SdlValue is a tagged union for every possible type representable in SDLang.
type SdlValue struct {
	tag       sdlValueTag
	vString   string
	vInt      int64
	vFloat    float64
	vDateTime time.Time
	vTimeSpan time.Duration
	vBool     bool
	vBinary   []byte
}

// SdlAttribute is a Key-Value pair between a string and an SdlValue
type SdlAttribute struct {
	// Namespace is the namespace of this attribute.
	Namespace string

	// Name is the name of this attribute.
	Name string

	// QualifiedName is the fully qualified ("namespace:name") name of this attribute.
	QualifiedName string

	// Value is the value.
	Value SdlValue
}

// SdlTag is a container consisting of a name; child tags; attributes, and values.
type SdlTag struct {
	// Namespace is the namespace of this tag.
	Namespace string

	// Name is the name of this tag.
	Name string

	// QualifiedName is the fully qualified ("namespace:name") name of this tag.
	QualifiedName string

	// Children contains the children of this tag. It is safe (and expected) to modify this value.
	Children []SdlTag

	// Children contains the attribtues of this tag. It is safe (and expected) to modify this value.
	Attributes map[string]SdlAttribute

	// Children contains the values of this tag. It is safe (and expected) to modify this value.
	Values []SdlValue
}

// Null creates a null SdlValue
func Null() SdlValue {
	return SdlValue{tag: tNull}
}

// String creates a string SdlValue
func String(value string) SdlValue {
	return SdlValue{tag: tString, vString: value}
}

// Int creates an int SdlValue
func Int(value int64) SdlValue {
	return SdlValue{tag: tInt, vInt: value}
}

// Float creates a float SdlValue
func Float(value float64) SdlValue {
	return SdlValue{tag: tFloat, vFloat: value}
}

// DateTime creates a datetime SdlValue
func DateTime(value time.Time) SdlValue {
	return SdlValue{tag: tDateTime, vDateTime: value}
}

// TimeSpan creates a timespan SdlValue
func TimeSpan(value time.Duration) SdlValue {
	return SdlValue{tag: tTimeSpan, vTimeSpan: value}
}

// Bool creates a bool SdlValue
func Bool(value bool) SdlValue {
	return SdlValue{tag: tBool, vBool: value}
}

// Binary creates a binary SdlValue
func Binary(value []byte) SdlValue {
	return SdlValue{tag: tBinary, vBinary: value}
}

func (v SdlValue) IsNull() bool {
	return v.tag == tNull
}
func (v SdlValue) IsString() bool {
	return v.tag == tString
}
func (v SdlValue) IsInt() bool {
	return v.tag == tInt
}
func (v SdlValue) IsFloat() bool {
	return v.tag == tFloat
}
func (v SdlValue) IsDateTime() bool {
	return v.tag == tDateTime
}
func (v SdlValue) IsTimeSpan() bool {
	return v.tag == tTimeSpan
}
func (v SdlValue) IsBool() bool {
	return v.tag == tBool
}
func (v SdlValue) IsBinary() bool {
	return v.tag == tBinary
}

func (v SdlValue) String() (string, error) {
	if !v.IsString() {
		return "", errors.New("this value is not a string")
	}
	return v.vString, nil
}
func (v SdlValue) Int() (int64, error) {
	if !v.IsInt() {
		return 0, errors.New("this value is not an integer")
	}
	return v.vInt, nil
}
func (v SdlValue) Float() (float64, error) {
	if !v.IsString() {
		return 0, errors.New("this value is not a float")
	}
	return v.vFloat, nil
}
func (v SdlValue) DateTime() (time.Time, error) {
	if !v.IsDateTime() {
		return time.Now(), errors.New("this value is not a datetime")
	}
	return v.vDateTime, nil
}
func (v SdlValue) TimeSpan() (time.Duration, error) {
	if !v.IsTimeSpan() {
		return 0, errors.New("this value is not a timespan")
	}
	return v.vTimeSpan, nil
}
func (v SdlValue) Bool() (bool, error) {
	if !v.IsBool() {
		return false, errors.New("this value is not a bool")
	}
	return v.vBool, nil
}
func (v SdlValue) Binary() ([]byte, error) {
	if !v.IsBinary() {
		return []byte{}, errors.New("this value is not a binary blob")
	}
	return v.vBinary, nil
}

// ForEachChild applies the function `f` onto each child of the tag.
func (t SdlTag) ForEachChild(f func(child *SdlTag)) {
	for i := 0; i < len(t.Children); i++ {
		f(&t.Children[i])
	}
}

// ForEachChildFiltered applies the function `f` onto each child of the tag that passes the `filter` predicate.
func (t SdlTag) ForEachChildFiltered(filter func(child *SdlTag) bool, f func(child *SdlTag)) {
	t.ForEachChild(func(child *SdlTag) {
		if filter(child) {
			f(child)
		}
	})
}

// ForEachChildByName applies the function `f` onto each child of the tag that has the specified `name`.
func (t SdlTag) ForEachChildByName(name string, f func(child *SdlTag)) {
	t.ForEachChildFiltered(func(child *SdlTag) bool {
		return child.Name == name
	}, f)
}

// ForEachChildByName applies the function `f` onto each child of the tag that has the specified `namespace`.
func (t SdlTag) ForEachChildByNamespace(namespace string, f func(child *SdlTag)) {
	t.ForEachChildFiltered(func(child *SdlTag) bool {
		return child.Namespace == namespace
	}, f)
}

// Using the given SaxParser, an AST is constructed.
// The returned value contains the root tag, which is nameless and only contains children.
func (p SaxParser) ParseIntoAst() (SdlTag, error) {
	var currTagStack []SdlTag
	currTagStack = append(currTagStack, SdlTag{})

	prevWasNewLine := true
	for {
		err := p.Next()
		if err != nil {
			return SdlTag{}, err
		}
		if p.IsEof() {
			break
		}

		if p.IsTagName() {
			if !prevWasNewLine {
				return SdlTag{}, p.NewError(0, "(probably a bug) Tag names can only appear at the start of new lines.")
			}
			var tag SdlTag
			tag.Name = p.Text()
			tag.Namespace = p.AdditionalText()
			tag.QualifiedName = p.AdditionalText() + ":" + p.Text()
			if tag.QualifiedName[0] == ':' {
				tag.QualifiedName = tag.QualifiedName[1:]
			}
			currTagStack = append(currTagStack, tag)
			prevWasNewLine = false
		} else if p.IsAttributeName() {
			var attr SdlAttribute
			attr.Name = p.Text()
			attr.Namespace = p.AdditionalText()
			attr.QualifiedName = p.AdditionalText() + ":" + p.Text()
			if attr.QualifiedName[0] == ':' {
				attr.QualifiedName = attr.QualifiedName[1:]
			}
			err = p.Next()
			if err != nil {
				return SdlTag{}, err
			}
			handleValue(&attr.Value, &p)

			if currTagStack[len(currTagStack)-1].Attributes == nil {
				currTagStack[len(currTagStack)-1].Attributes = map[string]SdlAttribute{}
			}
			currTagStack[len(currTagStack)-1].Attributes[attr.QualifiedName] = attr
			prevWasNewLine = false
		} else if p.IsNewLine() {
			if !prevWasNewLine {
				parent := &currTagStack[len(currTagStack)-2]
				child := currTagStack[len(currTagStack)-1]
				parent.Children = append(parent.Children, child)
				currTagStack = currTagStack[0 : len(currTagStack)-1]
			}
			prevWasNewLine = true
		} else if p.IsOpenTag() {
			if prevWasNewLine {
				return SdlTag{}, p.NewError(0, "Opening braces have to be on the same line as a tag.")
			}

			prevWasNewLine = true
			err = p.Next()
			if err != nil {
				return SdlTag{}, err
			}
			if !p.IsNewLine() {
				return SdlTag{}, p.NewError(0, "Expected a new line following opening brace.")
			}
		} else if p.IsCloseTag() {
			if !prevWasNewLine {
				return SdlTag{}, p.NewError(0, "Closing braces must be on their own line.")
			}

			err = p.Next()
			if err != nil {
				return SdlTag{}, err
			}
			if !p.IsNewLine() && !p.IsEof() {
				return SdlTag{}, p.NewError(0, "Expected a new line or end of file following closing brace.")
			}
			parent := &currTagStack[len(currTagStack)-2]
			child := currTagStack[len(currTagStack)-1]
			parent.Children = append(parent.Children, child)
			currTagStack = currTagStack[0 : len(currTagStack)-1]
			prevWasNewLine = true
		} else {
			var val SdlValue
			handleValue(&val, &p)
			currTagStack[len(currTagStack)-1].Values = append(currTagStack[len(currTagStack)-1].Values, val)
			prevWasNewLine = false
		}
	}

	return currTagStack[0], nil
}

func handleValue(v *SdlValue, p *SaxParser) {
	if p.IsBinary() {
		v.tag = tBinary
		base64.NewDecoder(base64.RawStdEncoding, bytes.NewBufferString(p.Text())).Read(v.vBinary)
	} else if p.IsBool() {
		v.tag = tBool
		v.vBool = p.Bool()
	} else if p.IsDate() || p.IsDateTime() {
		v.tag = tDateTime
		v.vDateTime = p.Time()
	} else if p.IsDouble() || p.IsFloat() {
		v.tag = tFloat
		v.vFloat, _ = strconv.ParseFloat(p.Text(), 64)
	} else if p.IsInteger() || p.IsLong() {
		v.tag = tInt
		v.vInt, _ = strconv.ParseInt(p.Text(), 10, 64)
	} else if p.IsNull() {
		v.tag = tNull
	} else if p.IsString() {
		v.tag = tString
		v.vString = p.Text()
	} else if p.IsTimeSpan() {
		v.tag = tTimeSpan
		v.vTimeSpan = p.TimeSpan()
	} else {
		fmt.Printf("p: %v\n", p.t)
		panic("bug: this error should've been caught earlier on")
	}
}
