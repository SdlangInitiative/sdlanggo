package sdlang

import (
	"errors"
	"strconv"
	"time"

	"github.com/BradleyChatha/decorator"
)

type saxType int

const (
	failsafe saxType = iota
	tagName
	attributeName
	string_
	character
	integer
	long
	float
	double
	decimal
	boolean
	date
	dateTime
	timeSpan
	binary
	null
	newLine
	eof
	openTag
	closeTag
)

// SaxParser provides a SAX-style of parsing.
// This is more efficient than constructing an AST, but involves more effort on the user's side.
type SaxParser struct {
	// Input is the input to parse.
	Input string

	// FileName is used for debug messages.
	FileName string
	cursor   int
	t        saxType
	text     string
	addText  string
	dateTime time.Time
	timeSpan time.Duration
	boolean  bool
}

func (s *SaxParser) IsTagName() bool {
	return s.t == tagName
}
func (s *SaxParser) IsAttributeName() bool {
	return s.t == attributeName
}
func (s *SaxParser) IsString() bool {
	return s.t == string_
}
func (s *SaxParser) IsChar() bool {
	return s.t == character
}
func (s *SaxParser) IsInteger() bool {
	return s.t == integer
}
func (s *SaxParser) IsLong() bool {
	return s.t == long
}
func (s *SaxParser) IsFloat() bool {
	return s.t == float
}
func (s *SaxParser) IsDouble() bool {
	return s.t == double
}
func (s *SaxParser) IsDecimal() bool {
	return s.t == decimal
}
func (s *SaxParser) IsBool() bool {
	return s.t == boolean
}
func (s *SaxParser) IsDate() bool {
	return s.t == date
}
func (s *SaxParser) IsDateTime() bool {
	return s.t == dateTime
}
func (s *SaxParser) IsTimeSpan() bool {
	return s.t == timeSpan
}
func (s *SaxParser) IsBinary() bool {
	return s.t == binary
}
func (s *SaxParser) IsNull() bool {
	return s.t == null
}
func (s *SaxParser) IsEof() bool {
	return s.t == eof
}
func (s *SaxParser) IsNewLine() bool {
	return s.t == newLine
}
func (s *SaxParser) IsOpenTag() bool {
	return s.t == openTag
}
func (s *SaxParser) IsCloseTag() bool {
	return s.t == closeTag
}

// Text is the parsed text.
// For tag/attribute names, this is the non-namespace value.
func (s *SaxParser) Text() string {
	return s.text
}

// Additional text is some additional text to compliment the main text.
// For tag/attribute names, this is the namespace value.
func (s *SaxParser) AdditionalText() string {
	return s.addText
}

// Time is the value for DateTime and Date literals.
func (s *SaxParser) Time() time.Time {
	return s.dateTime
}

// TimeSpan is the value for the TimeSpan literal.
func (s *SaxParser) TimeSpan() time.Duration {
	return s.timeSpan
}

// Bool is the value for the Bool literal.
func (s *SaxParser) Bool() bool {
	return s.boolean
}

func (s *SaxParser) peek(offset int) byte {
	if s.cursor+offset >= len(s.Input) {
		return '\u00ff'
	}
	return s.Input[s.cursor+offset]
}

func (s *SaxParser) advance(amount int) {
	s.cursor += amount
}

func (s *SaxParser) eof() bool {
	return s.cursor >= len(s.Input)
}

func (s *SaxParser) eatWhite() {
	for !s.eof() && (s.peek(0) == ' ' || s.peek(0) == '\t') {
		s.advance(1)
	}
}

func (s *SaxParser) getLine(at int) (string, int, int) {
	start := at
	end := at

	if start != 0 && (start >= len(s.Input) || s.Input[start] == '\n') {
		start--
	}
	for start > 0 && s.Input[start] != '\n' {
		start--
	}
	for end < len(s.Input) && s.Input[end] != '\n' && s.Input[end] != '\r' {
		end++
	}

	if s.Input[start] == '\n' {
		start++
	}

	line := 1
	for i := 0; i < start; i++ {
		if s.Input[i] == '\n' {
			line++
		}
	}

	return s.Input[start:end], at - start, line
}

// Generates a fancy error message.
func (s *SaxParser) NewError(offset int, msg string) error {
	line, loc, ln := s.getLine(s.cursor + offset)
	var d decorator.Decorator
	d.AddLine(line, decorator.LineMetadata{FileName: s.FileName, LineNumber: ln})
	d.AddBottomComment(0, loc, msg)
	return errors.New(d.String())
}

// Next parses the next token.
// You can query which token was parsed via the `IsBool`, `IsString`, `Is..` etc. functions.
// You can use the likes of `Text`, `DateTime`, and so on to retrieve the parsed values.
// You should keep calling this function until either an error is returned, or `IsEof` returns true.
// Error messages are already formatted for a user-friendly experience.
func (s *SaxParser) Next() error {
	s.eatWhite()
	if s.eof() {
		s.t = eof
		return nil
	}

	if (s.peek(0) == '/' && s.peek(1) == '/') || (s.peek(0) == '-' && s.peek(1) == '-') || s.peek(0) == '#' {
		for !s.eof() && s.peek(0) != '\n' {
			s.advance(1)
		}
		return s.Next()
	}

	if s.peek(0) == '\n' {
		s.advance(1)
		s.t = newLine
		return nil
	} else if s.peek(0) == '\r' {
		if s.peek(1) != '\n' {
			line, _, ln := s.getLine(s.cursor)
			var d decorator.Decorator
			d.AddLine(line+" ", decorator.LineMetadata{FileName: s.FileName, LineNumber: ln})
			d.AddBottomComment(0, len(line), "Stray \\r without a \\n following it.")
			return errors.New(d.String())
		}
		s.advance(2)
		s.t = newLine
		return nil
	} else if s.peek(0) == '\\' && s.peek(1) == '\n' {
		s.advance(2)
		return s.Next()
	}

	ch := s.peek(0)
	if isIdentifierStart(ch) {
		err := s.nextIdentifier()
		if err != nil {
			return err
		}
		if s.t == attributeName {
			if s.peek(0) != '=' {
				line, loc, ln := s.getLine(s.cursor)
				var d decorator.Decorator
				d.AddLine(line, decorator.LineMetadata{FileName: s.FileName, LineNumber: ln})
				d.AddBottomComment(0, loc, "Expected '=' following attribute name")
				return errors.New(d.String())
			}
			s.advance(1)
		}
		return nil
	} else if ch == '{' {
		s.t = openTag
		s.text = "{"
		s.advance(1)
		return nil
	} else if ch == '}' {
		s.t = closeTag
		s.text = "}"
		s.advance(1)
		return nil
	}

	if s.t == newLine || s.t == failsafe {
		s.t = tagName
		s.text = "content"
		return nil
	}

	if ch == '"' {
		return s.nextDoubleQuotedString()
	} else if ch == '`' {
		return s.nextBacktickString()
	} else if ch == '[' {
		return s.nextBinary()
	} else if isDigit(ch) || ch == '-' {
		return s.nextNumeric()
	}

	line, loc, ln := s.getLine(s.cursor)
	var d decorator.Decorator
	d.AddLine(line, decorator.LineMetadata{FileName: s.FileName, LineNumber: ln})
	d.AddBottomComment(0, loc, "Unexpected character.")
	return errors.New(d.String())
}

func (s *SaxParser) nextIdentifier() error {
	if s.t == newLine || s.t == failsafe {
		s.t = tagName
	} else {
		s.t = attributeName
	}

	start := s.cursor
	for isIdentifierContinue(s.peek(0)) {
		s.advance(1)
	}
	end := s.cursor

	if s.Input[start:end] == "true" || s.Input[start:end] == "on" {
		s.t = boolean
		s.text = "true"
		s.boolean = true
		return nil
	} else if s.Input[start:end] == "false" || s.Input[start:end] == "off" {
		s.t = boolean
		s.text = "false"
		s.boolean = false
		return nil
	} else if s.Input[start:end] == "null" {
		s.t = null
		s.text = "null"
		return nil
	}

	if s.peek(0) != ':' {
		s.text = s.Input[start:end]
		s.addText = ""
		return nil
	} else {
		s.addText = s.Input[start:end]
	}
	s.advance(1)

	start = s.cursor
	for isIdentifierContinue(s.peek(0)) {
		s.advance(1)
	}
	end = s.cursor
	s.text = s.Input[start:end]

	return nil
}

func (s *SaxParser) nextDoubleQuotedString() error {
	debugStart := s.cursor
	s.advance(1)
	s.t = string_

	text := ""
	start := s.cursor
	for !s.eof() {
		if s.peek(0) == '\\' {
			text += s.Input[start:s.cursor]

			s.advance(1)
			ch := s.peek(0)
			switch ch {
			case 'n':
				text += "\n"
				s.advance(1)
			case 't':
				text += "\t"
				s.advance(1)
			case 'r':
				text += "\r"
				s.advance(1)
			case '"':
				text += "\""
				s.advance(1)
			case '\\':
				text += "\\"
				s.advance(1)
			case '\n':
				s.advance(1)
				for s.peek(0) == ' ' || s.peek(0) == '\t' {
					s.advance(1)
				}
			default:
				line, loc, ln := s.getLine(s.cursor)
				var d decorator.Decorator
				d.AddLine(line, decorator.LineMetadata{FileName: s.FileName, LineNumber: ln})
				d.AddBottomComment(0, loc, "Invalid escape character. Only \\t, \\n, \\r, \\\", and \\\\ are allowed.")
				return errors.New(d.String())
			}

			start = s.cursor
			continue
		} else if s.peek(0) == '"' {
			if text == "" {
				text = s.Input[start:s.cursor]
			} else {
				text += s.Input[start:s.cursor]
			}
			s.advance(1)
			s.text = text
			return nil
		} else if s.peek(0) == '\n' {
			break
		}
		s.advance(1)
	}

	line, loc, ln := s.getLine(debugStart)
	var d decorator.Decorator
	d.AddLine(line, decorator.LineMetadata{FileName: s.FileName, LineNumber: ln})
	d.AddBottomComment(0, loc, "Unterminated string")
	line, loc, ln = s.getLine(s.cursor)
	d.AddLine(line, decorator.LineMetadata{FileName: s.FileName, LineNumber: ln})
	d.AddBottomComment(1, loc, "Expected a terminating '\"' before hitting end of file/line")
	return errors.New(d.String())
}

func (s *SaxParser) nextBacktickString() error {
	debugStart := s.cursor
	s.advance(1)
	s.t = string_

	start := s.cursor
	for !s.eof() {
		if s.peek(0) == '`' {
			s.text = s.Input[start:s.cursor]
			s.advance(1)
			return nil
		} else if s.peek(0) == '\r' {
			line, loc, ln := s.getLine(s.cursor)
			var d decorator.Decorator
			d.AddLine(line, decorator.LineMetadata{FileName: s.FileName, LineNumber: ln})
			d.AddBottomComment(0, loc, "Backtick strings do not support \\r characters")
			return errors.New(d.String())
		}
		s.advance(1)
	}

	line, loc, ln := s.getLine(debugStart)
	var d decorator.Decorator
	d.AddLine(line, decorator.LineMetadata{FileName: s.FileName, LineNumber: ln})
	d.AddBottomComment(0, loc, "Unterminated string")
	line, loc, ln = s.getLine(s.cursor)
	d.AddLine(line, decorator.LineMetadata{FileName: s.FileName, LineNumber: ln})
	d.AddBottomComment(1, loc, "Expected a terminating '`' before hitting end of file")
	return errors.New(d.String())
}

func (s *SaxParser) nextBinary() error {
	debugStart := s.cursor
	s.advance(1)
	s.t = binary

	var text []byte
	for !s.eof() {
		if s.peek(0) == ']' {
			s.text = string(text)
			s.advance(1)
			return nil
		} else if s.peek(0) == ' ' || s.peek(0) == '\t' || s.peek(0) == '\n' || s.peek(0) == '\r' {
			s.advance(1)
			continue
		}
		text = append(text, s.peek(0))
		s.advance(1)
	}

	line, loc, ln := s.getLine(debugStart)
	var d decorator.Decorator
	d.AddLine(line, decorator.LineMetadata{FileName: s.FileName, LineNumber: ln})
	d.AddBottomComment(0, loc, "Unterminated string")
	line, loc, ln = s.getLine(s.cursor)
	d.AddLine(line, decorator.LineMetadata{FileName: s.FileName, LineNumber: ln})
	d.AddBottomComment(1, loc, "Expected a terminating '`' before hitting end of file")
	return errors.New(d.String())
}

func (s *SaxParser) nextNumeric() error {
	if s.peek(4) == '/' {
		if s.peek(13) == ':' && isDigit(s.peek(11)) && isDigit(s.peek(12)) {
			return s.nextDateTime()
		}
		return s.nextDate()
	}

	start := s.cursor
	isNegative := s.peek(0) == '-'
	if isNegative {
		s.advance(1)
	}

	foundDot := false
	for !s.eof() {
		if s.peek(0) == '.' {
			if foundDot {
				line, loc, ln := s.getLine(s.cursor)
				var d decorator.Decorator
				d.AddLine(line, decorator.LineMetadata{LineNumber: ln, FileName: s.FileName})
				d.AddBottomComment(0, loc, "There are multiple decimal places in this number.")
				return errors.New(d.String())
			}
			foundDot = true
		} else if !isDigit(s.peek(0)) {
			break
		}
		s.advance(1)
	}
	num := s.Input[start:s.cursor]

	if s.peek(0) == 'd' || s.peek(0) == ':' {
		return s.nextTimeSpan(num)
	}

	s.t = integer
	if s.peek(0) == 'L' {
		s.t = long
		s.advance(1)
	} else if s.peek(0) == 'F' {
		s.t = float
		s.advance(1)
	} else if s.peek(0) == 'D' {
		s.t = double
		s.advance(1)
	} else if foundDot {
		s.t = double
	}

	if !s.eof() && s.peek(0) != ' ' && s.peek(0) != '\n' && s.peek(0) != '\r' {
		line, loc, ln := s.getLine(s.cursor)
		var d decorator.Decorator
		d.AddLine(line, decorator.LineMetadata{LineNumber: ln, FileName: s.FileName})
		d.AddBottomComment(0, loc, "Expected whitespace or End of line/file after number.")
		return errors.New(d.String())
	}

	s.text = num
	return nil
}

func (s *SaxParser) nextTimeSpan(first string) error {
	var days, hours, minutes, seconds, nsecs string
	isNegative := first[0] == '-'
	if s.peek(0) == 'd' {
		days = first
		if s.peek(1) != ':' {
			line, loc, ln := s.getLine(s.cursor)
			var d decorator.Decorator
			d.AddLine(line, decorator.LineMetadata{LineNumber: ln, FileName: s.FileName})
			d.AddBottomComment(0, loc, "Expected a : following the days component of a TimeSpan.")
			return errors.New(d.String())
		}
		s.advance(2)
	} else {
		// To keep the logic simple, we'll backtrack slightly
		s.cursor -= len(first)
		if isNegative {
			s.cursor += 1
		}
	}

	if s.peek(2) != ':' {
		line, loc, ln := s.getLine(s.cursor + 2)
		var d decorator.Decorator
		d.AddLine(line, decorator.LineMetadata{LineNumber: ln, FileName: s.FileName})
		d.AddBottomComment(0, loc, "Expected a : following the hours component of a TimeSpan.")
		return errors.New(d.String())
	} else if s.peek(5) != ':' {
		line, loc, ln := s.getLine(s.cursor + 5)
		var d decorator.Decorator
		d.AddLine(line, decorator.LineMetadata{LineNumber: ln, FileName: s.FileName})
		d.AddBottomComment(0, loc, "Expected a : following the minutes component of a TimeSpan.")
		return errors.New(d.String())
	}

	hasNsecs := s.peek(8) == '.'

	hours = s.Input[s.cursor : s.cursor+2]
	minutes = s.Input[s.cursor+3 : s.cursor+5]
	seconds = s.Input[s.cursor+6 : s.cursor+8]

	if hasNsecs {
		if !isDigit(s.peek(9)) || !isDigit(s.peek(10)) || !isDigit(s.peek(11)) {
			line, loc, ln := s.getLine(s.cursor + 9)
			var d decorator.Decorator
			d.AddLine(line, decorator.LineMetadata{LineNumber: ln, FileName: s.FileName})
			d.AddBottomComment(0, loc, "Expected exactly 3 digits for the nsecs portion of a TimeSpan.")
			return errors.New(d.String())
		}
		nsecs = s.Input[s.cursor+9 : s.cursor+12]
		s.advance(12)
	} else {
		s.advance(8)
	}

	daysn, _ := strconv.Atoi(days)
	hoursn, _ := strconv.Atoi(hours)
	minsn, _ := strconv.Atoi(minutes)
	secondsn, _ := strconv.Atoi(seconds)
	nsecsn, _ := strconv.Atoi(nsecs)

	isNegative = isNegative || daysn < 0
	if isNegative {
		daysn *= -1
	}

	s.t = timeSpan
	s.timeSpan = (time.Hour * time.Duration(24) * time.Duration(daysn)) +
		(time.Hour * time.Duration(hoursn)) +
		(time.Minute * time.Duration(minsn)) +
		(time.Second * time.Duration(secondsn)) +
		(time.Millisecond * time.Duration(nsecsn))
	if isNegative {
		s.timeSpan *= -1
	}

	return nil
}

func (s *SaxParser) nextDate() error {
	if s.cursor+10 > len(s.Input) {
		line, loc, ln := s.getLine(s.cursor)
		var d decorator.Decorator
		d.AddLine(line, decorator.LineMetadata{FileName: s.FileName, LineNumber: ln})
		d.AddBottomComment(0, loc, "Found what looks like a Date, but there's not enough characters to make a Date.")
		return errors.New(d.String())
	}

	if s.peek(7) != '/' {
		line, loc, ln := s.getLine(s.cursor + 7)
		var d decorator.Decorator
		d.AddLine(line, decorator.LineMetadata{FileName: s.FileName, LineNumber: ln})
		d.AddBottomComment(0, loc, "Expected a '/'")
		return errors.New(d.String())
	}

	year, yerr := strconv.Atoi(s.Input[s.cursor : s.cursor+4])
	month, merr := strconv.Atoi(s.Input[s.cursor+5 : s.cursor+7])
	day, derr := strconv.Atoi(s.Input[s.cursor+8 : s.cursor+10])

	if yerr != nil || merr != nil || derr != nil {
		line, loc, ln := s.getLine(s.cursor)
		var d decorator.Decorator
		d.AddLine(line, decorator.LineMetadata{FileName: s.FileName, LineNumber: ln})

		if yerr != nil {
			d.AddBottomComment(0, loc, "Invalid number")
		}
		if merr != nil {
			d.AddBottomComment(0, loc+5, "Invalid number")
		}
		if derr != nil {
			d.AddBottomComment(0, loc+8, "Invalid number")
		}
		return errors.New(d.String())
	}

	s.dateTime = time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	s.t = date
	s.advance(10)
	return nil
}

func (s *SaxParser) nextDateTime() error {
	err := s.nextDate()
	if err != nil {
		return err
	}

	s.eatWhite()
	if s.cursor+8 > len(s.Input) {
		line, loc, ln := s.getLine(s.cursor)
		var d decorator.Decorator
		d.AddLine(line, decorator.LineMetadata{FileName: s.FileName, LineNumber: ln})
		d.AddBottomComment(0, loc, "Found what looks like a DateTime, but there's not enough characters to make a DateTime.")
		return errors.New(d.String())
	}

	if s.peek(5) != ':' {
		line, loc, ln := s.getLine(s.cursor + 5)
		var d decorator.Decorator
		d.AddLine(line, decorator.LineMetadata{FileName: s.FileName, LineNumber: ln})
		d.AddBottomComment(0, loc, "Expected a ':'.")
		return errors.New(d.String())
	}

	hours, herr := strconv.Atoi(s.Input[s.cursor : s.cursor+2])
	minutes, merr := strconv.Atoi(s.Input[s.cursor+3 : s.cursor+5])
	seconds, serr := strconv.Atoi(s.Input[s.cursor+6 : s.cursor+8])
	frac := 0

	if herr != nil || merr != nil || serr != nil {
		line, loc, ln := s.getLine(s.cursor)
		var d decorator.Decorator
		d.AddLine(line, decorator.LineMetadata{FileName: s.FileName, LineNumber: ln})

		if herr != nil {
			d.AddBottomComment(0, loc, "Invalid number")
		}
		if merr != nil {
			d.AddBottomComment(0, loc+3, "Invalid number")
		}
		if serr != nil {
			d.AddBottomComment(0, loc+6, "Invalid number")
		}
		return errors.New(d.String())
	}

	s.advance(8)
	if s.peek(0) == '.' {
		if s.cursor+4 > len(s.Input) {
			line, loc, ln := s.getLine(s.cursor)
			var d decorator.Decorator
			d.AddLine(line, decorator.LineMetadata{FileName: s.FileName, LineNumber: ln})
			d.AddBottomComment(0, loc, "Found what looks like the fractional part of a DateTime, but there's not enough characters.")
			return errors.New(d.String())
		}

		var ferr error
		frac, ferr = strconv.Atoi(s.Input[s.cursor+1 : s.cursor+4])
		s.advance(4)

		if ferr != nil {
			line, loc, ln := s.getLine(s.cursor - 3)
			var d decorator.Decorator
			d.AddLine(line, decorator.LineMetadata{FileName: s.FileName, LineNumber: ln})
			d.AddBottomComment(0, loc, "Invalid number")
			return errors.New(d.String())
		}
	}

	s.t = dateTime
	s.dateTime = time.Date(s.dateTime.Year(), s.dateTime.Month(), s.dateTime.Day(), hours, minutes, seconds, frac, time.UTC)

	return nil
}

func isIdentifierStart(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_'
}

func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}

func isIdentifierContinue(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') ||
		(ch >= 'A' && ch <= 'Z') ||
		ch == '_' ||
		(ch >= '0' && ch <= '9') ||
		ch == '-' ||
		ch == '.' ||
		ch == '$'
}
